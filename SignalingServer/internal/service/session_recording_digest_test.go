package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// QD-004 requires the server to verify an artifact's digest before attaching
// it. A Host uploads bytes the server has never seen, so a truncated or
// corrupted transfer must not become a playable recording.
func TestArtifactDigestMatchesTheBytesOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-1.h264")
	content := []byte("host-recorded-frames")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])

	got, size, err := ArtifactDigest(path)
	if err != nil {
		t.Fatalf("ArtifactDigest(): %v", err)
	}
	if got != want {
		t.Fatalf("digest = %q, want %q", got, want)
	}
	if size != int64(len(content)) {
		t.Fatalf("size = %d, want %d", size, len(content))
	}
}

// A digest that does not match the bytes must be refused. This is the case the
// requirement exists for: the Host declared one thing and sent another.
func TestArtifactDigestRejectsAMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-2.h264")
	if err := os.WriteFile(path, []byte("actual-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	sum := sha256.Sum256([]byte("declared-bytes"))
	if err := VerifyArtifactDigest(path, hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("VerifyArtifactDigest() accepted a mismatching digest")
	}
}

// An empty file is not a recording, whatever digest it happens to have.
func TestArtifactDigestRejectsAnEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-3.h264")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if _, _, err := ArtifactDigest(path); err == nil {
		t.Fatal("ArtifactDigest() accepted an empty artifact")
	}
}

// A missing file must be reported as such rather than hashed as empty.
func TestArtifactDigestRejectsAMissingFile(t *testing.T) {
	if _, _, err := ArtifactDigest(filepath.Join(t.TempDir(), "absent.h264")); err == nil {
		t.Fatal("ArtifactDigest() accepted a missing artifact")
	}
}

// A digest with no declared value skips verification, because a caller that
// cannot compute one - a browser upload, for instance - must not be locked out.
func TestVerifyArtifactDigestSkipsAnUndeclaredDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-4.h264")
	if err := os.WriteFile(path, []byte("anything"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if err := VerifyArtifactDigest(path, ""); err != nil {
		t.Fatalf("VerifyArtifactDigest(\"\") = %v, want nil", err)
	}
	if err := VerifyArtifactDigest(path, "   "); err != nil {
		t.Fatalf("VerifyArtifactDigest(blank) = %v, want nil", err)
	}
}

// The comparison is case-insensitive: a Host that upper-cases its hex encoding
// has not sent different bytes.
func TestVerifyArtifactDigestAcceptsUpperHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-5.h264")
	content := []byte("case-insensitive")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	sum := sha256.Sum256(content)
	if err := VerifyArtifactDigest(path, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("lower hex rejected: %v", err)
	}
}

// A malformed digest is refused rather than silently skipped: a caller that
// sent garbage in the header is misbehaving, and treating it as "no digest"
// would defeat the check.
func TestVerifyArtifactDigestRejectsAMalformedDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rec-6.h264")
	if err := os.WriteFile(path, []byte("bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if err := VerifyArtifactDigest(path, "not-a-digest"); err == nil {
		t.Fatal("VerifyArtifactDigest() accepted a malformed digest")
	}
}

// The verification has to happen where the artifact is attached, not only in a
// helper nobody calls. A Host that declares one digest and sends different bytes
// must leave the row unattached rather than recorded as a playable recording.
func TestAttachArtifactRefusesADigestMismatch(t *testing.T) {
	db := newDigestAttachTestDB(t)
	const userID = uint(9201)
	seedDigestRecording(t, db, userID, 1, "")

	dir := t.TempDir()
	path := filepath.Join(dir, "rec-1.h264")
	if err := os.WriteFile(path, []byte("actual-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	sum := sha256.Sum256([]byte("declared-bytes"))

	_, err := NewSessionRecordingService(db).AttachArtifactVerified(
		context.Background(), userID, 1, path, ".h264",
		int64(len("actual-bytes")), hex.EncodeToString(sum[:]),
	)
	if err == nil {
		t.Fatal("AttachArtifactVerified() accepted a mismatching digest")
	}

	var row models.SessionRecording
	if err := db.First(&row, 1).Error; err != nil {
		t.Fatalf("load recording: %v", err)
	}
	if row.ArtifactPath != "" {
		t.Fatalf("recording gained artifact_path %q despite the mismatch", row.ArtifactPath)
	}
}

// A matching digest is accepted, so the check does not lock out the Host it is
// meant to protect.
func TestAttachArtifactAcceptsAMatchingDigest(t *testing.T) {
	db := newDigestAttachTestDB(t)
	const userID = uint(9202)
	seedDigestRecording(t, db, userID, 2, "")

	dir := t.TempDir()
	path := filepath.Join(dir, "rec-2.h264")
	content := []byte("host-recorded-frames")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	sum := sha256.Sum256(content)

	row, err := NewSessionRecordingService(db).AttachArtifactVerified(
		context.Background(), userID, 2, path, ".h264", int64(len(content)),
		hex.EncodeToString(sum[:]),
	)
	if err != nil {
		t.Fatalf("AttachArtifactVerified(): %v", err)
	}
	if row.ArtifactPath != path {
		t.Fatalf("artifact_path = %q, want %q", row.ArtifactPath, path)
	}
}

func newDigestAttachTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "digest-attach.db"),
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

func seedDigestRecording(t *testing.T, db *gorm.DB, userID uint, id uint, sessionID string) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       userID,
		Username: "digest-attach-owner",
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.SessionRecording{
		ID:        id,
		UserID:    userID,
		DeviceID:  "900000001",
		SessionID: sessionID,
		Status:    "recording",
	}).Error; err != nil {
		t.Fatalf("seed recording: %v", err)
	}
}
