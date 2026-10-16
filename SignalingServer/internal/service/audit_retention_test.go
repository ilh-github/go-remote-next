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
