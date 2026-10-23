package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// clipboardAuditBody is the request shape under test, named so a test reads as
// the transfer it is reporting rather than as a pile of JSON keys.
type clipboardAuditBody struct {
	DeviceID   string `json:"device_id"`
	TransferID string `json:"transfer_id"`
	Direction  string `json:"direction"`
	Status     string `json:"status"`
	MimeType   string `json:"mime_type"`
	Bytes      int64  `json:"bytes"`
}

func newClipboardAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "clipboard-audit.db"),
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
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}
	return db
}

// newClipboardAuditHandler builds the handler with the caller's device bound to
// the caller's account, and returns a gin context wired for that user.
func newClipboardAuditHandler(t *testing.T, db *gorm.DB, deviceID string, userID uint) (*ClipboardAuditHandler, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "clipboard-audit-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-clipboard-audit-owner",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewClipboardAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)
	return h, c
}

func recordClipboardAudit(t *testing.T, h *ClipboardAuditHandler, c *gin.Context, body clipboardAuditBody) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	rec := httptest.NewRecorder()
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Set("auth.user_id", c.GetUint("auth.user_id"))
		ctx.Next()
	})
	router.POST("/v1/me/clipboard-audits", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/v1/me/clipboard-audits", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.51:5555"
	router.ServeHTTP(rec, req)
	return rec
}
