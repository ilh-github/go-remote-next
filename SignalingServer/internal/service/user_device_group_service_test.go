package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newUserDeviceGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run user-device-group tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
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

func seedUserDeviceGroupUser(t *testing.T, db *gorm.DB, id uint, username string) {
	t.Helper()
	clearUserDeviceGroupFixture(t, db, id)
	if err := db.Exec("DELETE FROM users WHERE id = ?", id).Error; err != nil {
		t.Fatalf("clear user: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash')",
		id, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		clearUserDeviceGroupFixture(t, db, id)
		_ = db.Exec("DELETE FROM users WHERE id = ?", id).Error
	})
}

func clearUserDeviceGroupFixture(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	for _, stmt := range []string{
		"DELETE FROM user_device_group_members WHERE group_id IN (SELECT id FROM user_device_groups WHERE user_id = ?)",
		"DELETE FROM user_device_groups WHERE user_id = ?",
		"DELETE FROM user_devices WHERE user_id = ?",
		"DELETE FROM devices WHERE user_id = ?",
	} {
		if err := db.Exec(stmt, userID).Error; err != nil {
			t.Fatalf("clear fixture for user %d: %v", userID, err)
		}
	}
}

func seedUserDeviceGroupDevice(t *testing.T, db *gorm.DB, deviceID, uuid, name string, ownerID uint) {
	t.Helper()
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	owner := ownerID
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: uuid,
		DeviceName: name,
		UserID:     &owner,
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error
	})
}

func TestUserDeviceGroupsStayOwnerScoped(t *testing.T) {
	db := newUserDeviceGroupTestDB(t)
	ctx := context.Background()
	svc := NewUserDeviceGroupService(db)

	const (
		userA = uint(991401)
		userB = uint(991402)
	)
	seedUserDeviceGroupUser(t, db, userA, "device-group-owner-a")
	seedUserDeviceGroupUser(t, db, userB, "device-group-owner-b")

	groupA, err := svc.Create(ctx, userA, UserDeviceGroupInput{Name: "Office"})
	if err != nil {
		t.Fatalf("create owner-a group: %v", err)
	}
	groupB, err := svc.Create(ctx, userB, UserDeviceGroupInput{Name: "Office"})
	if err != nil {
		t.Fatalf("create owner-b group with the same name: %v", err)
	}

	gotA, err := svc.List(ctx, userA)
	if err != nil {
		t.Fatalf("list owner-a: %v", err)
	}
	if len(gotA) != 1 || gotA[0].ID != groupA.ID {
		t.Fatalf("owner-a groups = %+v, want only %d", gotA, groupA.ID)
	}

	gotB, err := svc.List(ctx, userB)
	if err != nil {
		t.Fatalf("list owner-b: %v", err)
	}
	if len(gotB) != 1 || gotB[0].ID != groupB.ID {
		t.Fatalf("owner-b groups = %+v, want only %d", gotB, groupB.ID)
	}
}

func TestUserDeviceGroupMembersMustBelongToTheOwner(t *testing.T) {
	db := newUserDeviceGroupTestDB(t)
	ctx := context.Background()
	svc := NewUserDeviceGroupService(db)

	const (
		userA   = uint(991411)
		userB   = uint(991412)
		deviceA = "885000411"
		deviceB = "885000412"
	)
	seedUserDeviceGroupUser(t, db, userA, "device-group-member-a")
	seedUserDeviceGroupUser(t, db, userB, "device-group-member-b")
	seedUserDeviceGroupDevice(t, db, deviceA, "uuid-device-group-a", "owner-a-device", userA)
	seedUserDeviceGroupDevice(t, db, deviceB, "uuid-device-group-b", "owner-b-device", userB)

	group, err := svc.Create(ctx, userA, UserDeviceGroupInput{Name: "Owner A"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := svc.ReplaceDevices(ctx, userA, group.ID, []string{deviceA}); err != nil {
		t.Fatalf("add owner-a device: %v", err)
	}

	if _, err := svc.ReplaceDevices(ctx, userA, group.ID, []string{deviceA, deviceB}); !errors.Is(err, ErrUserDeviceNotOwned) {
		t.Fatalf("replace with another owner's device error = %v, want %v", err, ErrUserDeviceNotOwned)
	}

	groups, err := svc.List(ctx, userA)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 || len(groups[0].DeviceIDs) != 1 || groups[0].DeviceIDs[0] != deviceA {
		t.Fatalf("membership after rejected replace = %+v, want only %s", groups, deviceA)
	}
}

func TestUserDeviceGroupUpdateAndDeleteAreOwnerScoped(t *testing.T) {
	db := newUserDeviceGroupTestDB(t)
	ctx := context.Background()
	svc := NewUserDeviceGroupService(db)

	const (
		userA = uint(991415)
		userB = uint(991416)
	)
	seedUserDeviceGroupUser(t, db, userA, "device-group-update-a")
	seedUserDeviceGroupUser(t, db, userB, "device-group-update-b")

	group, err := svc.Create(ctx, userA, UserDeviceGroupInput{Name: "Lab"})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	name := "Lab and office"
	description := "updated"
	updated, err := svc.Update(ctx, userA, group.ID, UserDeviceGroupPatch{
		Name:        &name,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("update own group: %v", err)
	}
	if updated.Name != name || updated.Description != description {
		t.Fatalf("updated group = %+v", updated)
	}
	if _, err := svc.Update(ctx, userB, group.ID, UserDeviceGroupPatch{Name: &name}); !errors.Is(err, ErrUserDeviceGroupNotFound) {
		t.Fatalf("other owner update error = %v, want not found", err)
	}
	if err := svc.Delete(ctx, userB, group.ID); !errors.Is(err, ErrUserDeviceGroupNotFound) {
		t.Fatalf("other owner delete error = %v, want not found", err)
	}
	if err := svc.Delete(ctx, userA, group.ID); err != nil {
		t.Fatalf("delete own group: %v", err)
	}
	groups, err := svc.List(ctx, userA)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("groups after delete = %+v", groups)
	}
}

func TestTakeoverRemovesDeviceFromPreviousOwnersGroups(t *testing.T) {
	db := newUserDeviceGroupTestDB(t)
	ctx := context.Background()
	groups := NewUserDeviceGroupService(db)
	devices := NewDeviceService(repository.NewDeviceRepository(db), NewDeviceSecretService())

	const (
		oldOwner = uint(991421)
		newOwner = uint(991422)
		deviceID = "885000421"
	)
	seedUserDeviceGroupUser(t, db, oldOwner, "device-group-old-owner")
	seedUserDeviceGroupUser(t, db, newOwner, "device-group-new-owner")
	seedUserDeviceGroupDevice(t, db, deviceID, "uuid-device-group-takeover", "takeover-device", oldOwner)

	group, err := groups.Create(ctx, oldOwner, UserDeviceGroupInput{Name: "Old owner"})
	if err != nil {
		t.Fatalf("create old-owner group: %v", err)
	}
	if _, err := groups.ReplaceDevices(ctx, oldOwner, group.ID, []string{deviceID}); err != nil {
		t.Fatalf("add device before takeover: %v", err)
	}
	if _, err := devices.BindToUser(ctx, deviceID, newOwner); err != nil {
		t.Fatalf("take over device: %v", err)
	}

	got, err := groups.List(ctx, oldOwner)
	if err != nil {
		t.Fatalf("list old-owner groups: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("old-owner groups = %+v, want one group", got)
	}
	if len(got[0].DeviceIDs) != 0 {
		t.Fatalf("old-owner group still contains %v after takeover", got[0].DeviceIDs)
	}
}
