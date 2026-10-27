package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/redis/go-redis/v9"
)

// LiveInstances decides which open session rows are orphans, and the orphan
// sweep *closes* what it is told is not live. So an answer that is missing a
// live instance does not degrade gracefully - it stamps a working instance's
// sessions as finished, which is the exact accident the sweep's own comment says
// it was written to prevent ("a restart on one machine would stamp the other's
// live sessions as finished").
//
// The comment on LiveInstances calls itself "a liveness view, not a registry",
// and Redis documents SCAN as returning a possibly incomplete set with only a
// best-effort guarantee: the same key can come back twice, or not at all, before
// the cursor reaches zero. The loop treats "cursor == 0" as "the enumeration is
// over", which is what the implementation can observe, but it is not the same
// promise the caller needs.
//
// miniredis returns every matching key in one iteration, so these tests drive a
// peer-level hook that answers SCAN the way real Redis is allowed to.

// scanHook answers SCAN with the keys the test wants, in as many iterations as
// it wants. It is installed as a pre-hook, so it replaces the built-in SCAN
// rather than wrapping it - the point is to control what the caller sees, not to
// re-implement the enumeration.
//
// It only intercepts SCAN. SMEMBERS, SADD and EXISTS are miniredis's own, and
// exact: that asymmetry is the whole point, because it is the asymmetry real
// Redis has. The set is the exact part, the enumeration is the part Redis allows
// to be incomplete, and the fix is to stop using the incomplete one for the
// decision that closes other processes' sessions.
type scanHook struct {
	mu       sync.Mutex
	batches  [][]string // one entry per SCAN iteration
	calls    int
	commands []string
}

func (h *scanHook) hook(c *server.Peer, cmd string, args ...string) bool {
	if strings.ToUpper(cmd) != "SCAN" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commands = append(h.commands, cmd)
	batch := []string{}
	next := "0"
	if h.calls < len(h.batches) {
		batch = h.batches[h.calls]
	}
	// A non-zero cursor keeps the caller looping, exactly as real Redis does.
	if h.calls < len(h.batches)-1 {
		next = "17"
	}
	h.calls++
	c.WriteLen(2)
	c.WriteBulk(next)
	c.WriteLen(len(batch))
	for _, key := range batch {
		c.WriteBulk(key)
	}
	return true
}

func (h *scanHook) callCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls
}

