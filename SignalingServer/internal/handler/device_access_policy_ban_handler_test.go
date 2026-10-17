package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPolicyHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run access-policy handler tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.DeviceAccessPolicy{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedPolicyHandlerDevice(t *testing.T, db *gorm.DB, deviceID string, userID uint, username string) {
	t.Helper()
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	if err := db.Exec("DELETE FROM users WHERE id = ?", userID).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash')",
		userID, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	owner := userID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-policy-ban-handler",
		DeviceName: "policy-ban-handler",
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM device_access_policies WHERE device_id = ?", deviceID).Error
		_ = db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error
		_ = db.Exec("DELETE FROM users WHERE id = ?", userID).Error
	})
}

func TestBanIPHandlerPersistsTheObservedSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPolicyHandlerTestDB(t)
	const (
		deviceID = "880000002"
		userID   = uint(990302)
	)
	seedPolicyHandlerDevice(t, db, deviceID, userID, "policy-ban-handler")
	h := NewDeviceAccessPolicyHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewDeviceAccessPolicyService(db),
		nil,
	)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "device_id", Value: deviceID}}
	c.Set("auth.user_id", userID)
	body, _ := json.Marshal(map[string]string{"source": " 203.0.113.7 "})
	c.Request = httptest.NewRequest(http.MethodPost,
		"/v1/me/devices/"+deviceID+"/access-bans", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BanIP(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		DenyIPs []string `json:"deny_ips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.DenyIPs) != 1 || response.DenyIPs[0] != "203.0.113.7" {
		t.Fatalf("deny_ips = %v", response.DenyIPs)
	}
}

func TestAdminBanIPHandlerPersistsAcrossAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPolicyHandlerTestDB(t)
	const (
		deviceID = "880000003"
		userID   = uint(990303)
		adminID  = uint(77)
	)
	seedPolicyHandlerDevice(t, db, deviceID, userID, "admin-ban-target")
	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	policies := service.NewDeviceAccessPolicyService(db)
	h := NewAdminRemoteSessionHandler(
		service.NewRemoteSessionService(db),
		service.NewSessionRecordingService(db),
		service.NewDeviceAccessService(db),
		devices,
		nil,
		nil,
		policies,
		nil,
	)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "device_id", Value: deviceID}}
	c.Set("auth.admin_id", adminID)
	body, _ := json.Marshal(map[string]string{"source": "198.51.100.9"})
	c.Request = httptest.NewRequest(http.MethodPost,
		"/v1/admin/devices/"+deviceID+"/access-bans", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BanIP(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		DenyIPs []string `json:"deny_ips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.DenyIPs) != 1 || response.DenyIPs[0] != "198.51.100.9" {
		t.Fatalf("deny_ips = %v", response.DenyIPs)
	}
}
