package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// QD-004 的上传票据与播放票据不同：播放票据可以重放（用户反复看同一个视频），
// 上传票据必须一次性——否则拿到链接的人可以反复覆盖同一个录像文件。
// 一次性需要服务端状态，因此这条用例只验证绑定与签名，一次性由下面的
// Redis 用例覆盖；无 Redis 的降级行为也有显式用例，不被当成安全保证。
func TestUploadTicketBindingAndSignature(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	ticket, _, err := store.Issue(4242, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}

	first, err := store.Consume(context.Background(), ticket)
	if err != nil {
		t.Fatalf("first Consume(): %v", err)
	}
	if first.RecordingID != 4242 || first.UserID != 7 {
		t.Fatalf("ticket binding = %d/%d, want 4242/7", first.RecordingID, first.UserID)
	}
	if first.Nonce == "" {
		t.Fatal("a redeemed ticket must carry its nonce")
	}
}

// A ticket for one recording must not be replayable against another, even
// though both belong to the same user.
func TestUploadTicketIsBoundToOneRecording(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	first, _, err := store.Issue(100, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(first): %v", err)
	}
	second, _, err := store.Issue(200, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(second): %v", err)
	}

	consumed, err := store.Consume(context.Background(), first)
	if err != nil {
		t.Fatalf("Consume(first): %v", err)
	}
	if consumed.RecordingID != 100 {
		t.Fatalf("recording = %d, want 100", consumed.RecordingID)
	}

	// The other ticket is untouched and still carries its own binding.
	other, err := store.Consume(context.Background(), second)
	if err != nil {
		t.Fatalf("Consume(second): %v", err)
	}
	if other.RecordingID != 200 {
		t.Fatalf("recording = %d, want 200", other.RecordingID)
	}
}

// An expired ticket must be refused rather than consumed, so a leaked link
// stops working on its own.
func TestUploadTicketExpires(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	ticket, expiresAt, err := store.Issue(4242, 7, -time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}
	if !expiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expires_at = %s, want a past instant", expiresAt)
	}
	if _, err := store.Consume(context.Background(), ticket); err == nil {
		t.Fatal("Consume() succeeded on an expired ticket")
	}
}

// A tampered ticket must be refused. The signature is what stops a caller
// from minting their own upload ticket for someone else's recording.
func TestUploadTicketRejectsTampering(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	ticket, _, err := store.Issue(4242, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}
	parts := strings.SplitN(ticket, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("ticket = %q, want body.signature", ticket)
	}

	if _, err := store.Consume(context.Background(), parts[0]+".AAAA"); err == nil {
		t.Fatal("Consume() succeeded on a tampered ticket")
	}
}

// A ticket signed with a different secret must not verify here.
func TestUploadTicketRejectsAForeignSecret(t *testing.T) {
	ours := NewRecordingUploadTicketStore(nil, "our-secret")
	ticket, _, err := NewRecordingUploadTicketStore(nil, "their-secret").Issue(4242, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}
	if _, err := ours.Consume(context.Background(), ticket); err == nil {
		t.Fatal("Consume() succeeded on a ticket signed with another secret")
	}
}
// Without Redis there is no state to remember a redemption, so the store must
// say so rather than letting a caller believe it issued a one-shot ticket.
func TestUploadTicketStoreReportsWhenItIsNotSingleUse(t *testing.T) {
	if NewRecordingUploadTicketStore(nil, "unit-secret").SingleUse() {
		t.Fatal("a store without Redis must not claim to be single-use")
	}
}

// The real one-shot behaviour needs the Redis marker, so it is tested against
// a real (in-process) server rather than assumed.
func TestUploadTicketIsSingleUseWithRedis(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRecordingUploadTicketStore(client, "unit-secret")
	if !store.SingleUse() {
		t.Fatal("a store with Redis must report single-use")
	}
	ticket, _, err := store.Issue(4242, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}

	if _, err := store.Consume(context.Background(), ticket); err != nil {
		t.Fatalf("first Consume(): %v", err)
	}
	_, err = store.Consume(context.Background(), ticket)
	if !errors.Is(err, ErrUploadTicketConsumed) {
		t.Fatalf("second Consume() error = %v, want ErrUploadTicketConsumed", err)
	}
}

