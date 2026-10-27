package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newPolicyHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openTestPostgres(t)
	// All three tables are defined by migrations/001_init.sql. AutoMigrate names
	// a unique index uni_<table>_<column> while that file writes plain `UNIQUE`
	// and lets Postgres name it, so its index comparison tries to drop a
	// constraint that was never there as soon as anything has created the table
	// first -- which the shared scratch database guarantees by the second run.
	applyTestMigration(t, db)
	return db
}

func newPolicyHandlerSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(
		"file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "policy-handler.db"))+"?cache=shared",
	), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.DeviceAccessPolicy{},
		&models.EventOutbox{},
	); err != nil {
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

func TestUnbanIPHandlerRemovesTheSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPolicyHandlerTestDB(t)
	const (
		deviceID = "880000006"
		userID   = uint(990306)
	)
	seedPolicyHandlerDevice(t, db, deviceID, userID, "policy-unban-handler")
	policies := service.NewDeviceAccessPolicyService(db)
	if _, err := policies.BanIP(context.Background(), deviceID, userID, "203.0.113.77"); err != nil {
		t.Fatalf("seed ban: %v", err)
	}
	h := NewDeviceAccessPolicyHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		policies,
		nil,
	)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "device_id", Value: deviceID}}
	c.Set("auth.user_id", userID)
	body, _ := json.Marshal(map[string]string{"source": "203.0.113.77"})
	c.Request = httptest.NewRequest(http.MethodPost,
		"/v1/me/devices/"+deviceID+"/access-bans:remove", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UnbanIP(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		DenyIPs []string `json:"deny_ips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.DenyIPs) != 0 {
		t.Fatalf("deny_ips = %v, want empty", response.DenyIPs)
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

func TestAdminUnbanIPHandlerRemovesAcrossAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPolicyHandlerTestDB(t)
	const (
		deviceID = "880000007"
		userID   = uint(990307)
		adminID  = uint(78)
	)
	seedPolicyHandlerDevice(t, db, deviceID, userID, "admin-unban-target")
	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	policies := service.NewDeviceAccessPolicyService(db)
	if _, err := policies.BanIP(context.Background(), deviceID, userID, "198.51.100.10"); err != nil {
		t.Fatalf("seed ban: %v", err)
	}
	h := NewAdminRemoteSessionHandler(
		service.NewRemoteSessionService(db),
		service.NewSessionRecordingService(db),
		service.NewDeviceAccessService(db),
		devices,
		nil,
		nil,
		policies,
	)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Params = gin.Params{{Key: "device_id", Value: deviceID}}
	c.Set("auth.admin_id", adminID)
	body, _ := json.Marshal(map[string]string{"source": "198.51.100.10"})
	c.Request = httptest.NewRequest(http.MethodPost,
		"/v1/admin/devices/"+deviceID+"/access-bans:remove", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UnbanIP(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		DenyIPs []string `json:"deny_ips"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.DenyIPs) != 0 {
		t.Fatalf("deny_ips = %v, want empty", response.DenyIPs)
	}
}

func TestAdminAccessBanAndUnbanCommitPolicyEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPolicyHandlerSQLiteTestDB(t)
	const (
		deviceID = "880000009"
		userID   = uint(990309)
		adminID  = uint(79)
	)
	seedPolicyHandlerDevice(t, db, deviceID, userID, "admin-policy-events")
	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	policies := service.NewDeviceAccessPolicyService(db).
		WithOutbox(service.NewEventOutbox(db, nil))
	h := NewAdminRemoteSessionHandler(
		service.NewRemoteSessionService(db),
		service.NewSessionRecordingService(db),
		service.NewDeviceAccessService(db),
		devices,
		nil,
		nil,
		policies,
	)

	call := func(path, source string) {
		t.Helper()
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Params = gin.Params{{Key: "device_id", Value: deviceID}}
		c.Set("auth.admin_id", adminID)
		body, _ := json.Marshal(map[string]string{"source": source})
		c.Request = httptest.NewRequest(http.MethodPost,
			"/v1/admin/devices/"+deviceID+path, bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if path == "/access-bans" {
			h.BanIP(c)
		} else {
			h.UnbanIP(c)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body = %s", path, rec.Code, rec.Body.String())
		}
	}

	call("/access-bans", "198.51.100.19")
	call("/access-bans:remove", "198.51.100.19")

	var rows []models.EventOutbox
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("outbox rows = %d, want 2", len(rows))
	}
	reasons := map[string]bool{}
	for _, row := range rows {
		var event service.Event
		if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		if event.Type != service.EventDevicePolicyChanged ||
			event.UserID != userID || event.DeviceID != deviceID {
			t.Fatalf("event routing = %+v", event)
		}
		reasons[event.Data["reason"].(string)] = true
	}
	if !reasons["administrator banned access source"] ||
		!reasons["administrator unbanned access source"] {
		t.Fatalf("event reasons = %+v", reasons)
	}
}
