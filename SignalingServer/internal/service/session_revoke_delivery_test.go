package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// The delivery loop is what makes a recorded command mean anything. A row nobody
// acts on is worse than no row: it looks like the revoke is being handled while
// the socket stays open.

type fakeRevokeChannel struct {
	published []publishedRevoke
	failOn    int // 1-based attempt index that fails; 0 means never
	attempts  int
}

type publishedRevoke struct {
	instanceID string
	sessionID  string
}

func (f *fakeRevokeChannel) Publish(_ context.Context, instanceID, sessionID string) error {
	f.attempts++
	if f.failOn != 0 && f.attempts == f.failOn {
		return errors.New("redis is unreachable")
	}
	f.published = append(f.published, publishedRevoke{instanceID: instanceID, sessionID: sessionID})
	return nil
}

func newRevokeDeliveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(
		"file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "revoke-delivery.db"))+"?cache=shared",
	), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.SessionRevokeCommand{}, &models.RemoteSession{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedOpenSession(t *testing.T, db *gorm.DB, sessionID, instanceID string) {
	t.Helper()
	if err := db.Create(&models.RemoteSession{
		SessionID:  sessionID,
		DeviceID:   "123456789",
		UserID:     7,
		InstanceID: instanceID,
		StartedAt:  time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed session %s: %v", sessionID, err)
	}
}

// The recorded command reaches the instance that holds the socket, addressed at
// that instance and nobody else.
func TestDeliverOncePendsTheCommandToItsInstance(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-deliver", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-deliver", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}

	channel := &fakeRevokeChannel{}
	deliverer := NewSessionRevokeCommandDeliverer(store, channel, nil)
	delivered, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}
	if delivered != 1 {
		t.Fatalf("delivered = %d, want 1", delivered)
	}
	if len(channel.published) != 1 {
		t.Fatalf("published = %+v, want one command", channel.published)
	}
	if channel.published[0].instanceID != "inst-far" || channel.published[0].sessionID != "sig-deliver" {
		t.Fatalf("published = %+v, want the command addressed at inst-far", channel.published)
	}

	// Delivered, and not owed again: a second pass must be a no-op rather than
	// re-closing a socket that is already closed.
	again, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("second DeliverOnce(): %v", err)
	}
	if again != 0 || len(channel.published) != 1 {
		t.Fatalf("second pass delivered = %d published = %+v, want 0 and 1", again, channel.published)
	}
}

// A command recorded by a process that then died is delivered by whoever runs
// the loop next. This is the whole point of the store, so it is asserted
// directly: record, close the "writer", and let a fresh deliverer act.
func TestDeliverOncePicksUpACommandItsRecorderNeverPublished(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-orphan", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-orphan", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}
	// The recorder dies here: nothing was ever published.
	channel := &fakeRevokeChannel{}
	delivered, err := NewSessionRevokeCommandDeliverer(store, channel, nil).
		DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}
	if delivered != 1 || len(channel.published) != 1 {
		t.Fatalf("delivered = %d published = %+v, want the abandoned command to be delivered", delivered, channel.published)
	}
	if channel.published[0].sessionID != "sig-orphan" {
		t.Fatalf("published = %+v, want sig-orphan", channel.published)
	}
}

// A publish that fails must not consume the command. The claim is released with
// the reason recorded, so the next pass tries again instead of treating the
// attempt as done - which is the failure mode that would make this whole
// mechanism a no-op.
func TestDeliverOnceRetriesAfterAPublishFailure(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-retry", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-retry", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}

	channel := &fakeRevokeChannel{failOn: 1}
	deliverer := NewSessionRevokeCommandDeliverer(store, channel, nil)
	first, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("first DeliverOnce(): %v", err)
	}
	if first != 0 {
		t.Fatalf("first pass delivered = %d, want 0 - the publish failed", first)
	}
	if len(channel.published) != 0 {
		t.Fatalf("published = %+v, want nothing after a failed publish", channel.published)
	}

	second, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("second DeliverOnce(): %v", err)
	}
	if second != 1 || len(channel.published) != 1 {
		t.Fatalf("second pass delivered = %d published = %+v, want the retry to land", second, channel.published)
	}
}

