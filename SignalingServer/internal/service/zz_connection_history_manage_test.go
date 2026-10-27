package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// The connection history grows one row on every attempt, and until now nothing
// could remove one. Upstream issue #18 asks for exactly this ("delete single
// entry, clear-all history"), and it is the difference between a log an owner
// can use and a log that only accumulates.
//
// The rows are scoped by user id, so the interesting cases are the ones where a
// delete would reach past its owner: another account's row must not disappear,
// and a clear-all must not empty anybody else's history.

func newConnectionHistoryManageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openTestPostgres(t)
	applyTestMigration(t, db)
	for _, table := range []string{"connection_histories"} {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	return db
}

func seedConnectionRow(t *testing.T, db *gorm.DB, userID uint, deviceID string) models.ConnectionHistory {
	t.Helper()
	// `connection_histories.user_id` is a foreign key, so the owner has to exist
	// before a row claiming them can.
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		userID, fmt.Sprintf("qa-conn-history-%d", userID),
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	row := models.ConnectionHistory{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: "Seeded " + deviceID,
		Status:     "success",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return row
}

func TestDeleteConnectionRemovesOnlyTheNamedRowOfThatOwner(t *testing.T) {
	db := newConnectionHistoryManageDB(t)
	ctx := context.Background()
	const (
		owner      = uint(991501)
		otherOwner = uint(991502)
	)
	mine := seedConnectionRow(t, db, owner, "700000101")
	alsoMine := seedConnectionRow(t, db, owner, "700000102")
	theirs := seedConnectionRow(t, db, otherOwner, "700000103")

	svc := NewConnectionService(db)
	if err := svc.Delete(ctx, owner, mine.ID); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	if err := db.First(&models.ConnectionHistory{}, mine.ID).Error; err == nil {
		t.Fatal("the named row is still present after Delete()")
	}
	// The owner's other rows survive: a delete is one entry, not a clear.
	if err := db.First(&models.ConnectionHistory{}, alsoMine.ID).Error; err != nil {
		t.Fatalf("the owner's other row disappeared: %v", err)
	}
	// And somebody else's row is untouched.
	if err := db.First(&models.ConnectionHistory{}, theirs.ID).Error; err != nil {
		t.Fatalf("another owner's row disappeared: %v", err)
	}
}

func TestDeleteConnectionRefusesAnotherOwnersRow(t *testing.T) {
	db := newConnectionHistoryManageDB(t)
	ctx := context.Background()
	theirs := seedConnectionRow(t, db, uint(991503), "700000104")

	svc := NewConnectionService(db)
	err := svc.Delete(ctx, uint(991504), theirs.ID)
	if err == nil {
		t.Fatal("Delete() on another owner's row succeeded; the id alone must not be authority")
	}
	if !errors.Is(err, ErrConnectionHistoryNotFound) {
		t.Fatalf("Delete() error = %v, want ErrConnectionHistoryNotFound", err)
	}
	if err := db.First(&models.ConnectionHistory{}, theirs.ID).Error; err != nil {
		t.Fatalf("the row was removed anyway: %v", err)
	}
}

func TestClearConnectionsRemovesOnlyThatOwnersHistory(t *testing.T) {
	db := newConnectionHistoryManageDB(t)
	ctx := context.Background()
	const (
		owner      = uint(991505)
		otherOwner = uint(991506)
	)
	seedConnectionRow(t, db, owner, "700000105")
	seedConnectionRow(t, db, owner, "700000106")
	theirs := seedConnectionRow(t, db, otherOwner, "700000107")

	svc := NewConnectionService(db)
	removed, err := svc.Clear(ctx, owner)
	if err != nil {
		t.Fatalf("Clear(): %v", err)
	}
	if removed != 2 {
		t.Fatalf("Clear() removed %d rows, want 2", removed)
	}

	var mineLeft int64
	if err := db.Model(&models.ConnectionHistory{}).Where("user_id = ?", owner).Count(&mineLeft).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if mineLeft != 0 {
		t.Fatalf("%d of the owner's rows survived Clear()", mineLeft)
	}
	if err := db.First(&models.ConnectionHistory{}, theirs.ID).Error; err != nil {
		t.Fatalf("another owner's row disappeared: %v", err)
	}
}

// Clearing an already-empty history is not an error: the button has to stay
// usable after the first press, and a stale page must not 500 on it.
func TestClearConnectionsOnAnEmptyHistorySucceeds(t *testing.T) {
	db := newConnectionHistoryManageDB(t)
	svc := NewConnectionService(db)
	removed, err := svc.Clear(context.Background(), uint(991507))
	if err != nil {
		t.Fatalf("Clear() on an empty history: %v", err)
	}
	if removed != 0 {
		t.Fatalf("Clear() reported %d removed, want 0", removed)
	}
}
