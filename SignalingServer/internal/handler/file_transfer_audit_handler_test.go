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

func newFileTransferAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "file-transfer-audit.db"),
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

func TestFileTransferAuditHandlerRecordsOwnerUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newFileTransferAuditTestDB(t)
	const (
		userID     = uint(4101)
		deviceID   = "884104101"
		transferID = "transfer-upload-4101"
	)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "file-transfer-audit-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-file-transfer-audit-owner",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewFileTransferAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userID)
		c.Next()
	})
	router.POST("/v1/me/file-transfer-audits", h.Create)

	body, _ := json.Marshal(map[string]any{
		"device_id":         deviceID,
		"transfer_id":       transferID,
		"direction":         "upload",
		"status":            "completed",
		"filename":          "report.pdf",
		"bytes_transferred": 123,
		"total_bytes":       123,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.41:4321"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body.String())
	}

	var row models.AuditLog
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("load audit row: %v", err)
	}
	if row.ActorUserID != userID || row.Action != "file.transfer.upload" {
		t.Fatalf("actor/action = %d/%q, want %d/file.transfer.upload",
			row.ActorUserID, row.Action, userID)
	}
	if row.ResourceType != "file_transfer" || row.ResourceID != transferID {
		t.Fatalf("resource = %s/%s, want file_transfer/%s",
			row.ResourceType, row.ResourceID, transferID)
	}
	if row.IP != "203.0.113.41" {
		t.Fatalf("ip = %q, want observed client address", row.IP)
	}
	var details map[string]any
	if err := json.Unmarshal([]byte(row.Details), &details); err != nil {
		t.Fatalf("details = %q, want JSON: %v", row.Details, err)
	}
	if details["device_id"] != deviceID ||
		details["status"] != "completed" ||
		details["filename"] != "report.pdf" ||
		details["bytes_transferred"] != float64(123) ||
		details["total_bytes"] != float64(123) {
		t.Fatalf("details = %#v", details)
	}
}

func TestFileTransferAuditHandlerIsIdempotentByTransferID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newFileTransferAuditTestDB(t)
	const (
		userID     = uint(4102)
		deviceID   = "884104102"
		transferID = "transfer-download-4102"
	)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "file-transfer-audit-idempotent",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-file-transfer-audit-idempotent",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewFileTransferAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userID)
		c.Next()
	})
	router.POST("/v1/me/file-transfer-audits", h.Create)

	body, _ := json.Marshal(map[string]any{
		"device_id":         deviceID,
		"transfer_id":       transferID,
		"direction":         "download",
		"status":            "failed",
		"filename":          "archive.zip",
		"bytes_transferred": 42,
		"total_bytes":       100,
		"error":             "disk write error",
	})
	for attempt := 1; attempt <= 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("attempt %d status = %d, want 204: %s", attempt, rec.Code, rec.Body.String())
		}
	}

	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("audit rows = %d, want one row for a retried transfer report", count)
	}
}

func TestFileTransferAuditHandlerRejectsUnknownDirection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newFileTransferAuditTestDB(t)
	const (
		userID   = uint(4103)
		deviceID = "884104103"
	)
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "file-transfer-audit-validation",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-file-transfer-audit-validation",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewFileTransferAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", userID)
		c.Next()
	})
	router.POST("/v1/me/file-transfer-audits", h.Create)

	body, _ := json.Marshal(map[string]any{
		"device_id":   deviceID,
		"transfer_id": "transfer-invalid-direction",
		"direction":   "rename",
		"status":      "completed",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("audit rows = %d, want none for an invalid report", count)
	}
}

func TestFileTransferAuditHandlerRejectsDeviceOwnedByAnotherUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newFileTransferAuditTestDB(t)
	const (
		ownerID    = uint(4104)
		reporterID = uint(4105)
		deviceID   = "884104104"
	)
	for _, user := range []models.User{
		{ID: ownerID, Username: "file-transfer-audit-owner-4104", Password: "test-hash"},
		{ID: reporterID, Username: "file-transfer-audit-reporter-4105", Password: "test-hash"},
	} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("seed user %d: %v", user.ID, err)
		}
	}
	owner := ownerID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-file-transfer-audit-foreign",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}

	h := NewFileTransferAuditHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewAuditService(db),
	)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("auth.user_id", reporterID)
		c.Next()
	})
	router.POST("/v1/me/file-transfer-audits", h.Create)

	body, _ := json.Marshal(map[string]any{
		"device_id":   deviceID,
		"transfer_id": "transfer-foreign-device",
		"direction":   "download",
		"status":      "completed",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/me/file-transfer-audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
	var count int64
	if err := db.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("audit rows = %d, want none for another user's device", count)
	}
}
