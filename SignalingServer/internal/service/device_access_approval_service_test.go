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
	if err := db.AutoMigrate(&models.DeviceAccessApproval{}); err != nil {
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
