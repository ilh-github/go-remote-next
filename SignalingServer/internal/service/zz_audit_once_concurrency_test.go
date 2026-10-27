package service

import (
	"context"
	"sync"
	"testing"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// QD-909's "one row per transfer id" claim, under the concurrency it actually
// faces.
//
// `LogActorOnce` is documented as accepting the same (actor, action, resource
// type, resource id) only once, and the transfer-report endpoints rely on it: a
// controller retries a report, and the trail must not gain a second row for the
// same transfer. The check is a count followed by an insert, with no unique
// constraint behind it, so two concurrent reports both see zero and both write.
//
// This is the difference between an idempotency that holds and one that holds
// only when nobody is in a hurry. A retry is exactly when two reports overlap.

// concurrentAuditDB gives each goroutine its own handle. A shared *gorm.DB is
// safe for concurrent use, but the point of the test is two racing inserts, and
// two handles make the race the database actually sees rather than one the
// connection pool serialises away.
func concurrentAuditDB(t *testing.T) []*gorm.DB {
	t.Helper()
	base := newAuditTestDB(t)
	out := make([]*gorm.DB, 2)
	for i := range out {
		handle, err := gorm.Open(base.Config.Dialector, base.Config)
		if err != nil {
			t.Fatalf("open handle %d: %v", i, err)
		}
		sqlDB, err := handle.DB()
		if err != nil {
			t.Fatalf("handle %d sql.DB: %v", i, err)
		}
		t.Cleanup(func() { _ = sqlDB.Close() })
		out[i] = handle
	}
	return out
}

func TestLogActorOnceWritesOneRowWhenTwoReportsRace(t *testing.T) {
	handles := concurrentAuditDB(t)
	ctx := context.Background()
	const (
		actor      = uint(4242)
		action     = "file.transfer"
		resourceID = "tr-race-1"
	)

	var wg sync.WaitGroup
	written := make([]bool, len(handles))
	for i, handle := range handles {
		wg.Add(1)
		go func(i int, handle *gorm.DB) {
			defer wg.Done()
			ok, err := NewAuditService(handle).LogActorOnce(
				ctx, actor, action, "file_transfer", resourceID, `{"directions":["upload"]}`, "203.0.113.9")
			if err != nil {
				t.Errorf("LogActorOnce() error: %v", err)
				return
			}
			written[i] = ok
		}(i, handle)
	}
	wg.Wait()

	winners := 0
	for _, ok := range written {
		if ok {
			winners++
		}
	}
	if winners > 1 {
		t.Fatalf("%d concurrent reports were each told they wrote, want at most 1", winners)
	}
	if count := auditRowCount(t, handles[0]); count != 1 {
		t.Fatalf("audit_logs has %d rows for one transfer id, want 1", count)
	}
}

// The index is scoped to the actions in models.AuditOnceActions, and that scope
// is what keeps two features alive at once.
//
// `LogActorDeduped` writes several rows for the same (actor, action, resource)
// over a time window - that is the entire point of a window - so a unique index
// covering every actor row would forbid it. This test failed while the index was
// unscoped, which is how the scope was found: two features wanted opposite things
// on the same tuple, and only one of them was written down.
func TestLogActorDedupedStillRepeatsUnderTheIdentityIndex(t *testing.T) {
	db := newAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	// Past the window, the same read is a new row. The media audit exists to
	// collapse a burst, not to make one row per resource forever.
	svc.LogActorDeduped(ctx, 9, "recording.download", "session_recording", "77", "", "", 0)
	svc.LogActorDeduped(ctx, 9, "recording.download", "session_recording", "77", "", "", 0)

	if count := auditRowCount(t, db); count != 2 {
		t.Fatalf("audit_logs has %d rows for two deduped reads, want 2; the identity index has swallowed the media audit", count)
	}
}

// And the actions the index does cover stay covered, so the scope cannot be
// widened by accident to "nothing" and silently drop the guarantee.
func TestOnceActionsAreCoveredByTheIdentityIndex(t *testing.T) {
	if len(models.AuditOnceActions) != 2 {
		t.Fatalf("AuditOnceActions = %v, want the two transfer actions", models.AuditOnceActions)
	}
	for _, want := range []string{"file.transfer", "clipboard.sync"} {
		found := false
		for _, action := range models.AuditOnceActions {
			if action == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("AuditOnceActions is missing %q; its rows would silently lose the one-row-per-transfer guarantee", want)
		}
	}
}
