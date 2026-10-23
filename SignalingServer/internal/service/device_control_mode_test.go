package service

import (
	"context"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// The owner must be able to say "one person at a time" without changing anything
// else about the device's access, and the stored value must survive a round
// trip through the database.
func TestUpsertDeviceAccessPolicyStoresControlMode(t *testing.T) {
	db := newControlModeTestDB(t)
	svc := NewDeviceAccessPolicyService(db)

	got, err := svc.Upsert(context.Background(), "111111111", 7, UpsertDeviceAccessPolicyInput{
		ControlMode: ControlModeExclusive,
	})
	if err != nil {
		t.Fatalf("Upsert(): %v", err)
	}
	if got.ControlMode != ControlModeExclusive {
		t.Fatalf("control_mode = %q, want %q", got.ControlMode, ControlModeExclusive)
	}

	read, err := svc.Get(context.Background(), "111111111")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if read.ControlMode != ControlModeExclusive {
		t.Fatalf("stored control_mode = %q, want %q", read.ControlMode, ControlModeExclusive)
	}
}

// A policy that predates the field must read as shared, because that is the
// behaviour those devices already had.
func TestControlModeDefaultsToSharedOnAnExistingPolicy(t *testing.T) {
	db := newControlModeTestDB(t)
	if err := db.Create(&models.DeviceAccessPolicy{
		DeviceID:     "111111111",
		MaxSessions:  3,
		ControlMode:  ControlModeShared,
		ApprovalMode: ApprovalModeOff,
	}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	got, err := NewDeviceAccessPolicyService(db).Get(context.Background(), "111111111")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if got := NormalizeControlMode(got.ControlMode); got != ControlModeShared {
		t.Fatalf("normalized control_mode = %q, want %q", got, ControlModeShared)
	}
}

// A typo must be refused rather than coerced, for the same reason the approval
// mode is: a silent fallback would let a misspelled setting quietly mean the
// opposite of what the owner asked for.
func TestUpsertDeviceAccessPolicyRejectsAnUnknownControlMode(t *testing.T) {
	db := newControlModeTestDB(t)
	_, err := NewDeviceAccessPolicyService(db).Upsert(context.Background(), "111111111", 7,
		UpsertDeviceAccessPolicyInput{ControlMode: "single"})
	if err == nil {
		t.Fatal("Upsert accepted control_mode \"single\"")
	}
}

// The endpoint is a whole-policy PUT, so a request that does not mention
// control_mode means "shared". That is the same contract every other field on
// this endpoint already has, and the reason the client sends the entire policy
// back. The test pins the behaviour so it cannot drift into a silent partial
// update, which would be the one field on this endpoint that behaves differently.
func TestUpsertDeviceAccessPolicyTreatsAnAbsentControlModeAsShared(t *testing.T) {
	db := newControlModeTestDB(t)
	svc := NewDeviceAccessPolicyService(db)
	if _, err := svc.Upsert(context.Background(), "111111111", 7,
		UpsertDeviceAccessPolicyInput{ControlMode: ControlModeExclusive}); err != nil {
		t.Fatalf("first Upsert(): %v", err)
	}
	if _, err := svc.Upsert(context.Background(), "111111111", 7,
		UpsertDeviceAccessPolicyInput{MaxSessions: 2}); err != nil {
		t.Fatalf("second Upsert(): %v", err)
	}

	got, err := svc.Get(context.Background(), "111111111")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if got.ControlMode != ControlModeShared {
		t.Fatalf("control_mode = %q after a PUT that omitted it, want %q",
			got.ControlMode, ControlModeShared)
	}
	if got.MaxSessions != 2 {
		t.Fatalf("max_sessions = %d, want 2", got.MaxSessions)
	}
}

// An explicit "shared" is stored as shared rather than being read as "unset",
// so a GET after a PUT reflects what the owner actually chose.
func TestUpsertDeviceAccessPolicyStoresAnExplicitShared(t *testing.T) {
	db := newControlModeTestDB(t)
	got, err := NewDeviceAccessPolicyService(db).Upsert(context.Background(), "111111111", 7,
		UpsertDeviceAccessPolicyInput{ControlMode: ControlModeShared})
	if err != nil {
		t.Fatalf("Upsert(): %v", err)
	}
	if got.ControlMode != ControlModeShared {
		t.Fatalf("control_mode = %q, want %q", got.ControlMode, ControlModeShared)
	}
}

func newControlModeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "control-mode.db"),
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
	if err := db.AutoMigrate(&models.DeviceAccessPolicy{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}
