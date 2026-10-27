package service

import (
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// WebhookService.Dispatch is the admin-side counterpart of the user
// notification fan-out. It has the same shape and, until now, a worse version
// of the same gap.
//
// The user fan-out at least logged its failures; this one returns silently. A
// read error while loading the enabled webhooks drops the event with no log
// line, no retry and no record - and the outbox row carrying it was already
// marked delivered by the time the subscriber ran.
//
// The three events that reach here are `session.revoked`, `turn.config.changed`
// and `device.secret.rotated`. A revoked session an operator's SIEM never hears
// about is the one that matters: the revocation happened, the audit trail has
// it, and the system that was supposed to be told was not.
func TestWebhookDispatchRetriesWhenTheLoadFails(t *testing.T) {
	db := newWebhookDispatchDB(t)
	seedWebhookSubscriber(t, db, "https://example.invalid/hook", EventSessionRevoked)

	// Hide the table for the first pass and restore it before the retry. The
	// delivery table is untouched, so this is a broken read rather than a
	// broken database.
	if err := db.Exec("ALTER TABLE webhooks RENAME TO webhooks_hidden").Error; err != nil {
		t.Fatalf("hide webhook table: %v", err)
	}
	go restoreWebhookTable(db, 250*time.Millisecond)

	svc := NewWebhookService(db)
	delivered := make(chan struct{}, 1)
	svc.WithDeliveryObserver(func() { delivered <- struct{}{} })

	svc.Dispatch(EventSessionRevoked, map[string]interface{}{"reason": "admin_forced"})

	select {
	case <-delivered:
	case <-time.After(3 * time.Second):
		t.Fatal("no delivery was attempted; the load failure was not retried")
	}
}

// A load that never recovers must not hang the subscriber worker. The budget is
// bounded and the whole call finishes in about a second.
func TestWebhookDispatchGivesUpAfterTheRetryBudget(t *testing.T) {
	db := newWebhookDispatchDB(t)
	seedWebhookSubscriber(t, db, "https://example.invalid/hook", EventSessionRevoked)

	if err := db.Exec("ALTER TABLE webhooks RENAME TO webhooks_hidden").Error; err != nil {
		t.Fatalf("hide webhook table: %v", err)
	}

	svc := NewWebhookService(db)
	attempts := make(chan struct{}, 8)
	svc.WithDeliveryObserver(func() { attempts <- struct{}{} })

	start := time.Now()
	svc.Dispatch(EventSessionRevoked, map[string]interface{}{"reason": "admin_forced"})
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Fatalf("Dispatch took %s, want the bounded retry budget to finish quickly", elapsed)
	}
	select {
	case <-attempts:
		t.Fatal("a delivery was attempted even though the webhooks were never readable")
	default:
	}
}

// The success path must stay exactly as it was. The retry is only allowed to
// change what happens when something fails.
func TestWebhookDispatchSuccessPathIsUnchanged(t *testing.T) {
	db := newWebhookDispatchDB(t)
	hook := seedWebhookSubscriber(t, db, "https://example.invalid/hook", EventSessionRevoked)
	other := seedWebhookSubscriber(t, db, "https://example.invalid/other", EventDeviceSecretRotated)

	svc := NewWebhookService(db)
	var got []string
	svc.WithDeliveryRecorder(func(url string) { got = append(got, url) })
	delivered := make(chan struct{}, 2)
	svc.WithDeliveryObserver(func() { delivered <- struct{}{} })

	start := time.Now()
	svc.Dispatch(EventSessionRevoked, map[string]interface{}{"reason": "admin_forced"})
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("success path took %s, want no retry delay on the happy path", elapsed)
	}

	// The delivery runs in its own goroutine, so the recorder is filled shortly
	// after Dispatch returns rather than during it.
	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("no delivery was attempted on the success path")
	}

	if len(got) != 1 {
		t.Fatalf("delivered to %d webhooks, want exactly 1", len(got))
	}
	if got[0] != hook.URL {
		t.Fatalf("delivered to %q, want %q (the %q webhook subscribes to a different event)",
			got[0], hook.URL, other.URL)
	}
}

func newWebhookDispatchDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "webhook-dispatch.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.Webhook{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedWebhookSubscriber(t *testing.T, db *gorm.DB, url, event string) models.Webhook {
	t.Helper()
	hook := models.Webhook{
		Name:    "probe-" + event,
		URL:     url,
		Events:  `["` + event + `"]`,
		Enabled: true,
	}
	if err := db.Create(&hook).Error; err != nil {
		t.Fatalf("seed webhook: %v", err)
	}
	return hook
}

// restoreWebhookTable puts the hidden table back after a delay, so a retry that
// lands after the delay sees a readable table. A goroutine rather than a sleep
// in the test body, because the backoff is what is under test.
func restoreWebhookTable(db *gorm.DB, after time.Duration) {
	time.Sleep(after)
	_ = db.Exec("ALTER TABLE webhooks_hidden RENAME TO webhooks")
}
