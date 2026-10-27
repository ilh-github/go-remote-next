package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"

	"gorm.io/gorm"
)

// QD-105 says "device quota and per-user device isolation". The isolation half
// has always been enforced (every /v1/me/devices/* route is owner-scoped), but
// `users.device_count` was a column the admin API could write and /v1/me echoed
// back while nothing anywhere read it: a user with device_count = 1 could bind an
// unlimited number of machines. These tests pin the enforcement half.

func newDeviceQuotaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "device-quota.db"),
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
	// The group tables are here because unbind and takeover both clean up
	// group membership: without them those paths fail on a missing table
	// instead of reaching the quota check the test is about.
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.UserDevice{},
		&models.UserDeviceGroup{},
		&models.UserDeviceGroupMember{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedQuotaUser(t *testing.T, db *gorm.DB, id uint, quota int) {
	t.Helper()
	username := fmt.Sprintf("quota-user-%d", id)
	if err := db.Where("id = ?", id).Delete(&models.User{}).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Where("username = ?", username).Delete(&models.User{}).Error; err != nil {
		t.Fatalf("clear stale user: %v", err)
	}
	if err := db.Create(&models.User{
		ID:          id,
		Username:    username,
		Password:    "test-hash",
		DeviceCount: quota,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("user_id = ?", id).Delete(&models.UserDeviceGroupMember{}).Error
		_ = db.Where("user_id = ?", id).Delete(&models.UserDeviceGroup{}).Error
		_ = db.Where("user_id = ?", id).Delete(&models.UserDevice{}).Error
		_ = db.Where("user_id = ?", id).Delete(&models.Device{}).Error
		_ = db.Where("id = ?", id).Delete(&models.User{}).Error
		_ = db.Where("username = ?", username).Delete(&models.User{}).Error
	})
}

func seedQuotaDevice(t *testing.T, db *gorm.DB, deviceID string, owner *uint) {
	t.Helper()
	if err := db.Where("device_id = ?", deviceID).Delete(&models.Device{}).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		DeviceName: deviceID,
		UserID:     owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("device_id = ?", deviceID).Delete(&models.Device{}).Error
	})
}

// The first device a user binds always succeeds: a quota of 0 means "unset", not
// "zero machines", and defaulting it to a hard zero would lock every newly
// registered account out of its own first machine.
func TestBindAllowsTheFirstDeviceWhenQuotaIsZero(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const userID = uint(8801)

	seedQuotaUser(t, db, userID, 0)
	seedQuotaDevice(t, db, "880000001", nil)

	if _, err := devices.BindToUser(context.Background(), "880000001", userID); err != nil {
		t.Fatalf("BindToUser(): %v", err)
	}
}

// Once the quota is full the next bind is refused, and the refusal is a named
// sentinel so the handler can map it to its own problem code instead of
// surfacing an internal error.
func TestBindRefusesADeviceOnceTheQuotaIsFull(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const userID = uint(8802)

	seedQuotaUser(t, db, userID, 1)
	seedQuotaDevice(t, db, "880000101", nil)
	seedQuotaDevice(t, db, "880000102", nil)

	if _, err := devices.BindToUser(context.Background(), "880000101", userID); err != nil {
		t.Fatalf("first BindToUser(): %v", err)
	}
	_, err := devices.BindToUser(context.Background(), "880000102", userID)
	if !errors.Is(err, ErrDeviceQuotaExceeded) {
		t.Fatalf("second BindToUser() err = %v, want ErrDeviceQuotaExceeded", err)
	}

	// The refused device must stay unowned: a half-applied bind would leave a
	// machine the user cannot see in their list but the server considers theirs.
	var stored models.Device
	if err := db.Where("device_id = ?", "880000102").First(&stored).Error; err != nil {
		t.Fatalf("reload refused device: %v", err)
	}
	if stored.UserID != nil {
		t.Fatalf("refused device was still bound: %+v", stored)
	}
}

// Re-binding a device the user already owns is not a new device. Counting it
// against the quota would make an idempotent retry (the Qt client re-binds after
// every reconnect) fail once the quota is full.
func TestBindOfAnAlreadyOwnedDeviceIgnoresTheQuota(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const userID = uint(8803)

	seedQuotaUser(t, db, userID, 1)
	seedQuotaDevice(t, db, "880000201", nil)

	for i := 0; i < 3; i++ {
		if _, err := devices.BindToUser(context.Background(), "880000201", userID); err != nil {
			t.Fatalf("BindToUser() attempt %d: %v", i, err)
		}
	}
}

