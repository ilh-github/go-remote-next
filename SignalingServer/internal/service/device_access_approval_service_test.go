package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestApprovalSecretHashAndMatch(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash := HashApprovalSecret(secret)
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64 hex chars", len(hash))
	}
	if !ApprovalSecretMatches(hash, secret) {
		t.Fatalf("matching secret rejected")
	}
	if ApprovalSecretMatches(hash, secret+"x") {
		t.Fatalf("wrong secret accepted")
	}
	if ApprovalSecretMatches("", secret) {
		t.Fatalf("empty stored hash accepted")
	}
}

func TestApprovalEffectiveStatusAt(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		approval models.DeviceAccessApproval
		want     string
	}{
		{"pending inside window", models.DeviceAccessApproval{Status: models.ApprovalPending, ExpiresAt: now.Add(time.Minute)}, models.ApprovalPending},
		{"pending past deadline is expired", models.DeviceAccessApproval{Status: models.ApprovalPending, ExpiresAt: now}, models.ApprovalExpired},
		// A decision is never un-made by the clock.
		{"approved stays approved past deadline", models.DeviceAccessApproval{Status: models.ApprovalApproved, ExpiresAt: now.Add(-time.Hour)}, models.ApprovalApproved},
		{"denied stays denied", models.DeviceAccessApproval{Status: models.ApprovalDenied, ExpiresAt: now.Add(-time.Hour)}, models.ApprovalDenied},
		{"consumed stays consumed", models.DeviceAccessApproval{Status: models.ApprovalConsumed, ExpiresAt: now.Add(-time.Hour)}, models.ApprovalConsumed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.approval.EffectiveStatusAt(now); got != tc.want {
				t.Fatalf("EffectiveStatusAt = %q, want %q", got, tc.want)
			}
		})
	}
	if (models.DeviceAccessApproval{Status: models.ApprovalApproved, ExpiresAt: now}).IsTerminal(now) {
		t.Fatalf("approved must still be claimable, not terminal")
	}
	if !(models.DeviceAccessApproval{Status: models.ApprovalPending, ExpiresAt: now}).IsTerminal(now) {
		t.Fatalf("expired must be terminal")
	}
}

// newApprovalTestDB opens the throwaway database and clears the approvals
// table. Like the grant tests this is skipped unless QD_TEST_DATABASE_URL
// points at a database the caller is happy to truncate.
func newApprovalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the approval tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.DeviceAccessApproval{}, &models.Device{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE device_access_approvals").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func TestApprovalDecideAndClaimExclusive(t *testing.T) {
	db := newApprovalTestDB(t)
	svc := NewDeviceAccessApprovalService(db)
	ctx := context.Background()

	row, secret, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1234567",
		OwnerID:  42,
		ClientID: "controller-1",
		ClientIP: "203.0.113.7",
		Source:   models.ApprovalSourceDeviceCode,
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if row.Status != models.ApprovalPending || row.OwnerID != 42 || len(row.SecretHash) != 64 {
		t.Fatalf("unexpected created row: %+v", row)
	}

	// Before any decision the client sees "pending" and a wrong secret is
	// indistinguishable from a missing id.
	if _, outcome, err := svc.Peek(ctx, "D1234567", row.ID, secret); err != nil || outcome != ClaimPending {
		t.Fatalf("peek pending = %v, err=%v", outcome, err)
	}
	if _, outcome, err := svc.Peek(ctx, "D1234567", row.ID, secret[:len(secret)-1]); err != nil || outcome != ClaimNotFound {
		t.Fatalf("peek wrong secret = %v, err=%v", outcome, err)
	}
	if _, outcome, _ := svc.Peek(ctx, "D1234567", row.ID+999, secret); outcome != ClaimNotFound {
		t.Fatalf("peek missing id = %v, want ClaimNotFound", outcome)
	}

	// Approve, then race two claimers: exactly one may win.
	if _, changed, err := svc.Decide(ctx, "D1234567", row.ID, true, 42, "on my way"); err != nil || !changed {
		t.Fatalf("decide approve: changed=%v err=%v", changed, err)
	}
	if _, changed, err := svc.Decide(ctx, "D1234567", row.ID, false, 42, "flip"); err != nil || changed {
		t.Fatalf("second decide must be a no-op: changed=%v err=%v", changed, err)
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		confirmed int
		consumed  int
	)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, outcome, err := svc.Claim(ctx, "D1234567", row.ID, secret)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			switch outcome {
			case ClaimConfirmed:
				confirmed++
			case ClaimConsumed:
				consumed++
			default:
				t.Errorf("unexpected claim outcome %v", outcome)
			}
		}()
	}
	wg.Wait()
	if confirmed != 1 || consumed != 1 {
		t.Fatalf("claim race: confirmed=%d consumed=%d, want 1/1", confirmed, consumed)
	}

	// The approval is single-use and now reports consumed.
	if _, outcome, _ := svc.Claim(ctx, "D1234567", row.ID, secret); outcome != ClaimConsumed {
		t.Fatalf("re-claim = %v, want ClaimConsumed", outcome)
	}
}

