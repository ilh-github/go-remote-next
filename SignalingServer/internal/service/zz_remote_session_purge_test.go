package service

import (
	"context"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// QD-207's fourth stage: cleanup.
//
// Open, close and the orphan sweep are all real. What is missing is any bound on
// the table: every session that has ever been admitted keeps its row forever.
// The audit trail solved this with retention, and its config says why in as many
// words - a table with no natural terminal state only grows, so retention is the
// only thing that bounds it. `remote_sessions` is exactly that table.
//
// The rows are the interesting part, not the delete: a sweep that reached an open
// row would erase a session somebody is still in, and one that ran with no live
// instances would erase everything.

// seedEndedSession writes one finished row, old enough to be past any retention
// window the tests use.
func seedEndedSession(t *testing.T, db *gorm.DB, sessionID, deviceID string, userID uint, endedAgo time.Duration) {
	t.Helper()
	seedRemoteSessionDevice(t, db, deviceID, userID)
	ended := time.Now().UTC().Add(-endedAgo)
	started := ended.Add(-5 * time.Minute)
	row := models.RemoteSession{
		SessionID: sessionID,
		UserID:    userID,
		DeviceID:  deviceID,
		StartedAt: started,
		EndedAt:   &ended,
		EndReason: "closed",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed ended session: %v", err)
	}
}

func TestPurgeEndedSessionsRemovesOnlyRowsPastTheWindow(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	ctx := context.Background()
	const (
		userID   = uint(991601)
		deviceID = "700000401"
	)
	seedEndedSession(t, db, "sig_old", deviceID, userID, 40*24*time.Hour)
	seedEndedSession(t, db, "sig_recent", deviceID, userID, 2*24*time.Hour)

	svc := NewRemoteSessionService(db)
	removed, err := svc.PurgeEndedSessions(ctx, 30)
	if err != nil {
		t.Fatalf("PurgeEndedSessions(): %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want only the 1 row past the window", removed)
	}
	if err := db.Where("session_id = ?", "sig_old").First(&models.RemoteSession{}).Error; err == nil {
		t.Fatal("the row past the window is still present")
	}
	// Inside the window it stays: the owner's recent history is the part they
	// actually look at.
	if err := db.Where("session_id = ?", "sig_recent").First(&models.RemoteSession{}).Error; err != nil {
		t.Fatalf("the row inside the window was removed: %v", err)
	}
}

// The sweep must never touch an open session. Erasing a row for a session
// somebody is currently in would make the control plane forget it exists.
//
// SQL's three-valued logic means `ended_at < cutoff` already excludes a NULL, so
// the guard is not what saves an open row here - the assertion is still worth
// having, because it is the one that fails if the WHERE clause is ever rewritten
// to something that matches open rows (a plain `started_at < cutoff`, say).
func TestPurgeEndedSessionsLeavesOpenSessionsAlone(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	ctx := context.Background()
	const (
		userID   = uint(991602)
		deviceID = "700000402"
	)
	seedRemoteSessionDevice(t, db, deviceID, userID)
	if err := db.Create(&models.RemoteSession{
		SessionID: "sig_open",
		UserID:    userID,
		DeviceID:  deviceID,
		StartedAt: time.Now().UTC().Add(-90 * 24 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed open session: %v", err)
	}

	svc := NewRemoteSessionService(db)
	removed, err := svc.PurgeEndedSessions(ctx, 1)
	if err != nil {
		t.Fatalf("PurgeEndedSessions(): %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0; an open session must never be purged", removed)
	}
	if err := db.Where("session_id = ?", "sig_open").First(&models.RemoteSession{}).Error; err != nil {
		t.Fatalf("the open session was removed: %v", err)
	}
}

// A retention of zero means "keep forever", the same convention the audit and
// instance-log sweeps use. A caller that has not configured retention must not
// get an empty table.
func TestPurgeEndedSessionsWithNoRetentionKeepsEverything(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	const (
		userID   = uint(991603)
		deviceID = "700000403"
	)
	seedEndedSession(t, db, "sig_keep", deviceID, userID, 400*24*time.Hour)

	svc := NewRemoteSessionService(db)
	removed, err := svc.PurgeEndedSessions(context.Background(), 0)
	if err != nil {
		t.Fatalf("PurgeEndedSessions(): %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0 when retention is unset", removed)
	}
}

// The window is measured from when the session ended, not from when it started:
// a long-running session that finished yesterday is recent history, and a
// retention measured from the start would delete it.
func TestPurgeEndedSessionsMeasuresFromTheEnd(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	const (
		userID   = uint(991604)
		deviceID = "700000404"
	)
	seedRemoteSessionDevice(t, db, deviceID, userID)
	ended := time.Now().UTC().Add(-2 * time.Hour)
	started := ended.Add(-60 * 24 * time.Hour)
	if err := db.Create(&models.RemoteSession{
		SessionID: "sig_long",
		UserID:    userID,
		DeviceID:  deviceID,
		StartedAt: started,
		EndedAt:   &ended,
		EndReason: "closed",
	}).Error; err != nil {
		t.Fatalf("seed long session: %v", err)
	}

	svc := NewRemoteSessionService(db)
	removed, err := svc.PurgeEndedSessions(context.Background(), 30)
	if err != nil {
		t.Fatalf("PurgeEndedSessions(): %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0; a session that ran long but ended recently is recent history", removed)
	}
}

// The failure this actually guards against: a sweep written against `started_at`
// deletes exactly the rows an owner is most likely to look up - the long-running
// sessions - while leaving yesterday's short ones. That is why the window is
// measured from the end, and why the test seeds a row that would be deleted by
// the wrong column.
func TestPurgeEndedSessionsDoesNotMeasureFromTheStart(t *testing.T) {
	db := newRemoteSessionTestDB(t)
	const (
		userID   = uint(991605)
		deviceID = "700000405"
	)
	seedRemoteSessionDevice(t, db, deviceID, userID)
	ended := time.Now().UTC().Add(-3 * 24 * time.Hour)
	started := ended.Add(-120 * 24 * time.Hour)
	if err := db.Create(&models.RemoteSession{
		SessionID: "sig_marathon",
		UserID:    userID,
		DeviceID:  deviceID,
		StartedAt: started,
		EndedAt:   &ended,
		EndReason: "closed",
	}).Error; err != nil {
		t.Fatalf("seed marathon session: %v", err)
	}

	svc := NewRemoteSessionService(db)
	removed, err := svc.PurgeEndedSessions(context.Background(), 30)
	if err != nil {
		t.Fatalf("PurgeEndedSessions(): %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d; a four-month session that ended three days ago must survive a 30-day window", removed)
	}
}