func TestLiveInstancesReportsEveryInstanceASingleScanIterationMissed(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx := context.Background()

	for _, id := range []string{"inst-a", "inst-b"} {
		if err := NewPresenceService(redisClient, id).MarkAlive(ctx); err != nil {
			t.Fatalf("seed instance %s: %v", id, err)
		}
	}

	hook := &scanHook{batches: [][]string{
		{"qd:presence:instance:inst-a"},
		{"qd:presence:instance:inst-b"},
	}}
	redisServer.Server().SetPreHook(hook.hook)

	presence := NewPresenceService(redisClient, "inst-self")
	got, err := presence.LiveInstances(ctx)
	if err != nil {
		t.Fatalf("LiveInstances(): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("live instances = %v, want both, because both batches were delivered", got)
	}
}

// The failure this file exists for: an enumeration that ends while a live
// instance is still missing. Real Redis is allowed to return a key set that
// omits an existing key before the cursor reaches zero, and when that happens the
// sweep closes that instance's sessions.
func TestLiveInstancesDropsAnInstanceWhenTheScanEndsEarly(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx := context.Background()

	for _, id := range []string{"inst-a", "inst-b", "inst-c"} {
		if err := NewPresenceService(redisClient, id).MarkAlive(ctx); err != nil {
			t.Fatalf("seed instance %s: %v", id, err)
		}
	}

	// One batch, then a zero cursor: an enumeration that ended while inst-c was
	// still missing.
	hook := &scanHook{batches: [][]string{
		{"qd:presence:instance:inst-a", "qd:presence:instance:inst-b"},
	}}
	redisServer.Server().SetPreHook(hook.hook)

	presence := NewPresenceService(redisClient, "inst-self")
	got, err := presence.LiveInstances(ctx)
	if err != nil {
		t.Fatalf("LiveInstances(): %v", err)
	}

	missing := ""
	for _, id := range []string{"inst-a", "inst-b", "inst-c"} {
		found := false
		for _, key := range got {
			if strings.TrimPrefix(key, "qd:presence:instance:") == id {
				found = true
				break
			}
		}
		if !found {
			missing = id
		}
	}
	if missing != "" {
		t.Fatalf("live instances = %v, which omits %s - the sweep would close that instance's sessions", got, missing)
	}
}

// The sweep's guard: an empty live set must not be read as "everybody is dead".
// This is the one case where a short answer is safe, and it must stay that way,
// because the alternative is turning a Redis blip into an outage.
func TestLiveInstancesErrorKeepsTheSweepFromClosingEverything(t *testing.T) {
	db := newEventOutboxTestDB(t)
	svc := NewRemoteSessionService(db)
	ctx := context.Background()

	open := models.RemoteSession{
		SessionID:  "sess-live-instance",
		DeviceID:   "123456789",
		UserID:     7,
		ClientID:   "client-1",
		InstanceID: "inst-elsewhere",
		StartedAt:  time.Now().UTC(),
	}
	if err := db.Create(&open).Error; err != nil {
		t.Fatalf("seed open session: %v", err)
	}

	if _, err := svc.CloseOrphanedSessions(ctx, nil, models.RemoteSessionEndServerRestart); err == nil {
		t.Fatal("CloseOrphanedSessions with no live instances must refuse")
	}

	var stored models.RemoteSession
	if err := db.Where("session_id = ?", open.SessionID).First(&stored).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if stored.EndedAt != nil {
		t.Fatalf("session was closed despite the sweep refusing: %+v", stored)
	}
}

// The two halves together: an instance that a partial enumeration would have
// missed must survive the sweep. This is the regression test for the bug the
// previous version had - it failed with "closed = 1" before LiveInstances read a
// set instead of enumerating keys, and the assertion is deliberately the
// opposite of what the bug produced.
func TestOrphanSweepSparesAnInstanceTheScanWouldHaveMissed(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx := context.Background()

	for _, id := range []string{"inst-a", "inst-b"} {
		if err := NewPresenceService(redisClient, id).MarkAlive(ctx); err != nil {
			t.Fatalf("seed instance %s: %v", id, err)
		}
	}

	// SCAN is rigged to never mention inst-b, the way real Redis is allowed to
	// behave. inst-b's lease is present and its set entry is present; only the
	// enumeration is short.
	hook := &scanHook{batches: [][]string{{"qd:presence:instance:inst-a"}}}
	redisServer.Server().SetPreHook(hook.hook)

	presence := NewPresenceService(redisClient, "inst-self")
	live, err := presence.LiveInstances(ctx)
	if err != nil {
		t.Fatalf("LiveInstances(): %v", err)
	}
	found := false
	for _, id := range live {
		if id == "inst-b" {
			found = true
		}
	}
	if !found {
		t.Fatalf("live = %v, which omits inst-b even though it holds a lease - the sweep would close its sessions", live)
	}

	db := newEventOutboxTestDB(t)
	svc := NewRemoteSessionService(db)
	session := models.RemoteSession{
		SessionID:  "sess-on-inst-b",
		DeviceID:   "123456789",
		UserID:     7,
		ClientID:   "client-b",
		InstanceID: "inst-b",
		StartedAt:  time.Now().UTC(),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session on inst-b: %v", err)
	}

	closed, err := svc.CloseOrphanedSessions(ctx, live, models.RemoteSessionEndServerRestart)
	if err != nil {
		t.Fatalf("CloseOrphanedSessions: %v", err)
	}
	if closed != 0 {
		t.Fatalf("closed = %d, want 0 - the sweep ended a session belonging to a live instance", closed)
	}
	var stored models.RemoteSession
	if err := db.Where("session_id = ?", session.SessionID).First(&stored).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if stored.EndedAt != nil {
		t.Fatalf("a live instance's session was stamped as ended: %+v", stored)
	}
}

// A set entry whose lease has expired is not evidence of life. The read is what
// removes it, which is what keeps the set from growing without bound without
// needing a reaper - and what keeps a dead instance from being reported as live
// just because its entry is still there.
func TestLiveInstancesDropsMembersWhoseLeaseHasExpired(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx := context.Background()

	if err := NewPresenceService(redisClient, "inst-alive").MarkAlive(ctx); err != nil {
		t.Fatalf("seed live instance: %v", err)
	}
	// A second instance that joined and then died: its entry is still in the set,
	// its lease is not.
	if err := NewPresenceService(redisClient, "inst-dead").MarkAlive(ctx); err != nil {
		t.Fatalf("seed dead instance: %v", err)
	}
	redisServer.Del("qd:presence:instance:inst-dead")

	presence := NewPresenceService(redisClient, "inst-self")
	live, err := presence.LiveInstances(ctx)
	if err != nil {
		t.Fatalf("LiveInstances(): %v", err)
	}
	if len(live) != 1 || live[0] != "inst-alive" {
		t.Fatalf("live = %v, want only inst-alive", live)
	}

	// The read must have cleaned the set, not just filtered the answer: otherwise
	// the entry is re-examined on every read forever.
	members, err := redisClient.SMembers(ctx, presence.liveInstancesKey()).Result()
	if err != nil {
		t.Fatalf("SMEMBERS: %v", err)
	}
	if len(members) != 1 || members[0] != "inst-alive" {
		t.Fatalf("set after the read = %v, want only inst-alive", members)
	}
}

// The lease key and the set entry are written together, and both are what makes
// the read trustworthy. If the set outlived the key by more than a read, a
// crashed instance would be reported as live for as long as its entry survived -
// so the set carries the same lease, and this is the assertion that keeps them
// in step.
func TestMarkAliveWritesTheLeaseAndTheSetTogether(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	ctx := context.Background()

	presence := NewPresenceService(redisClient, "inst-self")
	if err := presence.MarkAlive(ctx); err != nil {
		t.Fatalf("MarkAlive: %v", err)
	}

	if ttl := redisServer.TTL(presence.instanceKey("inst-self")); ttl <= 0 {
		t.Fatalf("lease key TTL = %v, want the presence lease", ttl)
	}
	if ttl := redisServer.TTL(presence.liveInstancesKey()); ttl <= 0 {
		t.Fatalf("instance set TTL = %v, want it to expire with the leases it names", ttl)
	}
	members, err := redisClient.SMembers(ctx, presence.liveInstancesKey()).Result()
	if err != nil {
		t.Fatalf("SMEMBERS: %v", err)
	}
	if len(members) != 1 || members[0] != "inst-self" {
		t.Fatalf("set = %v, want [inst-self]", members)
	}

	// A second MarkAlive must not duplicate the entry, and must renew both TTLs.
	redisServer.FastForward(presenceInstanceTTL / 2)
	if err := presence.MarkAlive(ctx); err != nil {
		t.Fatalf("second MarkAlive: %v", err)
	}
	members, err = redisClient.SMembers(ctx, presence.liveInstancesKey()).Result()
	if err != nil {
		t.Fatalf("SMEMBERS after refresh: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("set after refresh = %v, want one entry", members)
	}
	if ttl := redisServer.TTL(presence.liveInstancesKey()); ttl <= presenceInstanceTTL/2 {
		t.Fatalf("set TTL after refresh = %v, want renewed", ttl)
	}
}