func TestApprovalExpiryAndListFilter(t *testing.T) {
	db := newApprovalTestDB(t)
	svc := NewDeviceAccessApprovalService(db)
	ctx := context.Background()

	row, secret, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D7654321",
		OwnerID:  7,
		Source:   models.ApprovalSourceGrant,
		TTL:      time.Millisecond,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	if _, outcome, err := svc.Peek(ctx, "D7654321", row.ID, secret); err != nil || outcome != ClaimExpired {
		t.Fatalf("peek after deadline = %v, err=%v, want ClaimExpired", outcome, err)
	}
	// Listing pending first expires the overdue row, so it lands under
	// "expired" rather than lingering as a stale pending request.
	pending, err := svc.List(ctx, "D7654321", []string{models.ApprovalPending}, 10)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending list = %d rows, want 0", len(pending))
	}
	expired, err := svc.List(ctx, "D7654321", []string{models.ApprovalExpired}, 10)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(expired) != 1 || expired[0].ID != row.ID {
		t.Fatalf("expired list = %+v, want the one overdue row", expired)
	}
}

// A correct access code is not a licence to fill the owner's decision queue:
// while a request from the same client is waiting, a second one is refused with
// a retry hint instead of becoming another entry (and another notification).
func TestApprovalRefusesSecondPendingFromSameSource(t *testing.T) {
	db := newApprovalTestDB(t)
	svc := NewDeviceAccessApprovalService(db)
	ctx := context.Background()

	first, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1111111",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.9",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, _, err = svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1111111",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.9",
		TTL:      time.Minute,
	})
	var pending *PendingApprovalError
	if !errors.As(err, &pending) {
		t.Fatalf("second create from the same source = %v, want PendingApprovalError", err)
	}
	if pending.Reason != ApprovalRefusedAlreadyPending {
		t.Fatalf("reason = %q, want %q", pending.Reason, ApprovalRefusedAlreadyPending)
	}
	if pending.RetryAfter <= 0 || pending.RetryAfter > time.Minute {
		t.Fatalf("RetryAfter = %s, want a positive hint inside the pending TTL", pending.RetryAfter)
	}

	var count int64
	if err := db.Model(&models.DeviceAccessApproval{}).
		Where("device_id = ?", "D1111111").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("refused request still wrote a row: %d rows", count)
	}

	// A different source is not affected by one client's pending request.
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1111111",
		OwnerID:  1,
		ClientID: "controller-2",
		ClientIP: "203.0.113.10",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("a different source must still be able to ask: %v", err)
	}

	// Once the owner decides, the same source may ask again - being refused is
	// about "already waiting", not about having asked before.
	if _, changed, err := svc.Decide(ctx, "D1111111", first.ID, false, 1, "no"); err != nil || !changed {
		t.Fatalf("decide: changed=%t err=%v", changed, err)
	}
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1111111",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.9",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("create after a decision: %v", err)
	}
}