// The marker must outlive the ticket, so a link replayed after it expired is
// still refused for the right reason rather than silently accepted as fresh.
//
// The check compares the marker's own TTL against the ticket's expiry instead
// of fast-forwarding past both: once the ticket has expired the signature
// check refuses it anyway, which would prove nothing about the marker.
func TestUploadTicketMarkerOutlivesTheTicket(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	store := NewRecordingUploadTicketStore(client, "unit-secret")
	ticket, _, err := store.Issue(4242, 7, time.Minute)
	if err != nil {
		t.Fatalf("Issue(): %v", err)
	}
	redeemed, err := store.Consume(context.Background(), ticket)
	if err != nil {
		t.Fatalf("first Consume(): %v", err)
	}

	markerTTL := client.TTL(context.Background(), uploadTicketKey(redeemed.Nonce)).Val()
	if markerTTL <= time.Until(redeemed.ExpiresAt) {
		t.Fatalf("marker TTL = %s, want longer than the ticket's remaining %s",
			markerTTL, time.Until(redeemed.ExpiresAt))
	}
}

// Issuance is disabled without a secret, matching the playback ticket.
func TestUploadTicketIsDisabledWithoutASecret(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "")
	if _, _, err := store.Issue(4242, 7, time.Minute); err == nil {
		t.Fatal("Issue() succeeded without a configured secret")
	}
	if _, err := store.Consume(context.Background(), "anything"); err == nil {
		t.Fatal("Consume() succeeded without a configured secret")
	}
}

// A ticket must name both a recording and a user; a zero on either side is a
// caller bug rather than a credential to hand out.
func TestUploadTicketRejectsUnboundIdentifiers(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	if _, _, err := store.Issue(0, 7, time.Minute); err == nil {
		t.Fatal("Issue() succeeded without a recording id")
	}
	if _, _, err := store.Issue(4242, 0, time.Minute); err == nil {
		t.Fatal("Issue() succeeded without a user id")
	}
}
// The session binding is what stops a Host from being handed a ticket for one
// recording and pushing another session's media under it.
func TestUploadTicketCarriesItsSessionBinding(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	ticket, _, err := store.IssueForSession(4242, 7, "sess-alpha", time.Minute)
	if err != nil {
		t.Fatalf("IssueForSession(): %v", err)
	}

	redeemed, err := store.Consume(context.Background(), ticket)
	if err != nil {
		t.Fatalf("Consume(): %v", err)
	}
	if redeemed.SessionID != "sess-alpha" {
		t.Fatalf("session = %q, want sess-alpha", redeemed.SessionID)
	}
	if !redeemed.MatchesSession("sess-alpha") {
		t.Fatal("a ticket must match the session it was minted for")
	}
	if redeemed.MatchesSession("sess-beta") {
		t.Fatal("a ticket must not match a different session")
	}
}

// A recording created without a session cannot be cross-checked, so a
// session-bound ticket is still accepted for it. Making those recordings
// permanently unuploadable would be a worse outcome than the gap.
func TestSessionBindingYieldsWhenEitherSideHasNoSession(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	bound, _, err := store.IssueForSession(4242, 7, "sess-alpha", time.Minute)
	if err != nil {
		t.Fatalf("IssueForSession(bound): %v", err)
	}
	unbound, _, err := store.IssueForSession(4243, 7, "", time.Minute)
	if err != nil {
		t.Fatalf("IssueForSession(unbound): %v", err)
	}

	redeemedBound, err := store.Consume(context.Background(), bound)
	if err != nil {
		t.Fatalf("Consume(bound): %v", err)
	}
	redeemedUnbound, err := store.Consume(context.Background(), unbound)
	if err != nil {
		t.Fatalf("Consume(unbound): %v", err)
	}

	if !redeemedBound.MatchesSession("") {
		t.Fatal("a session-bound ticket must still cover a session-less recording")
	}
	if !redeemedUnbound.MatchesSession("sess-alpha") {
		t.Fatal("an unbound ticket must not be refused for a session-less recording")
	}
	// A nil ticket never matches.
	var missing *RecordingUploadTicket
	if missing.MatchesSession("sess-alpha") {
		t.Fatal("a nil ticket must not match any session")
	}
}

// A ticket signed for one session must not be replayable as another: the
// session travels inside the signed payload, so editing it breaks the
// signature rather than silently widening the ticket.
func TestUploadTicketSessionCannotBeTamperedWith(t *testing.T) {
	store := NewRecordingUploadTicketStore(nil, "unit-secret")
	ticket, _, err := store.IssueForSession(4242, 7, "sess-alpha", time.Minute)
	if err != nil {
		t.Fatalf("IssueForSession(): %v", err)
	}
	parts := strings.SplitN(ticket, ".", 2)
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	forged := base64.RawURLEncoding.EncodeToString(
		[]byte(strings.Replace(string(raw), "sid=sess-alpha", "sid=sess-beta", 1)),
	) + "." + parts[1]

	if _, err := store.Consume(context.Background(), forged); err == nil {
		t.Fatal("Consume() succeeded on a ticket whose session was edited")
	}
}
