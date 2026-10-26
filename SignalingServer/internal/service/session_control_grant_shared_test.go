package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// QD-314: exclusive control has to survive more than one signaling instance.
//
// The per-instance store is a map in one process, so two instances holding two
// controllers on the same exclusive machine would each answer "you may drive
// it" - the invisible fight the feature exists to prevent. These tests pin the
// shared store, which is what the handshake consults once a fleet exists.
func TestSharedControlGrantsSeeAnotherInstanceHolder(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	instanceA := NewControlGrants().WithShared(NewControlGrantStore(rdb, 5*time.Minute))
	instanceB := NewControlGrants().WithShared(NewControlGrantStore(rdb, 5*time.Minute))

	if got := instanceA.Request("111111111", "cli-a", ControlModeExclusive); got.Holder != "cli-a" {
		t.Fatalf("instance A holder = %q, want cli-a", got.Holder)
	}

	// B never saw the request. Reading through the shared store is the only way
	// it can know somebody already holds the machine.
	if got := instanceB.Get("111111111"); got.Holder != "cli-a" {
		t.Fatalf("instance B holder = %q, want cli-a - the holder was invisible across instances", got.Holder)
	}
	// Holds is the predicate the handshake's canControl calls: in exclusive mode
	// only the holder may drive the machine.
	if instanceB.Get("111111111").Holds("cli-b") {
		t.Fatal("instance B let a second controller drive an exclusive machine")
	}

	// The takeover is a write to the same key, so both instances agree afterwards.
	if got := instanceB.Request("111111111", "cli-b", ControlModeExclusive); got.Holder != "cli-b" {
		t.Fatalf("instance B holder after takeover = %q, want cli-b", got.Holder)
	}
	if got := instanceA.Get("111111111"); got.Holder != "cli-b" {
		t.Fatalf("instance A holder after takeover = %q, want cli-b", got.Holder)
	}
}

// A release from the holding instance must free the machine for everybody, or a
// controller that disconnected keeps the device reserved on the other instance.
func TestSharedControlGrantsReleaseCrossesInstances(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	instanceA := NewControlGrants().WithShared(NewControlGrantStore(rdb, 5*time.Minute))
	instanceB := NewControlGrants().WithShared(NewControlGrantStore(rdb, 5*time.Minute))

	instanceA.Request("111111111", "cli-a", ControlModeExclusive)
	instanceA.Release("111111111", "cli-a")

	got := instanceB.Get("111111111")
	if got.Mode != ControlModeExclusive {
		t.Fatalf("mode = %q, want exclusive: a released machine is free, not shared", got.Mode)
	}
	if got.Holder != "" {
		t.Fatalf("holder = %q, want empty after release", got.Holder)
	}
}

// The mode must travel with the holder. A device whose policy says exclusive is
// still exclusive after a Redis restart wiped the holder, and reading it as
// shared would hand control to whoever asked next.
func TestSharedControlGrantsKeepsTheModeWithoutAHolder(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	store := NewControlGrantStore(rdb, 5*time.Minute)
	instance := NewControlGrants().WithShared(store)

	instance.Request("111111111", "cli-a", ControlModeExclusive)
	instance.Release("111111111", "cli-a")

	got, err := store.Get(context.Background(), "111111111")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if got.Mode != ControlModeExclusive || got.Holder != "" {
		t.Fatalf("grant = %+v, want exclusive with no holder", got)
	}
}

// Without Redis the behaviour is exactly what it was: per-instance, and every
// controller may drive the machine. An install that never configured a shared
// store must not become an accidental exclusive machine.
func TestControlGrantsWithoutASharedStoreBehavePerInstance(t *testing.T) {
	instance := NewControlGrants()
	instance.Request("111111111", "cli-a", ControlModeExclusive)

	other := NewControlGrants()
	// An unread device reads as the zero value, which is why callers normalize:
	// "" means "nobody recorded a mode", not "shared". Normalizing is what the
	// handshake's controlMode does, so that is the path this test exercises.
	if got := NormalizeControlMode(other.Get("111111111").Mode); got != ControlModeShared {
		t.Fatalf("an instance with no shared store reported mode %q, want shared", got)
	}
	if !other.Get("111111111").Holds("cli-b") {
		t.Fatal("an instance with no shared store refused control")
	}
}