// The device's queue has a ceiling whatever the sources, and a lapsed request
// is expired before it is counted so a stale row cannot block a real one.
func TestApprovalQueueCapPerDevice(t *testing.T) {
	db := newApprovalTestDB(t)
	svc := NewDeviceAccessApprovalService(db)
	ctx := context.Background()

	// A pending row that has already lapsed must not count towards the cap.
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D2222222",
		OwnerID:  1,
		ClientIP: "198.51.100.1",
		TTL:      time.Millisecond,
	}); err != nil {
		t.Fatalf("stale seed: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	for i := 0; i < MaxPendingApprovalsPerDevice; i++ {
		// The stale row still occupies the first source slot; the second
		// request from that same address is what proves it was expired.
		ip := "198.51.100.1"
		if i > 0 {
			ip = fmt.Sprintf("198.51.100.%d", i+10)
		}
		if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
			DeviceID: "D2222222",
			OwnerID:  1,
			ClientID: "controller",
			ClientIP: ip,
			TTL:      time.Minute,
		}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	_, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D2222222",
		OwnerID:  1,
		ClientID: "controller",
		ClientIP: "203.0.113.200",
		TTL:      time.Minute,
	})
	var pending *PendingApprovalError
	if !errors.As(err, &pending) {
		t.Fatalf("overflow create = %v, want PendingApprovalError", err)
	}
	if pending.Reason != ApprovalRefusedQueueFull {
		t.Fatalf("reason = %q, want %q", pending.Reason, ApprovalRefusedQueueFull)
	}
	if pending.RetryAfter <= 0 {
		t.Fatalf("RetryAfter = %s, want a positive hint", pending.RetryAfter)
	}

	var livePending int64
	if err := db.Model(&models.DeviceAccessApproval{}).
		Where("device_id = ? AND status = ?", "D2222222", models.ApprovalPending).
		Count(&livePending).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if livePending != MaxPendingApprovalsPerDevice {
		t.Fatalf("pending = %d, want the cap of %d", livePending, MaxPendingApprovalsPerDevice)
	}

	// A different device is unaffected by another device's full queue.
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D3333333",
		OwnerID:  1,
		ClientIP: "203.0.113.200",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("another device must not inherit the full queue: %v", err)
	}
}

// A denial has to mean something: otherwise the same source can ask again the
// moment the owner says no, and a correct access code becomes a way to keep
// paging the owner.
func TestApprovalDenialCooldownRefusesSameSource(t *testing.T) {
	db := newApprovalTestDB(t)
	const cooldown = 5 * time.Minute
	svc := NewDeviceAccessApprovalService(db).WithDenialCooldown(cooldown)
	ctx := context.Background()

	first, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D4444444",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.44",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, _, err := svc.Decide(ctx, "D4444444", first.ID, false, 1, "not now"); err != nil {
		t.Fatalf("deny: %v", err)
	}

	_, _, err = svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D4444444",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.44",
		TTL:      time.Minute,
	})
	var pending *PendingApprovalError
	if !errors.As(err, &pending) {
		t.Fatalf("re-request after a denial = %v, want PendingApprovalError", err)
	}
	if pending.Reason != ApprovalRefusedDeniedRecently {
		t.Fatalf("reason = %q, want %q", pending.Reason, ApprovalRefusedDeniedRecently)
	}
	if pending.RetryAfter <= 0 || pending.RetryAfter > cooldown {
		t.Fatalf("RetryAfter = %s, want a positive hint inside the cooldown", pending.RetryAfter)
	}

	// The window is per source: another client is not tarred with the same
	// brush, which is what keeps one stranger from locking everyone out.
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D4444444",
		OwnerID:  1,
		ClientID: "controller-2",
		ClientIP: "203.0.113.45",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("a different source must not inherit the cooldown: %v", err)
	}

	// Once the window lapses the source may ask again. Age the decision rather
	// than sleeping for the configured window.
	if err := db.Model(&models.DeviceAccessApproval{}).
		Where("id = ?", first.ID).
		Update("decided_at", time.Now().Add(-(cooldown + time.Minute))).Error; err != nil {
		t.Fatalf("age the decision: %v", err)
	}
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D4444444",
		OwnerID:  1,
		ClientID: "controller-1",
		ClientIP: "203.0.113.44",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("after the cooldown the source must be allowed again: %v", err)
	}
}

// 0 keeps the historical behaviour, so the cooldown is not a surprise for a
// deployment that does not ask for it.
func TestApprovalDenialCooldownOffByDefault(t *testing.T) {
	db := newApprovalTestDB(t)
	svc := NewDeviceAccessApprovalService(db)
	if svc.DenialCooldown() != 0 {
		t.Fatalf("default denial cooldown = %s, want 0", svc.DenialCooldown())
	}
	ctx := context.Background()

	first, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D5555555",
		OwnerID:  1,
		ClientIP: "203.0.113.55",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, _, err := svc.Decide(ctx, "D5555555", first.ID, false, 1, "no"); err != nil {
		t.Fatalf("deny: %v", err)
	}
	if _, _, err := svc.Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D5555555",
		OwnerID:  1,
		ClientIP: "203.0.113.55",
		TTL:      time.Minute,
	}); err != nil {
		t.Fatalf("with the cooldown disabled a denial must not block: %v", err)
	}
}

