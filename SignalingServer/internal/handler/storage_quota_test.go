package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

func TestStorageWriteAllowanceUsesTheClosestRemainingLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "retained.bin"), make([]byte, 600), 0o640); err != nil {
		t.Fatalf("write retained file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "upload.part"), make([]byte, 100), 0o640); err != nil {
		t.Fatalf("write staged file: %v", err)
	}

	got, err := storageWriteAllowance(dir, 1000, 500, 100)
	if err != nil {
		t.Fatalf("storageWriteAllowance(): %v", err)
	}
	if got != 300 {
		t.Fatalf("allowance = %d, want the 300 bytes left by the 1KB quota", got)
	}
}

func TestStorageWriteAllowanceRejectsAFullQuota(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "full.bin"), make([]byte, 1000), 0o640); err != nil {
		t.Fatalf("write full file: %v", err)
	}

	_, err := storageWriteAllowance(dir, 1000, 500, 0)
	if !errors.Is(err, errStorageQuotaExceeded) {
		t.Fatalf("error = %v, want errStorageQuotaExceeded", err)
	}
}

func TestUpdateArtifactUploadRejectsStorageQuotaExceeded(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "quota.db"),
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
	if err := db.AutoMigrate(&models.UpdateRelease{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	payload := bytes.Repeat([]byte("q"), 300)
	sum := sha256.Sum256(payload)
	releases := service.NewUpdateReleaseService(db)
	release, err := releases.Publish(context.Background(), service.PublishUpdateReleaseInput{
		Platform:  "windows",
		Arch:      "x64",
		Version:   "4.4.0.0",
		SHA256:    hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	storeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(storeDir, "retained.bin"), make([]byte, 800), 0o640); err != nil {
		t.Fatalf("write retained file: %v", err)
	}
	h := NewAdminUpdateReleaseHandler(releases, nil, UpdateArtifactStore{
		Dir:        storeDir,
		MaxBytes:   1 << 20,
		QuotaBytes: 1000,
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/admin/update-releases/:id/artifact", h.UploadArtifact)

	path := "/v1/admin/update-releases/" + strconv.FormatUint(uint64(release.ID), 10) +
		"/artifact?filename=QuickDesk-4.4.0.0.exe"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	dest := filepath.Join(storeDir, strconv.FormatUint(uint64(release.ID), 10), "QuickDesk-4.4.0.0.exe")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("quota-rejected artifact was written to %s: %v", dest, err)
	}
}

func TestAdminRecordingStorageReportsIndexedAndDiskBytes(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "recording-storage.db"),
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
	if err := db.AutoMigrate(&models.SessionRecording{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "artifact.bin"), make([]byte, 700), 0o640); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "preview.jpg"), make([]byte, 100), 0o640); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	h := NewAdminSessionRecordingHandler(
		service.NewSessionRecordingService(db), nil, nil, 30, 240,
	).WithStorageSummary(dir, 1000)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/recordings:storage", h.Storage)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/recordings:storage", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ArtifactCount  int64 `json:"artifact_count"`
		IndexedBytes   int64 `json:"indexed_bytes"`
		TotalBytes     int64 `json:"total_bytes"`
		FileCount      int64 `json:"file_count"`
		QuotaBytes     int64 `json:"quota_bytes"`
		RemainingBytes int64 `json:"remaining_bytes"`
		OverQuota      bool  `json:"over_quota"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode storage response: %v", err)
	}
	if got.ArtifactCount != 0 || got.IndexedBytes != 0 || got.TotalBytes != 800 ||
		got.FileCount != 2 || got.QuotaBytes != 1000 || got.RemainingBytes != 200 || got.OverQuota {
		t.Fatalf("storage response = %+v, want indexed=0 disk=800 files=2 quota=1000 remaining=200", got)
	}
}

func TestAdminUpdateStorageReportsDiskUsageAndQuota(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "installer.exe"), make([]byte, 900), 0o640); err != nil {
		t.Fatalf("write installer: %v", err)
	}
	h := NewAdminUpdateReleaseHandler(nil, nil, UpdateArtifactStore{
		Dir:        dir,
		QuotaBytes: 1000,
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/update-releases:storage", h.Storage)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/update-releases:storage", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		FileCount      int64 `json:"file_count"`
		TotalBytes     int64 `json:"total_bytes"`
		QuotaBytes     int64 `json:"quota_bytes"`
		RemainingBytes int64 `json:"remaining_bytes"`
		OverQuota      bool  `json:"over_quota"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode storage response: %v", err)
	}
	if got.FileCount != 1 || got.TotalBytes != 900 || got.QuotaBytes != 1000 ||
		got.RemainingBytes != 100 || got.OverQuota {
		t.Fatalf("storage response = %+v, want files=1 used=900 quota=1000 remaining=100", got)
	}
}
