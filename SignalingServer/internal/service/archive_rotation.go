package service

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Daily archive files are the second half of retention: the table (or outbox)
// is bounded, and the bytes that left it live in `<dir>/<prefix>-YYYY-MM-DD.jsonl`.
// That directory only ever grew, so a long-lived deployment accumulated one
// file per day forever. This is the sweep for it.

// archiveFileLayout is the date stamp both archive writers put in their file
// names (`audit-logs-YYYY-MM-DD.jsonl`, `notification-deliveries-YYYY-MM-DD.jsonl`).
const archiveFileLayout = "2006-01-02"

// SweepArchiveDir deletes daily archive files older than retentionDays and
// reports how many it removed. Only files named `<prefix>-YYYY-MM-DD.jsonl` are
// candidates: anything else in the directory (a README, a hand-rotated copy, a
// temporary file) is left alone, because the archive directory belongs to the
// operator as much as to this server.
//
// retentionDays <= 0 disables it. That default matters: archiving exists so a
// purged row stays recoverable, so "keep the archives forever" is the safe
// default and deleting them has to be asked for explicitly.
//
// The boundary is inclusive of today: `retentionDays = 3` keeps today plus the
// two previous days and deletes anything older. A missing directory is not an
// error - a deployment that never archived simply has nothing to sweep.
func sweepArchiveDir(dir, prefix string, retentionDays int, now time.Time) (int, error) {
	if retentionDays <= 0 || strings.TrimSpace(dir) == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	// Midnight of the oldest day to keep. Comparing against a day boundary
	// rather than "now" avoids deleting a file whose date happens to equal the
	// current time-of-day subtraction.
	utcNow := now.UTC()
	keepFrom := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(retentionDays - 1))

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		date, ok := archiveFileDate(entry.Name(), prefix)
		if !ok || !date.Before(keepFrom) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// archiveFileDate extracts the day from one archive file name, reporting false
// for anything that does not match the expected `<prefix>-YYYY-MM-DD.jsonl`
// shape so unrecognised files are never deleted.
func archiveFileDate(name, prefix string) (time.Time, bool) {
	if !strings.HasPrefix(name, prefix+"-") || !strings.HasSuffix(name, ".jsonl") {
		return time.Time{}, false
	}
	stamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix+"-"), ".jsonl")
	date, err := time.Parse(archiveFileLayout, stamp)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}
