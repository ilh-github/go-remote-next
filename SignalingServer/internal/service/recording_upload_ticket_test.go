package service

import (
	"context"
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
