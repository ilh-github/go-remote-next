package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// The scheduled "log out everywhere" action revokes the owner's token
// families in Redis and then publishes `session.revoked`. A lost publish
// leaves the connected events WebSocket holding a session the server
// already killed, so the publish has to be recoverable.
func TestScheduledSignOutPublishesDurableSessionRevoked(t *testing.T) {
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "scheduled-signout.db"),
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
	if err := db.AutoMigrate(&models.EventOutbox{}, &models.ScheduledTask{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bus := NewEventBus(client)
	outbox := NewEventOutbox(db, bus)

	action := NewSignOutAllSessionsAction(NewTokenService(client), bus, db).
		WithOutbox(outbox)
	task := models.ScheduledTask{
		ID:        4242,
		UserID:    77,
		Kind:      models.ScheduledTaskSignOutAllSessions,
		Enabled:   true,
		NextRunAt: time.Now().UTC(),
	}
	if err := action.Run(context.Background(), task); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	var rows []models.EventOutbox
	if err := db.Where("event_type = ?", string(EventSessionRevoked)).Find(&rows).Error; err != nil {
		t.Fatalf("query event_outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("event_outbox rows = %d, want 1", len(rows))
	}
	var event Event
	if err := json.Unmarshal([]byte(rows[0].Payload), &event); err != nil {
		t.Fatalf("decode outbox event: %v", err)
	}
	if event.UserID != 77 || event.Data["reason"] != "scheduled_task" {
		t.Fatalf("event = %+v, want user 77 / reason scheduled_task", event)
	}
	if event.Data["task_id"] != float64(4242) {
		t.Fatalf("event task_id = %v, want 4242", event.Data["task_id"])
	}
}

// Without an outbox the action keeps its historical direct publish, so a
// deployment that never configured one still kicks connected clients.
func TestScheduledSignOutFallsBackToDirectPublish(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bus := NewEventBus(client)
	seen := make(chan Event, 1)
	bus.Subscribe(&funcSubscriber{fn: func(evt Event) { seen <- evt }})

	action := NewSignOutAllSessionsAction(NewTokenService(client), bus, nil)
	if err := action.Run(context.Background(), models.ScheduledTask{
		ID:     9,
		UserID: 12,
		Kind:   models.ScheduledTaskSignOutAllSessions,
	}); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	select {
	case evt := <-seen:
		if evt.Type != EventSessionRevoked || evt.UserID != 12 {
			t.Fatalf("event = %+v, want session.revoked for user 12", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the direct publish")
	}
}
