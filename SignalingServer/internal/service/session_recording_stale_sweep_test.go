package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// QD-004 requires that a recording which cannot finish - the Host crashed, the
// disk filled, the encoder died - reaches an explicit terminal state rather
// than sitting in `recording` forever.
func TestSweepStaleRecordingsMarksFailedAndEnded(t *testing.T) {
	db := newStaleSweepTestDB(t)
	const userID = uint(9401)
	quiet := time.Now().UTC().Add(-2 * time.Hour)
	seedStaleSweepRecording(t, db, userID, 1, models.SessionRecording{
		Status:    "recording",
		CreatedAt: quiet,
		UpdatedAt: quiet,
	})
	// A live recording between chunks must not be touched.
	seedStaleSweepRecording(t, db, userID, 2, models.SessionRecording{Status: "recording"})
	// A recording that already finished is not a stale-sweep candidate, so the
	// pass is idempotent.
	seedStaleSweepRecording(t, db, userID, 3, models.SessionRecording{
		Status:    "completed",
		CreatedAt: quiet,
		UpdatedAt: quiet,
	})

	cutoff := time.Now().UTC().Add(-30 * time.Minute)
	failed, err := NewSessionRecordingService(db).SweepStaleRecordings(
		context.Background(), cutoff)
	if err != nil {
		t.Fatalf("SweepStaleRecordings(): %v", err)
	}
	if failed != 1 {
		t.Fatalf("failed = %d, want 1", failed)
	}

	var row models.SessionRecording
	if err := db.First(&row, 1).Error; err != nil {
		t.Fatalf("load recording: %v", err)
	}
	if row.Status != "failed" {
		t.Fatalf("status = %q, want failed", row.Status)
	}
	// `ended_at` is what the detail view reads to say when the capture stopped,
	// and what the retention pass dates the media by. A terminal row without it
	// still reads as "running, but failed".
	if row.EndedAt == nil {
		t.Fatal("a failed recording must record when it ended")
	}
	if row.EndedAt.Before(cutoff) || row.EndedAt.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("ended_at = %v is not the sweep time", *row.EndedAt)
	}

	var live models.SessionRecording
	if err := db.First(&live, 2).Error; err != nil {
		t.Fatalf("load live recording: %v", err)
	}
	if live.Status != "recording" || live.EndedAt != nil {
		t.Fatalf("live recording changed: status=%q ended_at=%v", live.Status, live.EndedAt)
	}
}

func newStaleSweepTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "stale-sweep.db"),
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
	if err := db.AutoMigrate(&models.SessionRecording{}, &models.User{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedStaleSweepRecording(t *testing.T, db *gorm.DB, userID uint, id uint, row models.SessionRecording) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "stale-sweep-owner",
		Password: "test-hash",
	}).Error; err != nil {
		// The same owner seeds several recordings in one test, so a duplicate
		// insert is expected rather than a failure.
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			t.Fatalf("seed user: %v", err)
		}
	}
	row.ID = id
	row.UserID = userID
	row.DeviceID = "900000001"
	// `updated_at` is the liveness signal the sweep reads, so each case sets its
	// own: a stale row is quiet for hours, a live one was touched just now.
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
		row.CreatedAt = row.UpdatedAt.Add(-2 * time.Hour)
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}
}
