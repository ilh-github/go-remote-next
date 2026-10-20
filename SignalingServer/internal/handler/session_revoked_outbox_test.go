package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Admin force-logout revokes Redis token families and then publishes
// `session.revoked`. The revocation cannot join the admin's transaction
// because it lives in Redis, but the publish must still survive a crash:
// otherwise the connected events WebSocket keeps a session the server
// already killed.
func TestAdminRevokeSessionsEnqueuesDurableSessionRevoked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, rdb, bus, outbox := newSessionRevokedHandlerTestStack(t)
	const userID = uint(9101)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "admin-revoke-outbox",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	h := newAdminUsersHandlerForSessionRevoked(db, rdb, bus).WithSessionRevokedOutbox(outbox)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "9101"}}
	c.Set("auth.admin_id", uint(5))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/admin/users/9101/sessions:revoke", nil)

	h.RevokeSessions(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	assertSessionRevokedOutboxRow(t, db, userID, "admin_forced")
}

// Without an outbox the handler keeps publishing directly, so a deployment
// that never configured one still kicks connected clients.
func TestAdminRevokeSessionsFallsBackToDirectPublish(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, rdb, bus, _ := newSessionRevokedHandlerTestStack(t)
	const userID = uint(9102)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "admin-revoke-direct",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	seen := make(chan service.Event, 1)
	bus.Subscribe(&handlerEventCapture{events: seen})

	h := newAdminUsersHandlerForSessionRevoked(db, rdb, bus)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "9102"}}
	c.Set("auth.admin_id", uint(5))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/admin/users/9102/sessions:revoke", nil)

	h.RevokeSessions(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	select {
	case evt := <-seen:
		if evt.Type != service.EventSessionRevoked || evt.UserID != userID {
			t.Fatalf("event = %+v, want session.revoked for user %d", evt, userID)
		}
	default:
		t.Fatal("no direct session.revoked publish was observed")
	}
}

func newAdminUsersHandlerForSessionRevoked(
	db *gorm.DB, rdb *redis.Client, bus *service.EventBus,
) *AdminUsersHandler {
	return NewAdminUsersHandler(
		service.NewUserService(db, service.UserServiceDeps{}),
		service.NewTokenService(rdb),
		bus,
		service.NewAuditService(db),
		service.NewPresenceService(rdb, "test-instance"),
		db,
	)
}

func newSessionRevokedHandlerTestStack(t *testing.T) (*gorm.DB, *redis.Client, *service.EventBus, *service.EventOutbox) {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "session-revoked-handler.db"),
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
	if err := db.AutoMigrate(&models.User{}, &models.AuditLog{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bus := service.NewEventBus(client)
	return db, client, bus, service.NewEventOutbox(db, bus)
}

func assertSessionRevokedOutboxRow(t *testing.T, db *gorm.DB, userID uint, reason string) {
	t.Helper()
	var rows []models.EventOutbox
	if err := db.Where("event_type = ?", string(service.EventSessionRevoked)).Find(&rows).Error; err != nil {
		t.Fatalf("query event_outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("event_outbox rows = %d, want 1", len(rows))
	}
	if rows[0].UserID != userID {
		t.Fatalf("outbox user_id = %d, want %d", rows[0].UserID, userID)
	}
	var event service.Event
	if err := json.Unmarshal([]byte(rows[0].Payload), &event); err != nil {
		t.Fatalf("decode outbox event: %v", err)
	}
	if event.Data["reason"] != reason {
		t.Fatalf("event reason = %v, want %q", event.Data["reason"], reason)
	}
}

type handlerEventCapture struct {
	events chan service.Event
}

func (s *handlerEventCapture) Name() string { return "test-capture" }

func (s *handlerEventCapture) HandleEvent(_ context.Context, evt service.Event) {
	select {
	case s.events <- evt:
	default:
	}
}
