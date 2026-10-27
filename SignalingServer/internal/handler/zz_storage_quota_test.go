package handler

import (
	"os"
	"path/filepath"
	"testing"
)

// QD-810 promises a retained-bytes quota, and a quota is only as good as the
// boundaries it holds.
//
// `storage_quota_test.go` already covers the two headline cases: the smaller of
// the two caps wins, and a full quota refuses. What is not covered is the staged
// upload, which is the interesting one because staged bytes are already inside the
// directory being measured. Getting that wrong charges the same bytes twice and
// rejects uploads that would have fitted.

func writeSizedQuotaFile(t *testing.T, dir, name string, size int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o640); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// Staged bytes are on disk, so the directory quota already counts them. They must
// be subtracted from the per-upload cap only - charging them against the quota as
// well would reject uploads that fit.
func TestStorageWriteAllowanceCountsStagedBytesAgainstTheDirectoryOnce(t *testing.T) {
	dir := t.TempDir()
	writeSizedQuotaFile(t, dir, "staged.bin", 40)

	got, err := storageWriteAllowance(dir, 100, 100, 30)
	if err != nil {
		t.Fatalf("storageWriteAllowance(): %v", err)
	}
	if got != 60 {
		t.Fatalf("allowance = %d, want 60; staged bytes were charged against the quota as well as the upload cap", got)
	}
}

// The per-upload cap wins when it is the tighter of the two, so a caller cannot
// use a generous quota to exceed the per-request limit.
func TestStorageWriteAllowanceTakesThePerUploadCapWhenItIsTighter(t *testing.T) {
	dir := t.TempDir()
	writeSizedQuotaFile(t, dir, "part.bin", 40)

	got, err := storageWriteAllowance(dir, 1000, 70, 40)
	if err != nil {
		t.Fatalf("storageWriteAllowance(): %v", err)
	}
	if got != 30 {
		t.Fatalf("allowance = %d, want 30 (the per-upload cap, not the quota's 960)", got)
	}
}

// No caps configured means no ceiling, which is how a deployment that has not set
// a quota keeps working.
func TestStorageWriteAllowanceWithNoCapsIsUnbounded(t *testing.T) {
	dir := t.TempDir()
	writeSizedQuotaFile(t, dir, "some.bin", 50)

	got, err := storageWriteAllowance(dir, 0, 0, 0)
	if err != nil {
		t.Fatalf("storageWriteAllowance(): %v", err)
	}
	if got >= 0 {
		t.Fatalf("allowance = %d, want a negative sentinel meaning unbounded", got)
	}
}

// A staged upload larger than the per-upload cap leaves nothing to write rather
// than a negative allowance, which a caller would read as "unbounded".
func TestStorageWriteAllowanceNeverReturnsNegativeForAnOversizedStage(t *testing.T) {
	dir := t.TempDir()

	got, err := storageWriteAllowance(dir, 0, 10, 25)
	if err != nil {
		t.Fatalf("storageWriteAllowance(): %v", err)
	}
	if got != 0 {
		t.Fatalf("allowance = %d, want 0", got)
	}
}
