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
	if err := db.AutoMigrate(&models.User{}, &models.Device{}); err != nil {
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
		redisRuntime.Client,
		bus,
		presence,
		repository.NewDeviceRepository(db),
	)
	watcher.scanInterval = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	watcher.Start(ctx)

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
		t.Fatal("timed out waiting for fallback offline event")
	}
}
