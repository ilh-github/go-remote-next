package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeLogFile drops a synthetic log file into dir. Lines are written verbatim
// so a test can control exactly what the reverse scanner sees.
func writeLogFile(t *testing.T, dir, name string, lines []string) {
	t.Helper()
	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// stamp renders the prefix the standard logger writes.
func stamp(at time.Time) string {
	return at.UTC().Format("2006/01/02 15:04:05.000000")
}

func sampleLogDir(t *testing.T) (string, time.Time) {
	t.Helper()
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)
	writeLogFile(t, dir, "signaling.log", []string{
		stamp(base) + " component=api_auth event=request path=/v1/auth/sessions",
		stamp(base.Add(time.Second)) + " component=api event=other plain line",
		stamp(base.Add(5*time.Minute)) + " component=notification event=enqueue channel=7",
		stamp(base.Add(10*time.Minute)) + " a line without any component field",
	})
	// .1 is the most recent rotation, i.e. older than signaling.log.
	writeLogFile(t, dir, "signaling.log.1", []string{
		stamp(base.Add(-time.Hour)) + " component=api_auth event=old",
	})
	return dir, base
}

func lineTexts(entries []LogEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Line)
	}
	return out
}

// Newest first is the whole point of a reverse scan, and the order has to
// survive crossing into a rotated file.
func TestLogQueryReturnsNewestFirstAcrossRotations(t *testing.T) {
	dir, _ := sampleLogDir(t)
	svc := NewLogQueryService(dir)

	res, err := svc.Query(context.Background(), LogQuery{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 5 {
		t.Fatalf("got %d lines, want all 5 across both files: %v", len(res.Entries), lineTexts(res.Entries))
	}
	wantSubstrings := []string{
		"a line without any component field",
		"component=notification",
		"component=api event=other",
		"component=api_auth event=request",
		"component=api_auth event=old",
	}
	for i, want := range wantSubstrings {
		if !strings.Contains(res.Entries[i].Line, want) {
			t.Fatalf("entry %d = %q, want it to contain %q", i, res.Entries[i].Line, want)
		}
	}
	if len(res.Files) != 2 || res.Files[0].Name != "signaling.log" || res.Files[1].Name != "signaling.log.1" {
		t.Fatalf("files = %+v, want the current log first then .1", res.Files)
	}
	if res.Entries[0].Timestamp == "" {
		t.Fatal("entries should carry a parsed timestamp")
	}
	// The limit is a page size, not a truncation of the search.
	page, err := svc.Query(context.Background(), LogQuery{Limit: 2})
	if err != nil {
		t.Fatalf("paged query: %v", err)
	}
	if len(page.Entries) != 2 || page.Truncated {
		t.Fatalf("limit 2 returned %d entries (truncated=%t), want 2 and not truncated", len(page.Entries), page.Truncated)
	}
}

// `q` is a substring match; `component` is a whole token. The difference
// matters: filtering component=api must not drag in component=api_auth.
func TestLogQueryFiltersSubstringAndComponentToken(t *testing.T) {
	dir, _ := sampleLogDir(t)
	svc := NewLogQueryService(dir)
	ctx := context.Background()

	res, err := svc.Query(ctx, LogQuery{Query: "NOTIFICATION"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "component=notification") {
		t.Fatalf("case-insensitive substring match got %v", lineTexts(res.Entries))
	}

	res, err = svc.Query(ctx, LogQuery{Component: "api"})
	if err != nil {
		t.Fatalf("component query: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("component=api matched %d lines, want 1 (api_auth is a different component): %v",
			len(res.Entries), lineTexts(res.Entries))
	}
	if !strings.Contains(res.Entries[0].Line, "component=api event=other") {
		t.Fatalf("component=api matched the wrong line: %q", res.Entries[0].Line)
	}

	res, err = svc.Query(ctx, LogQuery{Component: "api_auth"})
	if err != nil {
		t.Fatalf("component query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("component=api_auth matched %d lines, want 2", len(res.Entries))
	}
}

// The time bounds are inclusive and the window is read from the line's own
// timestamp, so a boundary line is never lost to an off-by-one.
func TestLogQueryTimeWindowIsInclusive(t *testing.T) {
	dir, base := sampleLogDir(t)
	svc := NewLogQueryService(dir)
	ctx := context.Background()

	at := base.Add(5 * time.Minute)
	res, err := svc.Query(ctx, LogQuery{Since: &at, Until: &at})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "component=notification") {
		t.Fatalf("exact-boundary window returned %v, want the 10:05 line", lineTexts(res.Entries))
	}

	since := base.Add(5 * time.Minute)
	res, err = svc.Query(ctx, LogQuery{Since: &since})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("since=10:05 returned %d lines, want 2", len(res.Entries))
	}

	until := base.Add(-time.Hour)
	res, err = svc.Query(ctx, LogQuery{Until: &until})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != 1 || !strings.Contains(res.Entries[0].Line, "event=old") {
		t.Fatalf("until=09:00 returned %v, want only the rotated line", lineTexts(res.Entries))
	}
}

// The scanner reads fixed-size blocks from the end, so a line that straddles a
// block boundary has to be reassembled rather than handed out in pieces. This
// writes a file several blocks long and checks the count and the oldest line.
func TestLogQueryReassemblesLinesAcrossBlocks(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	const total = 1500
	lines := make([]string, 0, total)
	for i := 1; i <= total; i++ {
		lines = append(lines, fmt.Sprintf("%s component=load event=beat seq=%04d payload=%s",
			stamp(base.Add(time.Duration(i)*time.Second)), i, strings.Repeat("x", 30)))
	}
	writeLogFile(t, dir, "signaling.log", lines)
	info, err := os.Stat(filepath.Join(dir, "signaling.log"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() <= logBlockSize {
		t.Fatalf("fixture is only %d bytes, it has to exceed one %d-byte block", info.Size(), logBlockSize)
	}

	svc := NewLogQueryService(dir)
	res, err := svc.Query(context.Background(), LogQuery{Limit: total})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Entries) != total {
		t.Fatalf("got %d lines, want %d", len(res.Entries), total)
	}
	if res.Truncated {
		t.Fatal("a full read of the file must not report truncation")
	}
	if !strings.Contains(res.Entries[0].Line, "seq=1500") {
		t.Fatalf("newest line is %q, want seq=1500", res.Entries[0].Line)
	}
	if !strings.Contains(res.Entries[total-1].Line, "seq=0001") {
		t.Fatalf("oldest line is %q, want seq=0001", res.Entries[total-1].Line)
	}
	// Every line has to come back whole; a block-boundary bug shows up as a
	// fragment with no timestamp.
	for i, entry := range res.Entries {
		if entry.Timestamp == "" {
			t.Fatalf("entry %d is not a whole line: %q", i, entry.Line)
		}
	}
}

// Running out of byte budget must be reported, because an empty result from a
// truncated scan is not evidence that the event never happened.
func TestLogQueryReportsTruncatedScan(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	lines := make([]string, 0, 1200)
	for i := 1; i <= 1200; i++ {
		lines = append(lines, fmt.Sprintf("%s component=load event=beat seq=%04d", stamp(base), i))
	}
	writeLogFile(t, dir, "signaling.log", lines)

	svc := NewLogQueryService(dir)
	svc.scanBudget = 32 * 1024 // smaller than one block, so the walk stops early
	res, err := svc.Query(context.Background(), LogQuery{Limit: 5000})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("scan stopped at %d bytes but truncated=false (scanned %d lines)",
			res.ScannedBytes, res.ScannedLines)
	}
	if res.ScannedBytes > svc.scanBudget {
		t.Fatalf("scanned %d bytes, over the %d-byte budget", res.ScannedBytes, svc.scanBudget)
	}

	// A missing log is not an error: a server that has not written yet simply
	// has nothing to show.
	empty := NewLogQueryService(t.TempDir())
	res, err = empty.Query(context.Background(), LogQuery{})
	if err != nil {
		t.Fatalf("missing log: %v", err)
	}
	if len(res.Entries) != 0 || len(res.Files) != 0 {
		t.Fatalf("missing log returned %+v, want empty", res)
	}
}
