package service

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func remainingFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// The sweep must delete only old, correctly named archive files: today plus the
// previous N-1 days are kept, and anything the operator put in the directory
// that this server does not recognise is never touched.
func TestSweepArchiveDirRemovesOnlyOldMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2026-10-16.jsonl") // today
	writeFile(t, dir, "audit-logs-2026-10-14.jsonl") // inside the window
	writeFile(t, dir, "audit-logs-2026-10-01.jsonl") // outside
	writeFile(t, dir, "audit-logs-2000-01-01.jsonl") // far outside
	writeFile(t, dir, "audit-logs-notadate.jsonl")   // unrecognised stamp
	writeFile(t, dir, "other-2026-10-01.jsonl")      // wrong prefix
	writeFile(t, dir, "README.md")                   // not an archive

	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)
	removed, err := sweepArchiveDir(dir, "audit-logs", 3, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed %d files, want the two out-of-window archives", removed)
	}
	want := []string{
		"README.md",
		"audit-logs-2026-10-14.jsonl",
		"audit-logs-2026-10-16.jsonl",
		"audit-logs-notadate.jsonl",
		"other-2026-10-01.jsonl",
	}
	if got := remainingFiles(t, dir); !equalStrings(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
}

// A retention of 1 keeps only today; the boundary is inclusive so the file for
// the cutoff day itself survives.
func TestSweepArchiveDirKeepsTheCutoffDay(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2026-10-16.jsonl")
	writeFile(t, dir, "audit-logs-2026-10-15.jsonl")

	now := time.Date(2026, 10, 16, 23, 30, 0, 0, time.UTC)
	removed, err := sweepArchiveDir(dir, "audit-logs", 1, now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed %d files, want exactly yesterday", removed)
	}
	if got, want := remainingFiles(t, dir), []string{"audit-logs-2026-10-16.jsonl"}; !equalStrings(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
}

// Disabled, or pointed at a directory that does not exist, must be a no-op
// rather than an error: the default is "keep the archives forever".
func TestSweepArchiveDirDisabledOrMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "audit-logs-2000-01-01.jsonl")
	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)

	if removed, err := sweepArchiveDir(dir, "audit-logs", 0, now); err != nil || removed != 0 {
		t.Fatalf("disabled sweep = %d,%v, want 0,nil", removed, err)
	}
	if removed, err := sweepArchiveDir(filepath.Join(dir, "nope"), "audit-logs", 3, now); err != nil || removed != 0 {
		t.Fatalf("missing dir sweep = %d,%v, want 0,nil", removed, err)
	}
	if got := remainingFiles(t, dir); len(got) != 1 {
		t.Fatalf("remaining = %v, want the archive untouched", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