// Unbinding frees the slot. Otherwise the only way back under the quota would be
// an admin editing the counter by hand, which is not something a user can do.
func TestUnbindFreesTheQuotaSlot(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const userID = uint(8804)

	seedQuotaUser(t, db, userID, 1)
	seedQuotaDevice(t, db, "880000301", nil)
	seedQuotaDevice(t, db, "880000302", nil)

	if _, err := devices.BindToUser(context.Background(), "880000301", userID); err != nil {
		t.Fatalf("first BindToUser(): %v", err)
	}
	if _, err := devices.BindToUser(context.Background(), "880000302", userID); !errors.Is(err, ErrDeviceQuotaExceeded) {
		t.Fatalf("bind over quota err = %v, want ErrDeviceQuotaExceeded", err)
	}
	if err := devices.UnbindFromUser(context.Background(), "880000301", userID); err != nil {
		t.Fatalf("UnbindFromUser(): %v", err)
	}
	if _, err := devices.BindToUser(context.Background(), "880000302", userID); err != nil {
		t.Fatalf("bind after unbind: %v", err)
	}
}

// A takeover moves a machine from one owner to another, so it must not count
// against the new owner's quota: the total number of machines bound to that user
// does not change.
func TestTakeoverDoesNotCountAgainstTheNewOwnersQuota(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const (
		oldOwner = uint(8805)
		newOwner = uint(8806)
	)

	seedQuotaUser(t, db, oldOwner, 5)
	seedQuotaUser(t, db, newOwner, 1)
	owner := oldOwner
	seedQuotaDevice(t, db, "880000401", &owner)

	if _, err := devices.BindToUser(context.Background(), "880000401", newOwner); err != nil {
		t.Fatalf("takeover BindToUser(): %v", err)
	}
}

// An admin raising the quota takes effect on the next bind, because the count is
// read inside the bind transaction rather than cached on the service.
func TestBindReadsTheQuotaFromTheDatabaseEveryTime(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())
	const userID = uint(8807)

	seedQuotaUser(t, db, userID, 1)
	seedQuotaDevice(t, db, "880000501", nil)
	seedQuotaDevice(t, db, "880000502", nil)

	if _, err := devices.BindToUser(context.Background(), "880000501", userID); err != nil {
		t.Fatalf("first BindToUser(): %v", err)
	}
	if _, err := devices.BindToUser(context.Background(), "880000502", userID); !errors.Is(err, ErrDeviceQuotaExceeded) {
		t.Fatalf("bind over quota err = %v, want ErrDeviceQuotaExceeded", err)
	}

	if err := db.Model(&models.User{}).Where("id = ?", userID).
		Update("device_count", 2).Error; err != nil {
		t.Fatalf("raise quota: %v", err)
	}
	if _, err := devices.BindToUser(context.Background(), "880000502", userID); err != nil {
		t.Fatalf("bind after the quota was raised: %v", err)
	}
}

// A quota that refuses a bind must roll the whole transaction back. The outbox
// leg is the one that would otherwise survive: an event announcing a device the
// user does not own would make every subscribed client show a machine that is
// not in its list.
func TestBindQuotaRefusalRollsBackItsEvents(t *testing.T) {
	db := newDeviceQuotaTestDB(t)
	failing := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService()).
		WithOutbox(&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")})
	const userID = uint(8808)

	seedQuotaUser(t, db, userID, 0)
	seedQuotaDevice(t, db, "880000601", nil)
	seedQuotaDevice(t, db, "880000602", nil)

	if _, err := failing.BindToUserWithEvents(
		context.Background(), "880000601", userID,
		func(_ *gorm.DB, result BindResult) ([]Event, error) {
			return []Event{{Type: EventDeviceBound, UserID: userID, DeviceID: result.Device.DeviceID}}, nil
		},
	); err == nil {
		t.Fatal("BindToUserWithEvents succeeded despite outbox failure")
	}
	var afterRollback models.Device
	if err := db.Where("device_id = ?", "880000601").First(&afterRollback).Error; err != nil {
		t.Fatalf("reload device: %v", err)
	}
	if afterRollback.UserID != nil {
		t.Fatalf("bind was not rolled back: %+v", afterRollback)
	}
}
