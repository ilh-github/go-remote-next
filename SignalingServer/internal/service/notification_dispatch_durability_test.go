package service

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// DispatchEvent runs on the event bus's single subscriber worker, and every
// failure inside it used to be a log line. The producers were made durable one
// by one (QD-610): they commit their event to the outbox, so a crash between
// "the business row changed" and "somebody was told" retries. But the fan-out
// itself is a plain function call on that worker, and the channel load that
// starts it talks to the same database the enqueues do.
//
// A transient error there dropped the event for every channel of that user, and
// the outbox row carrying it was already marked delivered - so nothing anywhere
// retried it. The user's webhook went quiet and the only evidence was one line
// in a log nobody was reading. Same shape as the presence gap: the durable half
// is the publish, and the half that decides who hears about it had no second
// chance at all.
func TestDispatchEventRetriesWhenTheChannelLoadFails(t *testing.T) {
	db := newNotificationDispatchDB(t)
	owner := uint(8401)
	seedNotificationOwner(t, db, owner)

	channel := models.UserNotificationChannel{
		UserID: owner,
		URL:    "https://example.invalid/hook",
		Events: jsonArray(EventDeviceOnlineChanged),
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	// Fail the first read and succeed on the second. SQLite has no SELECT
	// trigger, so the table is hidden for the first pass and restored before
	// the retry runs. The delivery table stays writable throughout, which is
	// what makes this a broken read rather than a broken database.
	if err := db.Exec("ALTER TABLE user_notification_channels RENAME TO channels_hidden").Error; err != nil {
		t.Fatalf("hide channel table: %v", err)
	}
	go restoreChannelTable(db, 250*time.Millisecond)

	deliveries := NewNotificationDeliveryService(db)
	channels := NewNotificationChannelService(db, deliveries)

	channels.DispatchEvent(context.Background(), Event{
		ID:       "evt_dispatch_retry_probe",
		Type:     EventDeviceOnlineChanged,
		UserID:   owner,
		DeviceID: "900000101",
		Data:     map[string]interface{}{"online": true},
	})

	var rows []models.NotificationDelivery
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query deliveries: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("delivery rows = %d, want 1 from the retried pass", len(rows))
	}
	if rows[0].ChannelID != channel.ID {
		t.Fatalf("delivery channel_id = %d, want %d", rows[0].ChannelID, channel.ID)
	}
	if rows[0].UserID != owner {
		t.Fatalf("delivery user_id = %d, want %d", rows[0].UserID, owner)
	}
}

// A read that never recovers must not be retried forever. The budget is three
// attempts on a worker every other subscriber shares, so the test also pins
// that the whole thing finishes in about a second rather than hanging the
// subscriber queue.
func TestDispatchEventGivesUpAfterTheRetryBudget(t *testing.T) {
	db := newNotificationDispatchDB(t)
	owner := uint(8402)
	seedNotificationOwner(t, db, owner)

	// The table stays hidden for the whole budget, so no pass can succeed.
	if err := db.Exec("ALTER TABLE user_notification_channels RENAME TO channels_hidden").Error; err != nil {
		t.Fatalf("hide channel table: %v", err)
	}

	deliveries := NewNotificationDeliveryService(db)
	channels := NewNotificationChannelService(db, deliveries)

	start := time.Now()
	channels.DispatchEvent(context.Background(), Event{
		ID:       "evt_dispatch_giveup_probe",
		Type:     EventDeviceOnlineChanged,
		UserID:   owner,
		DeviceID: "900000102",
		Data:     map[string]interface{}{"online": true},
	})
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Fatalf("DispatchEvent took %s, want the bounded retry budget to finish quickly", elapsed)
	}

	var rows []models.NotificationDelivery
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query deliveries: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("delivery rows = %d, want 0 - the read never succeeded, so there is nothing to deliver", len(rows))
	}
}

// The success path must stay exactly as it was: one row per matching channel,
// no retries, no duplicate deliveries. The retry is only allowed to change what
// happens when something fails.
func TestDispatchEventSuccessPathIsUnchanged(t *testing.T) {
	db := newNotificationDispatchDB(t)
	owner := uint(8403)
	seedNotificationOwner(t, db, owner)

	subscribed := models.UserNotificationChannel{
		UserID: owner,
		URL:    "https://example.invalid/subscribed",
		Events: jsonArray(EventDeviceOnlineChanged),
	}
	notSubscribed := models.UserNotificationChannel{
		UserID: owner,
		URL:    "https://example.invalid/other",
		Events: jsonArray(EventDeviceBound),
	}
	if err := db.Create(&subscribed).Error; err != nil {
		t.Fatalf("seed subscribed: %v", err)
	}
	if err := db.Create(&notSubscribed).Error; err != nil {
		t.Fatalf("seed not subscribed: %v", err)
	}

	deliveries := NewNotificationDeliveryService(db)
	channels := NewNotificationChannelService(db, deliveries)

	start := time.Now()
	channels.DispatchEvent(context.Background(), Event{
		ID:       "evt_success_path_probe",
		Type:     EventDeviceOnlineChanged,
		UserID:   owner,
		DeviceID: "900000103",
		Data:     map[string]interface{}{"online": true},
	})
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("success path took %s, want no retry delay on the happy path", elapsed)
	}

	var rows []models.NotificationDelivery
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query deliveries: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("delivery rows = %d, want exactly 1", len(rows))
	}
	if rows[0].ChannelID != subscribed.ID {
		t.Fatalf("delivery channel_id = %d, want %d", rows[0].ChannelID, subscribed.ID)
	}
	if rows[0].Status != DeliveryStatusPending {
		t.Fatalf("delivery status = %q, want %q", rows[0].Status, DeliveryStatusPending)
	}
}

func newNotificationDispatchDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "notification-dispatch.db"),
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
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserNotificationChannel{},
		&models.NotificationDelivery{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedNotificationOwner(t *testing.T, db *gorm.DB, owner uint) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       owner,
		Username: "dispatch-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// restoreChannelTable puts the hidden table back after a delay, so a retry that
// lands after the delay sees a readable table. It is a goroutine rather than a
// fixed sleep in the test body because the retry backoff is what is under test:
// sleeping first would make the first pass succeed and there would be nothing
// to retry.
func restoreChannelTable(db *gorm.DB, after time.Duration) {
	time.Sleep(after)
	_ = db.Exec("ALTER TABLE channels_hidden RENAME TO user_notification_channels")
}

// jsonArray builds the JSON text the Events column stores, so the seed says
// exactly what a real channel row says rather than a shape that happens to parse.
func jsonArray(values ...string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
