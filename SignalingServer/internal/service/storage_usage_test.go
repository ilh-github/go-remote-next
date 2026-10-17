package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectDirectoryStorageCountsNestedFilesAndRemainingQuota(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.bin"), make([]byte, 3), 0o640); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "two.bin"), make([]byte, 9), 0o640); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	got, err := InspectDirectoryStorage(dir, 20)
	if err != nil {
		t.Fatalf("InspectDirectoryStorage(): %v", err)
	}
	if got.Files != 2 || got.UsedBytes != 12 || got.QuotaBytes != 20 ||
		got.RemainingBytes != 8 || got.OverQuota {
		t.Fatalf("usage = %+v, want files=2 used=12 quota=20 remaining=8 over_quota=false", got)
	}
}

func TestInspectDirectoryStorageClampsOverQuotaRemainingToZero(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.bin"), make([]byte, 12), 0o640); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := InspectDirectoryStorage(dir, 10)
	if err != nil {
		t.Fatalf("InspectDirectoryStorage(): %v", err)
	}
	if got.UsedBytes != 12 || got.RemainingBytes != 0 || !got.OverQuota {
		t.Fatalf("usage = %+v, want used=12 remaining=0 over_quota=true", got)
	}
}