// A command whose session ended on its own is not delivered. Closing it would do
// nothing and would read as though the revoke had worked.
func TestDeliverOnceSkipsACommandWhoseSessionEnded(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-ended", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-ended", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}
	ended := time.Now().UTC()
	if err := db.Model(&models.RemoteSession{}).
		Where("session_id = ?", "sig-ended").
		Update("ended_at", ended).Error; err != nil {
		t.Fatalf("end the session: %v", err)
	}

	channel := &fakeRevokeChannel{}
	delivered, err := NewSessionRevokeCommandDeliverer(store, channel, nil).
		DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}
	if delivered != 0 || len(channel.published) != 0 {
		t.Fatalf("delivered = %d published = %+v, want nothing for a session that is over", delivered, channel.published)
	}
}

// The callback fires once per delivered command, because that is what the owning
// instance uses to log "closed on request from another instance" - and a
// duplicate callback would read as two closes.
func TestDeliverOnceReportsEachDeliveredCommandOnce(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-a", "inst-far")
	seedOpenSession(t, db, "sig-b", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	for _, id := range []string{"sig-a", "sig-b"} {
		if _, err := store.Record(context.Background(), id, "inst-far", "owner_revoke"); err != nil {
			t.Fatalf("Record(%s): %v", id, err)
		}
	}

	var reported []string
	channel := &fakeRevokeChannel{}
	deliverer := NewSessionRevokeCommandDeliverer(store, channel, func(sessionID string) {
		reported = append(reported, sessionID)
	})
	delivered, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}
	if delivered != 2 {
		t.Fatalf("delivered = %d, want 2", delivered)
	}
	if len(reported) != 2 {
		t.Fatalf("reported = %v, want two sessions", reported)
	}

	again, err := deliverer.DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("second DeliverOnce(): %v", err)
	}
	if again != 0 || len(reported) != 2 {
		t.Fatalf("second pass delivered = %d reported = %v, want 0 and 2", again, reported)
	}
}

// The session ending on its own must take its command with it. This is the path
// a disconnect takes, so without it every naturally-ended session leaves a row
// the deliverer keeps re-examining - and one that eventually gets delivered
// against a socket that closed minutes ago.
func TestCloseSessionClearsItsRevokeCommand(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-natural-end", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-natural-end", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}

	sessions := NewRemoteSessionService(db)
	if _, _, err := sessions.Close(context.Background(), "sig-natural-end", models.RemoteSessionEndDisconnected); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	all, err := store.All(context.Background())
	if err != nil {
		t.Fatalf("All(): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("commands after the session ended = %+v, want none", all)
	}

	// And the deliverer has nothing to do, which is the observable consequence.
	channel := &fakeRevokeChannel{}
	delivered, err := NewSessionRevokeCommandDeliverer(store, channel, nil).
		DeliverOnce(context.Background())
	if err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}
	if delivered != 0 || len(channel.published) != 0 {
		t.Fatalf("delivered = %d published = %+v, want nothing", delivered, channel.published)
	}
}

// A revoke that is decided while the session is still open must survive the
// session closing: that is the case the whole store exists for. The close here is
// the socket teardown the revoke itself triggers, which happens a moment after
// the command is marked delivered - so the command is already delivered by then
// and the clear is a no-op on it, which is why the row count matters rather than
// the delivered flag.
func TestCloseSessionKeepsADeliveredCommandReadable(t *testing.T) {
	db := newRevokeDeliveryTestDB(t)
	seedOpenSession(t, db, "sig-revoked", "inst-far")
	store := NewSessionRevokeCommandStore(db)
	if _, err := store.Record(context.Background(), "sig-revoked", "inst-far", "owner_revoke"); err != nil {
		t.Fatalf("Record(): %v", err)
	}

	channel := &fakeRevokeChannel{}
	deliverer := NewSessionRevokeCommandDeliverer(store, channel, nil)
	if _, err := deliverer.DeliverOnce(context.Background()); err != nil {
		t.Fatalf("DeliverOnce(): %v", err)
	}

	all, err := store.All(context.Background())
	if err != nil {
		t.Fatalf("All(): %v", err)
	}
	if len(all) != 1 || !all[0].Delivered {
		t.Fatalf("commands = %+v, want one delivered command still readable", all)
	}

	// The socket teardown that follows the revoke.
	if _, _, err := NewRemoteSessionService(db).Close(
		context.Background(), "sig-revoked", models.RemoteSessionEndRevoked); err != nil {
		t.Fatalf("Close(): %v", err)
	}
	all, err = store.All(context.Background())
	if err != nil {
		t.Fatalf("All() after close: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("commands after close = %+v, want the delivered command cleared", all)
	}
}
