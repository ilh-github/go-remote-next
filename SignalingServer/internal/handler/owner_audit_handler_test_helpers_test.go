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

// ownerAuditBody is the file-transfer report shape under test, kept separate from
// the clipboard one so a call site reads as the transfer it is reporting.
type fileTransferAuditBody struct {
	DeviceID         string `json:"device_id"`
	TransferID       string `json:"transfer_id"`
	Direction        string `json:"direction"`
	Status           string `json:"status"`
	Filename         string `json:"filename"`
	BytesTransferred int64  `json:"bytes_transferred"`
	TotalBytes       int64  `json:"total_bytes"`
	Error            string `json:"error"`
}

func newOwnerAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "owner-audit.db"),
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

// ownerAuditFixture carries the read handler under test together with the two
// write handlers that produce the rows it reads, so a test can report a transfer
// and read it back through the same wiring the server uses.
type ownerAuditFixture struct {
	Handler   *OwnerAuditHandler
	Clipboard *ClipboardAuditHandler
	Files     *FileTransferAuditHandler
	ctx       *gin.Context
}

func newOwnerAuditHandler(t *testing.T, db *gorm.DB, deviceID string, userID uint) (*ownerAuditFixture, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "owner-audit-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-owner-audit-owner",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	audit := service.NewAuditService(db)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)

	return &ownerAuditFixture{
		Handler:   NewOwnerAuditHandler(audit),
		Clipboard: NewClipboardAuditHandler(devices, audit),
		Files:     NewFileTransferAuditHandler(devices, audit),
		ctx:       c,
	}, c
}

// ownerAuditRouter wires the read route the same way the server does, with the
// caller's identity injected the way the auth middleware would.
func ownerAuditRouter(t *testing.T, h *OwnerAuditHandler, userID uint) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userID)
		c.Next()
	})
	router.GET("/v1/me/audit-logs", h.List)
	return router
}

func listOwnerAudit(t *testing.T, f *ownerAuditFixture, userID uint, action string) *httptest.ResponseRecorder {
	t.Helper()
	return listOwnerAuditWithLimit(t, f, userID, action, 0)
}

func listOwnerAuditWithLimit(t *testing.T, f *ownerAuditFixture, userID uint, action string, limit int) *httptest.ResponseRecorder {
	t.Helper()
	url := "/v1/me/audit-logs"
	if action != "" {
		url += "?action=" + action
	}
	if limit > 0 {
		sep := "?"
		if action != "" {
			sep = "&"
		}
		url += sep + "limit=" + itoa(limit)
	}
	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, userID).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec
}

func listOwnerAuditFromCursor(t *testing.T, f *ownerAuditFixture, userID uint, action, cursor string) *httptest.ResponseRecorder {
	t.Helper()
	url := "/v1/me/audit-logs?cursor=" + cursor
	if action != "" {
		url += "&action=" + action
	}
	rec := httptest.NewRecorder()
	ownerAuditRouter(t, f.Handler, userID).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	return rec
}

func decodeOwnerAuditItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var page struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v (body=%s)", err, rec.Body.String())
	}
	return page.Items
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func recordFileTransferAudit(t *testing.T, h *FileTransferAuditHandler, c *gin.Context, body fileTransferAuditBody) {
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
	router.POST("/v1/me/file-transfer-audits", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.77:5555"
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("recordFileTransferAudit status = %d, want 204: %s", rec.Code, rec.Body.String())
	}
}
