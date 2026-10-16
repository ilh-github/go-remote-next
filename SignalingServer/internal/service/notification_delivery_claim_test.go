package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestClaimDueIsExclusiveAcrossWorkers proves the outbox is safe to run from
// more than one worker (or more than one server instance): the
// `SELECT ... FOR UPDATE SKIP LOCKED` claim must hand each pending row to
// exactly one claimer, never twice.
//
// SKIP LOCKED is a Postgres feature, so the test needs a real database and is
// skipped unless QD_TEST_DATABASE_URL points at a *throwaway* database — it
// truncates notification_deliveries first.
func TestClaimDueIsExclusiveAcrossWorkers(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the claim-safety test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The database is ours alone; start from a known empty table.
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	const (
		rows    = 500
		batch   = 20
		workers = 8
	)

	now := time.Now().UTC()
	seed := make([]models.NotificationDelivery, 0, rows)
	for i := 0; i < rows; i++ {
		seed = append(seed, models.NotificationDelivery{
			ChannelID:     1,
			UserID:        1,
			EventType:     "test",
			Payload:       `{"event":"test"}`,
			Status:        DeliveryStatusPending,
			NextAttemptAt: now,
		})
	}
	if err := db.CreateInBatches(seed, 100).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewNotificationDeliveryService(db)

	var mu sync.Mutex
	seen := make(map[uint]int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				claimed, err := svc.claimDue(context.Background(), batch)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				if len(claimed) == 0 {
					return
				}
				mu.Lock()
				for _, r := range claimed {
					seen[r.ID]++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(seen) != rows {
		t.Errorf("claimed %d distinct rows, want %d", len(seen), rows)
	}
	duplicates := 0
	for _, n := range seen {
		if n != 1 {
			duplicates++
		}
	}
	if duplicates != 0 {
		t.Errorf("%d rows were claimed more than once; SKIP LOCKED is not exclusive", duplicates)
	}

	// Every row must have moved out of pending.
	var pending int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("status = ?", DeliveryStatusPending).
		Count(&pending).Error; err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pending != 0 {
		t.Errorf("%d rows still pending; a claimer gave up early", pending)
	}
}

// TestRetryRearmsFailedDelivery covers the dead-letter replay path: only a
// parked `failed` row is retryable, and replaying it resets the attempt budget
// and returns it to `pending` so the worker picks it up on its next pass.
func TestRetryRearmsFailedDelivery(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retry test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	failed := &models.NotificationDelivery{
		ChannelID: 42, UserID: 7, EventType: "test", Payload: `{"event":"test"}`,
		Status: DeliveryStatusFailed, Attempts: notificationMaxAttempts,
		NextAttemptAt: time.Now().UTC(),
	}
	if err := db.Create(failed).Error; err != nil {
		t.Fatalf("seed failed row: %v", err)
	}
	delivered := &models.NotificationDelivery{
		ChannelID: 42, UserID: 7, EventType: "test", Payload: `{"event":"test"}`,
		Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: time.Now().UTC(),
	}
	if err := db.Create(delivered).Error; err != nil {
		t.Fatalf("seed delivered row: %v", err)
	}

	row, err := svc.Retry(ctx, 42, failed.ID)
	if err != nil {
		t.Fatalf("retry failed row: %v", err)
	}
	if row.Status != DeliveryStatusPending || row.Attempts != 0 {
		t.Errorf("after retry status=%q attempts=%d, want pending/0", row.Status, row.Attempts)
	}
	var reloaded models.NotificationDelivery
	if err := db.First(&reloaded, failed.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Status != DeliveryStatusPending || reloaded.Attempts != 0 {
		t.Errorf("persisted status=%q attempts=%d, want pending/0", reloaded.Status, reloaded.Attempts)
	}

	if _, err := svc.Retry(ctx, 42, delivered.ID); !errors.Is(err, ErrDeliveryNotRetryable) {
		t.Errorf("retrying a delivered row err=%v, want ErrDeliveryNotRetryable", err)
	}
	if _, err := svc.Retry(ctx, 999, failed.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("retrying a foreign channel err=%v, want ErrRecordNotFound", err)
	}
}

// TestPurgeTerminalKeepsInFlightAndRecent protects the retention sweep: only
// finished (delivered/failed) rows past the cutoff may be deleted, and
// in-flight rows are never mistaken for history no matter how old they look.
func TestPurgeTerminalKeepsInFlightAndRecent(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retention test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	seed := []models.NotificationDelivery{
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 3, NextAttemptAt: old},
		{ChannelID: 1, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivering, Attempts: 0, NextAttemptAt: old},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Backdate the rows that should look expired; GORM stamps updated_at on
	// create, so set it explicitly afterwards.
	for i, row := range seed {
		stamp := old
		if i == 2 {
			stamp = now
		}
		if err := db.Model(&models.NotificationDelivery{}).
			Where("id = ?", row.ID).Update("updated_at", stamp).Error; err != nil {
			t.Fatalf("backdate %d: %v", row.ID, err)
		}
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("purged %d rows, want 2 (old delivered + old failed)", removed)
	}

	var remaining []models.NotificationDelivery
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	gotStatuses := make(map[string]int)
	for _, row := range remaining {
		gotStatuses[row.Status]++
	}
	if len(remaining) != 3 {
		t.Fatalf("kept %d rows, want 3 (recent delivered + old pending + old delivering)", len(remaining))
	}
	if gotStatuses[DeliveryStatusPending] != 1 || gotStatuses[DeliveryStatusDelivering] != 1 || gotStatuses[DeliveryStatusDelivered] != 1 {
		t.Errorf("kept statuses = %v, want one each of pending/delivering/delivered", gotStatuses)
	}
}

// TestRetryAllFailedRequeuesOnlyFailedRows covers the bulk replay: every
// `failed` row for the channel is re-armed, and delivered/pending rows are left
// exactly as they were.
func TestRetryAllFailedRequeuesOnlyFailedRows(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the bulk-retry test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	svc := NewNotificationDeliveryService(db)
	ctx := context.Background()

	now := time.Now().UTC()
	seed := []models.NotificationDelivery{
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 42, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 2, NextAttemptAt: now},
		{ChannelID: 7, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	requeued, err := svc.RetryAllFailed(ctx, 42)
	if err != nil {
		t.Fatalf("retry-all: %v", err)
	}
	if requeued != 3 {
		t.Errorf("requeued %d rows, want 3", requeued)
	}

	var rows []models.NotificationDelivery
	if err := db.Where("channel_id = ?", 42).Find(&rows).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	byStatus := make(map[string]int)
	for _, row := range rows {
		byStatus[row.Status]++
		if row.Status == DeliveryStatusPending && row.Attempts != 0 && row.Attempts != 2 {
			t.Errorf("re-armed row %d attempts=%d, want 0 (the pre-existing pending row keeps 2)", row.ID, row.Attempts)
		}
	}
	if byStatus[DeliveryStatusPending] != 4 || byStatus[DeliveryStatusDelivered] != 1 {
		t.Errorf("after retry-all statuses = %v, want 4 pending (3 re-armed + 1 old) and 1 delivered", byStatus)
	}

	// A different channel's failed row must be untouched.
	var other models.NotificationDelivery
	if err := db.Where("channel_id = ?", 7).First(&other).Error; err != nil {
		t.Fatalf("reload other: %v", err)
	}
	if other.Status != DeliveryStatusFailed || other.Attempts != 8 {
		t.Errorf("channel 7 row status=%q attempts=%d, want failed/8 (untouched)", other.Status, other.Attempts)
	}

	if n, err := svc.RetryAllFailed(ctx, 12345); err != nil || n != 0 {
		t.Errorf("retry-all on empty channel = (%d, %v), want (0, nil)", n, err)
	}
}

// newPurgeTestDB opens the throwaway database and truncates the outbox, the
// shared preamble for the retention tests.
func newPurgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the retention tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE notification_deliveries").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// TestPurgeTerminalArchivesBeforeDeleting checks the whole point of the archive
// path: the expired rows reach disk before they leave the database.
func TestPurgeTerminalArchivesBeforeDeleting(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	archiveDir := t.TempDir()
	svc := NewNotificationDeliveryService(db).WithArchiveDir(archiveDir)

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	seed := []models.NotificationDelivery{
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":1}`, Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: old},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":2}`, Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":3}`, Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":4}`, Status: DeliveryStatusPending, Attempts: 2, NextAttemptAt: old},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	for i, row := range seed {
		stamp := old
		if i == 2 {
			stamp = now
		}
		if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", row.ID).Update("updated_at", stamp).Error; err != nil {
			t.Fatalf("backdate %d: %v", row.ID, err)
		}
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("purged %d rows, want 2", removed)
	}

	var remaining []models.NotificationDelivery
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("kept %d rows, want 2 (recent delivered + old pending)", len(remaining))
	}
	for _, row := range remaining {
		if row.Status == DeliveryStatusDelivered && row.Payload == `{"n":1}` {
			t.Error("an expired delivered row survived")
		}
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("read archive dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("archive dir has %d files, want 1", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "notification-deliveries-") || !strings.HasSuffix(entries[0].Name(), ".jsonl") {
		t.Errorf("unexpected archive file name %q", entries[0].Name())
	}
	raw, err := os.ReadFile(filepath.Join(archiveDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("archive has %d lines, want 2", len(lines))
	}
	seen := map[string]string{}
	for _, line := range lines {
		var row models.NotificationDelivery
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("archive line is not valid JSON: %v (%q)", err, line)
		}
		seen[row.Payload] = row.Status
	}
	if seen[`{"n":1}`] != DeliveryStatusDelivered || seen[`{"n":2}`] != DeliveryStatusFailed {
		t.Errorf("archived rows = %v, want n:1 delivered and n:2 failed", seen)
	}
}

// TestPurgeTerminalKeepsRowsWhenArchiveFails is the safety property: if the
// archive cannot be written, retention must abort instead of deleting the rows.
func TestPurgeTerminalKeepsRowsWhenArchiveFails(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()

	// A regular file where the archive directory should be: MkdirAll fails.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	svc := NewNotificationDeliveryService(db).WithArchiveDir(filepath.Join(blocker, "archive"))

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -30)
	row := &models.NotificationDelivery{
		ChannelID: 5, UserID: 1, EventType: "test", Payload: `{"n":1}`,
		Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: old,
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Model(&models.NotificationDelivery{}).Where("id = ?", row.ID).Update("updated_at", old).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	removed, err := svc.PurgeTerminal(ctx, now.AddDate(0, 0, -7))
	if err == nil {
		t.Fatal("purge should fail when the archive cannot be written")
	}
	if removed != 0 {
		t.Errorf("purged %d rows despite a failed archive, want 0", removed)
	}
	var count int64
	if err := db.Model(&models.NotificationDelivery{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("kept %d rows, want the expired row to survive a failed archive", count)
	}
}

// TestCountByStatusCoversEveryState checks the admin summary: every status is
// reported, including the ones with no rows, so a dashboard never has to guess
// whether a missing key means zero or "unknown".
func TestCountByStatusCoversEveryState(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	svc := NewNotificationDeliveryService(db)
	now := time.Now().UTC()

	empty, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("count on empty table: %v", err)
	}
	for _, status := range []string{DeliveryStatusPending, DeliveryStatusDelivering, DeliveryStatusDelivered, DeliveryStatusFailed} {
		if empty[status] != 0 {
			t.Errorf("empty table %s = %d, want 0", status, empty[status])
		}
	}

	seed := []models.NotificationDelivery{
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 9, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusPending, Attempts: 0, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	counts, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	want := map[string]int64{
		DeliveryStatusPending:    1,
		DeliveryStatusDelivering: 0,
		DeliveryStatusDelivered:  2,
		DeliveryStatusFailed:     1,
	}
	for status, expected := range want {
		if counts[status] != expected {
			t.Errorf("%s = %d, want %d", status, counts[status], expected)
		}
	}
}

// TestRetryAllFailedForUserIsUserScoped covers the cross-channel replay: every
// parked delivery of one user is re-armed, and another user's backlog is
// untouched no matter which channels are involved.
func TestRetryAllFailedForUserIsUserScoped(t *testing.T) {
	db := newPurgeTestDB(t)
	ctx := context.Background()
	svc := NewNotificationDeliveryService(db)
	now := time.Now().UTC()

	seed := []models.NotificationDelivery{
		{ChannelID: 10, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 11, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
		{ChannelID: 10, UserID: 1, EventType: "test", Payload: "{}", Status: DeliveryStatusDelivered, Attempts: 1, NextAttemptAt: now},
		{ChannelID: 12, UserID: 2, EventType: "test", Payload: "{}", Status: DeliveryStatusFailed, Attempts: 8, NextAttemptAt: now},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	requeued, err := svc.RetryAllFailedForUser(ctx, 1)
	if err != nil {
		t.Fatalf("retry-all-for-user: %v", err)
	}
	if requeued != 2 {
		t.Errorf("requeued %d rows, want 2 (both of user 1's channels)", requeued)
	}

	var mine []models.NotificationDelivery
	if err := db.Where("user_id = ?", 1).Find(&mine).Error; err != nil {
		t.Fatalf("reload mine: %v", err)
	}
	pending := 0
	for _, row := range mine {
		if row.Status == DeliveryStatusPending {
			pending++
			if row.Attempts != 0 {
				t.Errorf("re-armed row %d kept attempts=%d, want 0", row.ID, row.Attempts)
			}
		}
	}
	if pending != 2 {
		t.Errorf("user 1 has %d pending rows, want 2 re-armed", pending)
	}

	var theirs models.NotificationDelivery
	if err := db.Where("user_id = ?", 2).First(&theirs).Error; err != nil {
		t.Fatalf("reload theirs: %v", err)
	}
	if theirs.Status != DeliveryStatusFailed || theirs.Attempts != 8 {
		t.Errorf("user 2 row status=%q attempts=%d, want failed/8 (untouched)", theirs.Status, theirs.Attempts)
	}

	// A zero user id would otherwise re-arm the whole table.
	if _, err := svc.RetryAllFailedForUser(ctx, 0); err == nil {
		t.Error("retry-all-for-user(0) should refuse rather than touch every user")
	}
}
