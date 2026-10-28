package service

import (
	"context"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
)

// SessionRevokeCommandStore.ClearForSession says, in its own doc, why a command
// must not outlive its session: "leaving the row behind would have the delivery
// loop re-examining it forever." Close obeys that. CloseOrphanedSessions did
// not.
//
// The orphan sweep is the path that closes sessions whose instance died, and
// those are exactly the sessions a revoke was most likely to be pending for - the
// instance that held the socket is gone, so the command never got delivered. The
// sweep closed the session row and left the command behind, and Pending loads
// every undelivered command on every pass with no limit, so each orphan added a
// row that was re-read forever and never acted on.
func TestCloseOrphanedSessionsClearsPendingRevokeCommands(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "orphan-command.db"),
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
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.RemoteSession{},
		&models.SessionRevokeCommand{},
		&models.EventOutbox{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	ctx := context.Background()
	svc := NewRemoteSessionService(db)

	// A session held by an instance that is no longer alive, plus the revoke
	// command that was written for it and never delivered.
	session := models.RemoteSession{
		SessionID:  "sig_orphaned_with_command",
		DeviceID:   "700000101",
		InstanceID: "inst-dead",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	command := models.SessionRevokeCommand{
		SessionID:  session.SessionID,
		InstanceID: "inst-dead",
		Reason:     "owner_kick",
	}
	if err := db.Create(&command).Error; err != nil {
		t.Fatalf("seed revoke command: %v", err)
	}

	closed, err := svc.CloseOrphanedSessions(ctx, []string{"inst-alive"}, models.RemoteSessionEndServerRestart)
	if err != nil {
		t.Fatalf("CloseOrphanedSessions(): %v", err)
	}
	if closed != 1 {
		t.Fatalf("closed %d sessions, want 1", closed)
	}

	var remaining int64
	if err := db.Model(&models.SessionRevokeCommand{}).
		Where("session_id = ?", session.SessionID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("count revoke commands: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("revoke command rows = %d, want 0 - a command for an ended session is re-examined on every Pending pass", remaining)
	}
}

// The command for a session the sweep does NOT close must stay: the sweep must
// not become a way to silently drop a revoke that is still owed.
func TestCloseOrphanedSessionsKeepsCommandsForLiveSessions(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "live-command.db"),
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
	if err := db.AutoMigrate(
		&models.RemoteSession{},
		&models.SessionRevokeCommand{},
		&models.EventOutbox{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	ctx := context.Background()
	svc := NewRemoteSessionService(db)

	session := models.RemoteSession{
		SessionID:  "sig_live_with_command",
		DeviceID:   "700000102",
		InstanceID: "inst-alive",
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&models.SessionRevokeCommand{
		SessionID:  session.SessionID,
		InstanceID: "inst-alive",
		Reason:     "owner_kick",
	}).Error; err != nil {
		t.Fatalf("seed revoke command: %v", err)
	}

	if _, err := svc.CloseOrphanedSessions(ctx, []string{"inst-alive"}, models.RemoteSessionEndServerRestart); err != nil {
		t.Fatalf("CloseOrphanedSessions(): %v", err)
	}

	var remaining int64
	if err := db.Model(&models.SessionRevokeCommand{}).
		Where("session_id = ?", session.SessionID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("count revoke commands: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("revoke command rows = %d, want 1 - a live session's command is still owed", remaining)
	}
}
