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
