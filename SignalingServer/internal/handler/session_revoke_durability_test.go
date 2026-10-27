package handler

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// sqliteOpen is the DSN shape the other handler tests use: a shared-cache file
// database so the same handle sees what a reopen of the path sees.
func sqliteOpen(path string) gorm.Dialector {
	return sqlite.Open("file:" + filepath.ToSlash(path) + "?cache=shared")
}

// QD-210's remaining gap, in one sentence: a revoke command that never leaves
// the process that read the row. The publish is fire-and-forget, so a process
// that dies between reading the session row and publishing to the owning
// instance's channel leaves nothing behind - the command is simply gone.
//
// The sweep closes the row about a minute later, so nothing is reported as a
// success that did not happen. But "about a minute later" is a real window in
// which a kicked controller still holds the socket, and the only thing that
// eventually acts is the presence lease expiring - which is a statement about
// the *instance* dying, not about this revoke.
//
// These tests pin the shape of the gap so the fix has something to fail against.

func newRevokeDurabilityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openRevokeDurabilitySQLite(t)
	if err := db.AutoMigrate(&models.RemoteSession{}, &models.SessionRevokeCommand{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func openRevokeDurabilitySQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqliteOpen(filepath.Join(t.TempDir(), "revoke-durability.db")))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// A command that was recorded but never delivered must be re-runnable, not just
// remembered. This is the assertion that distinguishes a durable queue from a
// log: the row has to carry enough to act on (which instance, which session) and
// has to be claimable more than once if the first attempt died too.
func TestRecordedRevokeCommandCarriesWhatIsNeededToDeliverIt(t *testing.T) {
	db := newRevokeDurabilityTestDB(t)
	store := service.NewSessionRevokeCommandStore(db)
	ctx := context.Background()

	row := models.RemoteSession{
		SessionID:  "sig_durable",
		DeviceID:   "123456789",
		UserID:     7,
		InstanceID: "inst-far",
		StartedAt:  time.Now().UTC(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	cmd, err := store.Record(ctx, "sig_durable", "inst-far", "owner_revoke")
	if err != nil {
		t.Fatalf("Record(): %v", err)
	}
	if cmd.SessionID != "sig_durable" {
		t.Fatalf("recorded session = %q, want sig_durable", cmd.SessionID)
	}
	if cmd.InstanceID != "inst-far" {
		t.Fatalf("recorded instance = %q, want inst-far - an unaddressed command cannot be delivered", cmd.InstanceID)
	}

	pending, err := store.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending(): %v", err)
	}
	if len(pending) != 1 || pending[0].ID != cmd.ID {
		t.Fatalf("pending = %+v, want the one recorded command", pending)
	}
}

// The command must survive the process that recorded it. That is the whole point
// of writing it to the database rather than to a channel: a row in Postgres
// outlives the writer, and a message on a Redis channel does not.
func TestRecordedRevokeCommandSurvivesItsWriter(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "revoke-durability.db")
	db, err := gorm.Open(sqliteOpen(dbPath))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Both tables, because Pending joins the session row to decide whether the
	// command is still owed.
	if err := db.AutoMigrate(&models.SessionRevokeCommand{}, &models.RemoteSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := service.NewSessionRevokeCommandStore(db)
	ctx := context.Background()
	// The session row too, because a command is only owed while its session is
	// open - which is the other half of what makes a stale command harmless.
	if err := db.Create(&models.RemoteSession{
		SessionID: "sig_survive",
		DeviceID:  "123456789",
		UserID:    7,
		StartedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := store.Record(ctx, "sig_survive", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	reopened, err := gorm.Open(sqliteOpen(dbPath))
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer func() {
		if sqlDB, err := reopened.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()
	pending, err := service.NewSessionRevokeCommandStore(reopened).Pending(ctx)
	if err != nil {
		t.Fatalf("Pending() after reopen: %v", err)
	}
	if len(pending) != 1 || pending[0].SessionID != "sig_survive" {
		t.Fatalf("pending after reopen = %+v, want the command recorded before the writer exited", pending)
	}
}

// Delivery is a state change, not a delete-on-read: a command that was delivered
// but whose socket close was never confirmed must still be findable, otherwise
// the retry loop would have no way to tell "not attempted" from "attempted and
// lost".
func TestDeliveredRevokeCommandIsDistinguishableFromAnUnattemptedOne(t *testing.T) {
	db := newRevokeDurabilityTestDB(t)
	store := service.NewSessionRevokeCommandStore(db)
	ctx := context.Background()

	cmd, err := store.Record(ctx, "sig_states", "inst-far", "owner_revoke")
	if err != nil {
		t.Fatalf("Record(): %v", err)
	}
	if err := store.MarkDelivered(ctx, cmd.ID); err != nil {
		t.Fatalf("MarkDelivered(): %v", err)
	}

	pending, err := store.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending(): %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending = %+v, want empty after delivery", pending)
	}
	// The row is still there, in a state that says it was attempted.
	all, err := store.All(ctx)
	if err != nil {
		t.Fatalf("All(): %v", err)
	}
	if len(all) != 1 || !all[0].Delivered {
		t.Fatalf("all = %+v, want one delivered command", all)
	}
}

// A session that ended on its own must clear its command. Otherwise every
// revoke ever issued leaves a row that a retry loop would keep re-examining, and
// "did this command do anything?" becomes unanswerable from the table alone.
func TestFinishedSessionClearsItsRevokeCommand(t *testing.T) {
	db := newRevokeDurabilityTestDB(t)
	store := service.NewSessionRevokeCommandStore(db)
	ctx := context.Background()

	cmd, err := store.Record(ctx, "sig_cleared", "inst-far", "owner_revoke")
	if err != nil {
		t.Fatalf("Record(): %v", err)
	}
	row := models.RemoteSession{
		SessionID: "sig_cleared",
		DeviceID:  "123456789",
		UserID:    7,
		StartedAt: time.Now().UTC(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := store.ClearForSession(ctx, "sig_cleared"); err != nil {
		t.Fatalf("ClearForSession(): %v", err)
	}
	all, err := store.All(ctx)
	if err != nil {
		t.Fatalf("All(): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("all = %+v, want the command cleared once the session ended", all)
	}
	if cmd.ID == 0 {
		t.Fatal("the recorded command had no id")
	}
}

// The retry loop must not act on a command whose session has already ended.
// Delivering it would close nothing and, worse, would look like the revoke worked
// when the session had ended for an unrelated reason.
func TestPendingSkipsCommandsWhoseSessionHasEnded(t *testing.T) {
	db := newRevokeDurabilityTestDB(t)
	store := service.NewSessionRevokeCommandStore(db)
	ctx := context.Background()

	if _, err := store.Record(ctx, "sig_gone", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}
	ended := time.Now().UTC()
	if err := db.Create(&models.RemoteSession{
		SessionID: "sig_gone",
		DeviceID:  "123456789",
		UserID:    7,
		StartedAt: time.Now().UTC(),
		EndedAt:   &ended,
	}).Error; err != nil {
		t.Fatalf("seed ended session: %v", err)
	}

	pending, err := store.Pending(ctx)
	if err != nil {
		t.Fatalf("Pending(): %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending = %+v, want empty - the session is already over", pending)
	}
}

// The handler-level consequence: a revoke whose publish never happened must not
// answer 200. This is the guarantee the docs already make ("it never becomes a
// silent lie"), pinned here so a refactor cannot quietly turn it into one.
func TestRevokeReportsFailureWhenTheCommandNeverLeft(t *testing.T) {
	c, rec := newRevokeTestContext(t)
	rev := &fakeRevoker{found: false, instanceID: "inst-here", askErr: errPublishNeverHappened}
	row := &models.RemoteSession{SessionID: "sig_test", Active: true, InstanceID: "inst-far"}
	if revokeSessionSocket(c, nil, rev, row) {
		t.Fatal("revokeSessionSocket reported success for a command that never left")
	}
	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500 - a command that never left is a server-side failure", rec.Code)
	}
}

var errPublishNeverHappened = errors.New("the revoke command was never published")
