package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// The service refuses the bind, but a refusal the user cannot act on is still a
// defect: the handler has to turn it into a status and a code that name the
// reason. Before this, ErrDeviceQuotaExceeded fell through to the internal-error
// branch, so the account's device limit surfaced as a 500.
func newBindQuotaHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(
		"file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "bind-quota-handler.db"))+"?cache=shared",
	), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.UserDevice{},
		&models.UserDeviceGroup{},
		&models.UserDeviceGroupMember{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedBindQuotaFixture(t *testing.T, db *gorm.DB, userID uint, quota int, deviceIDs ...string) {
	t.Helper()
	for _, deviceID := range deviceIDs {
		if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
			t.Fatalf("clear device %s: %v", deviceID, err)
		}
		if err := db.Exec(
			"INSERT INTO devices (device_id, device_uuid, device_name) VALUES (?, ?, ?)",
			deviceID, "uuid-"+deviceID, deviceID,
		).Error; err != nil {
			t.Fatalf("seed device %s: %v", deviceID, err)
		}
	}
	t.Cleanup(func() {
		for _, deviceID := range deviceIDs {
			_ = db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error
		}
		_ = db.Exec("DELETE FROM users WHERE id = ?", userID).Error
	})
	if err := db.Exec("DELETE FROM users WHERE id = ?", userID).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password, device_count) VALUES (?, ?, 'test-hash', ?)",
		userID, "bind-quota-user", quota,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// Bind asks PresenceService whether the machine is online before it touches the
// device row, so the handler needs a real Redis here: a nil client panics inside
// State() before the request ever reaches the service under test.
func newBindQuotaHandler(t *testing.T, db *gorm.DB) *DeviceHandler {
	t.Helper()
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewDeviceHandler(
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewFavoriteService(db),
		service.NewConnectionService(db),
		service.NewPresenceService(rdb, "bind-quota-test"),
		service.NewDeviceAccessService(db),
		db,
	)
}

func TestBindHandlerMapsTheQuotaRefusalToForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newBindQuotaHandlerTestDB(t)
	const (
		userID   = uint(991102)
		ownedID  = "991000101"
		targetID = "991000102"
	)
	seedBindQuotaFixture(t, db, userID, 1, ownedID, targetID)
	devices := service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService())
	if _, err := devices.BindToUser(t.Context(), ownedID, userID); err != nil {
		t.Fatalf("seed the first bind: %v", err)
	}

	h := newBindQuotaHandler(t, db)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("auth.user_id", userID)
	body, _ := json.Marshal(map[string]string{"device_id": targetID})
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/me/devices", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Bind(c)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rec.Code, rec.Body.String())
	}
	var problem struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != ProblemCodeDeviceQuotaExceeded {
		t.Fatalf("code = %q, want %q", problem.Code, ProblemCodeDeviceQuotaExceeded)
	}
	// The detail is what the user reads, so it has to say what to do rather than
	// naming an internal sentinel.
	if problem.Detail == "" {
		t.Fatal("problem detail is empty; the user would have nothing to act on")
	}
	// The refused machine must not have become a half-bound row.
	var stored models.Device
	if err := db.Where("device_id = ?", targetID).First(&stored).Error; err != nil {
		t.Fatalf("reload refused device: %v", err)
	}
	if stored.UserID != nil {
		t.Fatalf("refused device was still bound: %+v", stored)
	}
}