// The row list cannot show "one address has been refused six times this
// afternoon" without the owner counting rows, which is what the summary is for.
func TestApprovalSummaryAggregatesStatusesAndDeniedSources(t *testing.T) {
	db := newApprovalTestDB(t)
	cooldown := 5 * time.Minute
	svc := NewDeviceAccessApprovalService(db).WithDenialCooldown(cooldown)
	ctx := context.Background()
	const deviceID = "D6666666"
	now := time.Now().UTC()

	seed := func(status, ip string, created time.Time, decided *time.Time) {
		t.Helper()
		row := models.DeviceAccessApproval{
			DeviceID:  deviceID,
			OwnerID:   1,
			ClientIP:  ip,
			Source:    models.ApprovalSourceDeviceCode,
			Status:    status,
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: created,
			UpdatedAt: created,
			DecidedAt: decided,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed %s/%s: %v", status, ip, err)
		}
	}
	deniedAt := now.Add(-10 * time.Minute)
	// Three denials from one source (the abuse shape), one from another.
	seed(models.ApprovalDenied, "198.51.100.10", now.Add(-30*time.Minute), &deniedAt)
	seed(models.ApprovalDenied, "198.51.100.10", now.Add(-25*time.Minute), &deniedAt)
	seed(models.ApprovalDenied, "198.51.100.10", now.Add(-20*time.Minute), &deniedAt)
	seed(models.ApprovalDenied, "198.51.100.11", now.Add(-15*time.Minute), &deniedAt)
	// One approved, one still pending inside its TTL.
	seed(models.ApprovalApproved, "198.51.100.12", now.Add(-5*time.Minute), &deniedAt)
	seed(models.ApprovalPending, "198.51.100.13", now.Add(-time.Minute), nil)
	// Outside the window: must not be counted.
	seed(models.ApprovalDenied, "198.51.100.10", now.Add(-48*time.Hour), &deniedAt)

	summary, err := svc.Summary(ctx, deviceID, now.Add(-time.Hour), 0)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Total != 6 {
		t.Fatalf("total = %d, want 6 (the 48h-old row is outside the window)", summary.Total)
	}
	want := map[string]int64{
		models.ApprovalPending:  1,
		models.ApprovalApproved: 1,
		models.ApprovalDenied:   4,
		models.ApprovalExpired:  0,
		models.ApprovalConsumed: 0,
	}
	for status, n := range want {
		if summary.ByStatus[status] != n {
			t.Fatalf("by_status[%s] = %d, want %d (zeroes must be present too)",
				status, summary.ByStatus[status], n)
		}
	}
	if len(summary.ByStatus) != 5 {
		t.Fatalf("by_status has %d keys, want all five states", len(summary.ByStatus))
	}
	if len(summary.DeniedSources) != 2 {
		t.Fatalf("denied_sources = %+v, want two sources", summary.DeniedSources)
	}
	top := summary.DeniedSources[0]
	if top.ClientIP != "198.51.100.10" || top.Denials != 3 {
		t.Fatalf("top refused source = %+v, want 198.51.100.10 with 3 denials", top)
	}
	if top.LastDeniedAt == nil {
		t.Fatal("top refused source has no last_denied_at")
	}
	if got := summary.DenialCooldownSeconds; got != int(cooldown.Seconds()) {
		t.Fatalf("denial_cooldown_seconds = %d, want %d", got, int(cooldown.Seconds()))
	}

	// The source cap is bounded even when the caller asks for more.
	summary, err = svc.Summary(ctx, deviceID, now.Add(-time.Hour), 999)
	if err != nil {
		t.Fatalf("summary with a large limit: %v", err)
	}
	if len(summary.DeniedSources) > ApprovalSummaryMaxSources {
		t.Fatalf("returned %d sources, over the %d cap", len(summary.DeniedSources), ApprovalSummaryMaxSources)
	}

	// Another device's history must not leak into this one.
	other, err := svc.Summary(ctx, "D7777777", now.Add(-time.Hour), 0)
	if err != nil {
		t.Fatalf("summary for another device: %v", err)
	}
	if other.Total != 0 || len(other.DeniedSources) != 0 {
		t.Fatalf("another device's summary = %+v, want empty", other)
	}
}
