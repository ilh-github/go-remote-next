package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
)

// The account-session revocation producers (password change, password reset,
// family break, admin force logout, scheduled sign-out) all revoke token
// families in Redis and then publish `session.revoked`. The revocation cannot
// share a transaction with the event because it lives in Redis, but the
// publish itself must survive a crash: without the outbox a failed publish is
// lost forever and the connected events WebSocket never learns it was kicked.
func TestSessionRevokedPublishIsDurableInOutbox(t *testing.T) {
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "session-revoked.db"),
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
	if err := db.AutoMigrate(&models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	redisRuntime, err := database.InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime(): %v", err)
	}
	t.Cleanup(func() { _ = redisRuntime.Close() })
	bus := NewEventBus(redisRuntime.Client)

	publisher := NewSessionRevokedEmitter(db, bus).WithOutbox(NewEventOutbox(db, bus))
	if err := publisher.Publish(context.Background(), 4242, map[string]interface{}{
		"reason": "password_changed",
	}); err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	var rows []models.EventOutbox
	if err := db.Where("event_type = ?", string(EventSessionRevoked)).Find(&rows).Error; err != nil {
		t.Fatalf("query event_outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("event_outbox rows = %d, want 1", len(rows))
	}
	if rows[0].UserID != 4242 {
		t.Fatalf("outbox user_id = %d, want 4242", rows[0].UserID)
	}
}

// Without an outbox the publisher must keep the historical direct publish, so
// a deployment that never configured one does not silently stop emitting the
// event that kicks connected WebSockets.
func TestSessionRevokedPublishFallsBackToDirectPublish(t *testing.T) {
	redisRuntime, err := database.InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime(): %v", err)
	}
	t.Cleanup(func() { _ = redisRuntime.Close() })

	bus := NewEventBus(redisRuntime.Client)
	seen := make(chan Event, 1)
	bus.Subscribe(&funcSubscriber{fn: func(evt Event) { seen <- evt }})

	publisher := NewSessionRevokedEmitter(nil, bus)
	if err := publisher.Publish(context.Background(), 77, map[string]interface{}{
		"reason": "password_changed",
	}); err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	select {
	case evt := <-seen:
		if evt.Type != EventSessionRevoked {
			t.Fatalf("event type = %q, want %q", evt.Type, EventSessionRevoked)
		}
		if evt.UserID != 77 {
			t.Fatalf("event user = %d, want 77", evt.UserID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for direct publish")
	}
}

// A nil publisher is a no-op rather than a panic: handlers constructed without
// the dependency must not crash the request path.
func TestSessionRevokedPublishNilIsNoOp(t *testing.T) {
	var publisher *SessionRevokedEmitter
	if err := publisher.Publish(context.Background(), 99, nil); err != nil {
		t.Fatalf("Publish() on nil publisher = %v, want nil", err)
	}
}

type funcSubscriber struct {
	fn func(Event)
}

func (s *funcSubscriber) Name() string { return "func-subscriber" }

func (s *funcSubscriber) HandleEvent(_ context.Context, evt Event) { s.fn(evt) }
