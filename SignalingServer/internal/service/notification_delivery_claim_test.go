package service

import (
	"context"
	"errors"
	"os"
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
