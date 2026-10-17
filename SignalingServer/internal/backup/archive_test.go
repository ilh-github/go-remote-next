package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
)

func TestArchiveRoundTripRestoresDatabaseAndMedia(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "quickdesk.db")
	recordingsDir := filepath.Join(root, "data", "recordings")
	updatesDir := filepath.Join(root, "data", "updates")
	archivePath := filepath.Join(t.TempDir(), "quickdesk-backup.tar.gz")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		t.Fatalf("create database directory: %v", err)
	}
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: dbPath},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("AutoMigrate(User): %v", err)
	}
	user := models.User{Username: "backup-roundtrip", Password: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	recording := filepath.Join(recordingsDir, "7", "rec-42.mp4")
	if err := os.MkdirAll(filepath.Dir(recording), 0o750); err != nil {
		t.Fatalf("create recording directory: %v", err)
	}
	if err := os.WriteFile(recording, []byte("recording-bytes"), 0o640); err != nil {
		t.Fatalf("write recording: %v", err)
	}
	installer := filepath.Join(updatesDir, "9", "QuickDesk-4.4.0.0.exe")
	if err := os.MkdirAll(filepath.Dir(installer), 0o750); err != nil {
		t.Fatalf("create update directory: %v", err)
	}
	if err := os.WriteFile(installer, []byte("installer-bytes"), 0o640); err != nil {
		t.Fatalf("write installer: %v", err)
	}

	manifest, err := Create(context.Background(), CreateOptions{
		DatabasePath:  dbPath,
		RecordingsDir: recordingsDir,
		UpdatesDir:    updatesDir,
		OutputPath:    archivePath,
	})
	if err != nil {
		t.Fatalf("Create(): %v", err)
	}
	if manifest.FormatVersion != 1 || len(manifest.Files) != 3 {
		t.Fatalf("manifest = %+v, want version 1 and database plus two media files", manifest)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("backup archive missing: %v", err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("remove original data root: %v", err)
	}
	result, err := Restore(context.Background(), RestoreOptions{
		ArchivePath:   archivePath,
		DatabasePath:  dbPath,
		RecordingsDir: recordingsDir,
		UpdatesDir:    updatesDir,
	})
	if err != nil {
		t.Fatalf("Restore(): %v", err)
	}
	if result.RestoredFiles != 3 {
		t.Fatalf("restored files = %d, want 3", result.RestoredFiles)
	}

	restored, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{Driver: "sqlite", Path: dbPath},
	})
	if err != nil {
		t.Fatalf("reopen restored SQLite: %v", err)
	}
	restoredSQL, err := restored.DB()
	if err != nil {
		t.Fatalf("restored.DB(): %v", err)
	}
	t.Cleanup(func() { _ = restoredSQL.Close() })
	var got models.User
	if err := restored.Where("username = ?", user.Username).First(&got).Error; err != nil {
		t.Fatalf("read restored user: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("restored user id = %d, want %d", got.ID, user.ID)
	}

	for path, want := range map[string][]byte{
		recording: []byte("recording-bytes"),
		installer: []byte("installer-bytes"),
	} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read restored %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("restored %s = %q, want %q", path, got, want)
		}
	}
}

func TestRestoreRejectsArchivePathTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "malicious.tar.gz")
	writeTestArchive(t, archivePath, map[string][]byte{
		"../escaped.txt": []byte("must not escape"),
	})
	restoreRoot := filepath.Join(root, "restore")
	_, err := Restore(context.Background(), RestoreOptions{
		ArchivePath:  archivePath,
		DatabasePath: filepath.Join(restoreRoot, "quickdesk.db"),
	})
	if err == nil || !strings.Contains(err.Error(), "escapes the restore root") {
		t.Fatalf("Restore() error = %v, want path traversal rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "escaped.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("path traversal wrote outside the restore root: %v", statErr)
	}
}

func TestRestoreRejectsManifestChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "tampered.tar.gz")
	manifest := Manifest{
		FormatVersion: formatVersion,
		CreatedAt:     time.Now().UTC(),
		Database:      databaseName,
		Files: []FileRecord{{
			Path:   databaseName,
			Size:   4,
			SHA256: strings.Repeat("0", 64),
		}},
		TotalBytes: 4,
	}
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeTestArchive(t, archivePath, map[string][]byte{
		manifestName: rawManifest,
		databaseName: []byte("oops"),
	})
	_, err = Restore(context.Background(), RestoreOptions{
		ArchivePath:  archivePath,
		DatabasePath: filepath.Join(root, "restored.db"),
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Restore() error = %v, want checksum mismatch", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "restored.db")); !os.IsNotExist(statErr) {
		t.Fatalf("a checksum-rejected database was installed: %v", statErr)
	}
}

func TestRestoreForceMovesPreviousDatabaseAside(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "valid.tar.gz")
	body := []byte("new-database")
	sum := sha256.Sum256(body)
	manifest := Manifest{
		FormatVersion: formatVersion,
		CreatedAt:     time.Now().UTC(),
		Database:      databaseName,
		Files: []FileRecord{{
			Path:   databaseName,
			Size:   int64(len(body)),
			SHA256: hex.EncodeToString(sum[:]),
		}},
		TotalBytes: int64(len(body)),
	}
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeTestArchive(t, archivePath, map[string][]byte{
		manifestName: rawManifest,
		databaseName: body,
	})

	dbPath := filepath.Join(root, "data", "quickdesk.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		t.Fatalf("create database directory: %v", err)
	}
	if err := os.WriteFile(dbPath, []byte("old-database"), 0o600); err != nil {
		t.Fatalf("write existing database: %v", err)
	}
	result, err := Restore(context.Background(), RestoreOptions{
		ArchivePath:  archivePath,
		DatabasePath: dbPath,
		Force:        true,
	})
	if err != nil {
		t.Fatalf("Restore(): %v", err)
	}
	if len(result.PreviousCopies) != 1 {
		t.Fatalf("previous copies = %v, want one database backup", result.PreviousCopies)
	}
	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read restored database: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("restored database = %q, want %q", got, body)
	}
	previous, err := os.ReadFile(result.PreviousCopies[0])
	if err != nil {
		t.Fatalf("read previous database copy: %v", err)
	}
	if string(previous) != "old-database" {
		t.Fatalf("previous database = %q, want old-database", previous)
	}
}

func TestRestoreRollsBackDatabaseWhenMediaTargetCannotBeInstalled(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "two-targets.tar.gz")
	dbBody := []byte("database-body")
	mediaBody := []byte("media-body")
	dbSum := sha256.Sum256(dbBody)
	mediaSum := sha256.Sum256(mediaBody)
	mediaPath := "recordings/7/rec-42.mp4"
	manifest := Manifest{
		FormatVersion: formatVersion,
		CreatedAt:     time.Now().UTC(),
		Database:      databaseName,
		Files: []FileRecord{
			{Path: databaseName, Size: int64(len(dbBody)), SHA256: hex.EncodeToString(dbSum[:])},
			{Path: mediaPath, Size: int64(len(mediaBody)), SHA256: hex.EncodeToString(mediaSum[:])},
		},
		TotalBytes: int64(len(dbBody) + len(mediaBody)),
	}
	rawManifest, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeTestArchive(t, archivePath, map[string][]byte{
		manifestName: rawManifest,
		databaseName: dbBody,
		mediaPath:    mediaBody,
	})

	blockedParent := filepath.Join(root, "blocked")
	if err := os.MkdirAll(blockedParent, 0o500); err != nil {
		t.Fatalf("create blocked parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedParent, 0o750) })
	dbPath := filepath.Join(root, "target", "quickdesk.db")
	_, err = Restore(context.Background(), RestoreOptions{
		ArchivePath:   archivePath,
		DatabasePath:  dbPath,
		RecordingsDir: filepath.Join(blockedParent, "recordings"),
	})
	if err == nil {
		t.Fatal("Restore() succeeded despite an unwritable media target")
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Fatalf("database was left replaced after media restore failed: %v", statErr)
	}
}

func writeTestArchive(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create archive directory: %v", err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		header := &tar.Header{
			Name:     name,
			Mode:     0o640,
			Size:     int64(len(body)),
			ModTime:  time.Now().UTC(),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %s: %v", name, err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatalf("write tar body %s: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive missing: %v (%s)", err, path)
	}
}
