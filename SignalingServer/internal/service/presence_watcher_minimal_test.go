package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
)

type offlineEventCapture struct {
	events chan Event
}

func (c *offlineEventCapture) Name() string { return "offline-event-capture" }

func (c *offlineEventCapture) HandleEvent(_ context.Context, event Event) {
	c.events <- event
}

func TestPresenceWatcherFallbackPublishesOfflineWithoutKeyspaceEvents(t *testing.T) {
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "presence.db"),
		},
	})
	if err != nil {
		t.Fatalf("OpenDatabase(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	user := models.User{Username: "presence-owner", Password: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	device := models.Device{
		DeviceID:   "123456789",
		DeviceUUID: "presence-test-uuid",
		UserID:     &user.ID,
		AccessCode: "135790",
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	redisRuntime, err := database.InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime(): %v", err)
	}
	t.Cleanup(func() { _ = redisRuntime.Close() })

	presence := NewPresenceService(redisRuntime.Client, "minimal-test")
	if !presence.RememberOnlineCandidate(context.Background(), device.DeviceID) {
		t.Fatal("RememberOnlineCandidate() = false, want true")
	}
	bus := NewEventBus(redisRuntime.Client)
	capture := &offlineEventCapture{events: make(chan Event, 4)}
	bus.Subscribe(capture)

	watcher := NewPresenceWatcher(
		db,
		redisRuntime.Client,
		bus,
		presence,
		repository.NewDeviceRepository(db),
	)
	watcher.WithOutbox(NewEventOutbox(db, bus))
	watcher.scanInterval = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	watcher.Start(ctx)

	// Durable path: the transition is committed to the outbox before the worker
	// publishes it, so wait for the row and then drain it.
	var rows []models.EventOutbox
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Where("event_type = ?", string(EventDeviceOnlineChanged)).Find(&rows).Error; err != nil {
			t.Fatalf("query event_outbox: %v", err)
		}
		if len(rows) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(rows) != 1 {
		t.Fatalf("event_outbox rows = %d, want 1", len(rows))
	}
	if rows[0].DeviceID != device.DeviceID {
		t.Fatalf("outbox device_id = %q, want %q", rows[0].DeviceID, device.DeviceID)
	}

	if _, err := NewEventOutbox(db, bus).Drain(context.Background(), time.Now(), 10); err != nil {
		t.Fatalf("Drain(): %v", err)
	}
	select {
	case event := <-capture.events:
		if event.Type != EventDeviceOnlineChanged {
			t.Fatalf("event type = %q, want %q", event.Type, EventDeviceOnlineChanged)
		}
		if event.DeviceID != device.DeviceID {
			t.Fatalf("event device = %q, want %q", event.DeviceID, device.DeviceID)
		}
		if online, _ := event.Data["online"].(bool); online {
			t.Fatal("offline fallback published online=true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for drained offline event")
	}
}

func TestPresenceWatcherOfflineEventIsDurableInOutbox(t *testing.T) {
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "presence-outbox.db"),
		},
	})
	if err != nil {
		t.Fatalf("OpenDatabase(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	user := models.User{Username: "outbox-owner", Password: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	device := models.Device{
		DeviceID:   "987654321",
		DeviceUUID: "presence-outbox-uuid",
		UserID:     &user.ID,
		AccessCode: "135790",
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	redisRuntime, err := database.InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime(): %v", err)
	}
	t.Cleanup(func() { _ = redisRuntime.Close() })

	presence := NewPresenceService(redisRuntime.Client, "minimal-outbox-test")
	if !presence.RememberOnlineCandidate(context.Background(), device.DeviceID) {
		t.Fatal("RememberOnlineCandidate() = false, want true")
	}
	bus := NewEventBus(redisRuntime.Client)
	capture := &offlineEventCapture{events: make(chan Event, 4)}
	bus.Subscribe(capture)

	watcher := NewPresenceWatcher(
		db,
		redisRuntime.Client,
		bus,
		presence,
		repository.NewDeviceRepository(db),
	)
	watcher.WithOutbox(NewEventOutbox(db, bus))
	watcher.scanInterval = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	watcher.Start(ctx)

	var rows []models.EventOutbox
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Where("event_type = ?", string(EventDeviceOnlineChanged)).Find(&rows).Error; err != nil {
			t.Fatalf("query event_outbox: %v", err)
		}
		if len(rows) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(rows) != 1 {
		t.Fatalf("event_outbox rows = %d, want 1", len(rows))
	}

	if _, err := NewEventOutbox(db, bus).Drain(context.Background(), time.Now(), 10); err != nil {
		t.Fatalf("Drain(): %v", err)
	}
	select {
	case event := <-capture.events:
		if event.Type != EventDeviceOnlineChanged {
			t.Fatalf("event type = %q, want %q", event.Type, EventDeviceOnlineChanged)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for drained offline event")
	}
	if rows[0].UserID != user.ID {
		t.Fatalf("outbox user_id = %d, want %d", rows[0].UserID, user.ID)
	}
	if rows[0].DeviceID != device.DeviceID {
		t.Fatalf("outbox device_id = %q, want %q", rows[0].DeviceID, device.DeviceID)
	}
}
