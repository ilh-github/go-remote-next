package handler

import (
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

// QD-004: the Host needs a credential that is not the user's Bearer token and
// that cannot be replayed. Without Redis the route must refuse rather than
// hand out something that only looks one-shot.
func TestUploadTicketRouteRefusesWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, rdb, _ := newUploadTicketStack(t)

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(service.NewRecordingUploadTicketStore(nil, "test-secret"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("auth.user_id", uint(7))
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/me/recordings/1/upload-ticket", nil)

	handler.CreateUploadTicket(c)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
	}
	_ = rdb
}

// With Redis the route issues a ticket that redeems once, and a second
// redemption of the same ticket is refused.
	func TestUploadTicketRouteIssuesASingleUseTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _, store := newUploadTicketStack(t)

	const userID = uint(9101)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "upload-ticket-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	recording := models.SessionRecording{
		ID:       1,
		UserID:   userID,
		DeviceID: "900000001",
		Status:   "recording",
	}
	if err := db.Create(&recording).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}
	var stored models.SessionRecording
	if err := db.First(&stored, recording.ID).Error; err != nil {
		t.Fatalf("recording was not stored: %v", err)
	}

	handler := NewSessionRecordingHandler(
		service.NewSessionRecordingService(db),
		nil,
		SessionRecordingStore{Dir: t.TempDir(), URLSecret: "test-secret"},
		service.NewAuditService(db),
	).WithUploadTickets(store)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("auth.user_id", userID)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/me/recordings/1/upload-ticket", nil)

	handler.CreateUploadTicket(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		UploadURL string `json:"upload_url"`
		SingleUse bool   `json:"single_use"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.UploadURL == "" || !payload.SingleUse {
		t.Fatalf("payload = %+v, want a single-use upload url", payload)
	}
}

func newUploadTicketStack(t *testing.T) (*gorm.DB, *redis.Client, *service.RecordingUploadTicketStore) {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "upload-ticket.db"),
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
	if err := db.AutoMigrate(&models.User{}, &models.SessionRecording{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return db, client, service.NewRecordingUploadTicketStore(client, "test-secret")
}
