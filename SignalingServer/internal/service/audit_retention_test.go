package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

func seedAuditRowAt(t *testing.T, db *gorm.DB, action string, at time.Time) {
	t.Helper()
	if err := db.Create(&models.AuditLog{
		Action:    action,
		CreatedAt: at,
	}).Error; err != nil {
		t.Fatalf("seed audit row: %v", err)
	}
}

// Retention is the only thing that bounds `audit_logs`, so the window has to
// delete the expired rows and nothing else.
func TestAuditPurgeBeforeDeletesOnlyExpiredRows(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.action", now.AddDate(0, 0, -40))
	seedAuditRowAt(t, db, "recent.action", now.AddDate(0, 0, -1))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purged %d rows, want the one outside the window", purged)
	}
	var remaining []models.AuditLog
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Action != "recent.action" {
		t.Fatalf("remaining = %+v, want only the recent row", remaining)
	}
}

// Audit rows are evidence, so an archive directory has to hold the bytes before
// the delete happens.
func TestAuditPurgeBeforeArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.one", now.AddDate(0, 0, -90))
	seedAuditRowAt(t, db, "old.two", now.AddDate(0, 0, -60))
	seedAuditRowAt(t, db, "recent.action", now.AddDate(0, 0, -1))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 2 {
		t.Fatalf("purged %d rows, want 2", purged)
	}

	name := "audit-logs-" + time.Now().UTC().Format("2006-01-02") + ".jsonl"
	body, err := os.ReadFile(filepath.Join(archiveDir, name))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(string(body)), "\n") + 1
	if lines != 2 {
		t.Fatalf("archive holds %d lines, want one per purged row:\n%s", lines, body)
	}
	if !strings.Contains(string(body), "old.one") || !strings.Contains(string(body), "old.two") {
		t.Fatalf("archive is missing a purged row:\n%s", body)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows after purge, want only the recent one", count)
	}
}

// The archive is the point: if it cannot be written, the delete must not
// happen, because otherwise retention silently becomes the step that destroyed
// the only copy.
func TestAuditPurgeBeforeArchiveFailureDeletesNothing(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()

	now := time.Now()
	seedAuditRowAt(t, db, "old.action", now.AddDate(0, 0, -90))

	// A regular file standing where the archive directory should be makes
	// MkdirAll fail, which is the "archive is unusable" case.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	svc := NewAuditService(db).WithArchiveDir(filepath.Join(blocker, "archive"))

	purged, err := svc.PurgeBefore(ctx, now.AddDate(0, 0, -30))
	if err == nil {
		t.Fatal("purge reported success even though the archive could not be written")
	}
	if purged != 0 {
		t.Fatalf("purged = %d, want 0 when the archive failed", purged)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows, want the row kept because the archive failed", count)
	}
}

// The row cap is the age-independent second bound: it must keep exactly the
// newest N rows (by id) and drop the oldest, whatever their timestamps say.
func TestAuditCapRowsKeepsNewest(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()
	svc := NewAuditService(db)

	for _, action := range []string{"cap.r1", "cap.r2", "cap.r3", "cap.r4", "cap.r5"} {
		seedAuditRowAt(t, db, action, time.Now())
	}

	removed, err := svc.CapRows(ctx, 3)
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	var remaining []models.AuditLog
	if err := db.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 3 {
		t.Fatalf("table holds %d rows, want 3", len(remaining))
	}
	if remaining[0].Action != "cap.r3" || remaining[2].Action != "cap.r5" {
		t.Fatalf("kept %q..%q, want the newest three rows",
			remaining[0].Action, remaining[2].Action)
	}
}

// A cap that is already satisfied, or a cap of 0 (disabled), must delete
// nothing - the "off" switch is not a "keep zero" switch.
func TestAuditCapRowsNoopWhenUnderOrUnset(t *testing.T) {
	db := newAuditTestDB(t)
	ctx := context.Background()
	svc := NewAuditService(db)
	seedAuditRowAt(t, db, "cap.a", time.Now())
	seedAuditRowAt(t, db, "cap.b", time.Now())

	if removed, err := svc.CapRows(ctx, 5); err != nil || removed != 0 {
		t.Fatalf("under-cap CapRows(5) = %d,%v, want 0,nil", removed, err)
	}
	if removed, err := svc.CapRows(ctx, 0); err != nil || removed != 0 {
		t.Fatalf("disabled CapRows(0) = %d,%v, want 0,nil", removed, err)
	}
	if count := auditRowCount(t, db); count != 2 {
		t.Fatalf("table holds %d rows, want both kept", count)
	}
}

// The row cap must respect the same archive-before-delete rule as age
// retention: the trimmed rows have to land on disk before they disappear.
func TestAuditCapRowsArchivesBeforeDeleting(t *testing.T) {
	db := newAuditTestDB(t)
	archiveDir := t.TempDir()
	svc := NewAuditService(db).WithArchiveDir(archiveDir)
	ctx := context.Background()

	for _, action := range []string{"cap.old1", "cap.old2", "cap.new"} {
		seedAuditRowAt(t, db, action, time.Now())
	}

	removed, err := svc.CapRows(ctx, 1)
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d rows, want 2", removed)
	}
	if count := auditRowCount(t, db); count != 1 {
		t.Fatalf("table holds %d rows after cap, want only the newest", count)
	}

	matches, _ := filepath.Glob(filepath.Join(archiveDir, "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("archive files = %v, want exactly one daily file", matches)
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if !strings.Contains(string(body), "cap.old1") || !strings.Contains(string(body), "cap.old2") {
		t.Fatalf("archive is missing a capped row:\n%s", body)
	}
	if strings.Contains(string(body), "cap.new") {
		t.Fatalf("archive holds a row that should have been kept:\n%s", body)
	}
}
