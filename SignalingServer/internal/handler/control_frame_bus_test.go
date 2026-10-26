package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// QD-314, cross-instance half: a takeover notice has to reach a controller whose
// socket lives in another process.
//
// The single-instance tests pin the frames themselves. These pin the delivery:
// the displaced holder is usually on the instance the asker did *not* connect
// to, so delivering only to local sockets drops exactly the notice the feature
// exists to send - and the holder keeps typing into a machine it no longer owns.
func TestControlFrameBusDeliversToTheNamedInstance(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	// A channel rather than a bare variable: the callback runs on the bus's
	// subscriber goroutine, and reading a plain variable from the test body is a
	// data race that the race detector is right to fail.
	received := make(chan []byte, 1)
	bus := NewControlFrameBus(rdb, "instance-b", func(payload []byte) {
		select {
		case received <- payload:
		default:
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)

	// The subscription is accepted in runOnce before it starts reading, but the
	// goroutine has to get there; publishing immediately can race it. Waiting for
	// the subscriber count is how the test stops depending on that scheduling.
	waitForSubscriber(t, rdb, bus.channel("instance-b"))

	frame := controlFrameFor("111111111", "cli-a", controlRevokedFrame("exclusive", "cli-b"))
	if err := bus.Publish(ctx, "instance-b", mustMarshal(frame)); err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	var got []byte
	select {
	case got = <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("no frame arrived on the addressed instance")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("frame was not JSON: %v", err)
	}
	if decoded["type"] != signalMsgControlRevoked {
		t.Fatalf("type = %v, want %s", decoded["type"], signalMsgControlRevoked)
	}
	// The addressing is what lets the receiving instance route it; a frame
	// without it would be delivered to nobody.
	if decoded["client_id"] != "cli-a" {
		t.Fatalf("client_id = %v, want cli-a", decoded["client_id"])
	}
	if decoded["device_id"] != "111111111" {
		t.Fatalf("device_id = %v, want 111111111", decoded["device_id"])
	}
}

// A frame addressed to a different instance must not be delivered here. The
// channel is addressed rather than broadcast precisely so a takeover on one
// device does not notify controllers of every other device.
func TestControlFrameBusIgnoresFramesForOtherInstances(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	delivered := make(chan []byte, 4)
	bus := NewControlFrameBus(rdb, "instance-a", func(payload []byte) {
		delivered <- payload
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bus.Run(ctx)
	waitForSubscriber(t, rdb, bus.channel("instance-a"))

	other := NewControlFrameBus(rdb, "instance-b", func([]byte) {})
	if err := other.Publish(ctx, "instance-b", mustMarshal(controlFrameFor("111111111", "cli-a",
		controlRevokedFrame("exclusive", "cli-b")))); err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	select {
	case payload := <-delivered:
		t.Fatalf("a frame addressed elsewhere was delivered here: %s", payload)
	case <-time.After(300 * time.Millisecond):
	}
}

// The socket index answers "which instance holds this controller". Without it a
// forwarded frame has nowhere to go, and the takeover is silent again.
func TestSocketInstanceIndexTracksTheHoldingInstance(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()
	ctx := context.Background()

	rememberSocketInstance(ctx, rdb, "111111111", "cli-a", "instance-a")
	got, err := socketInstanceOf(ctx, rdb, "111111111", "cli-a")
	if err != nil {
		t.Fatalf("socketInstanceOf(): %v", err)
	}
	if got != "instance-a" {
		t.Fatalf("instance = %q, want instance-a", got)
	}

	// A controller that is not connected anywhere is a legitimate answer, not an
	// error: the socket is simply gone.
	got, err = socketInstanceOf(ctx, rdb, "111111111", "cli-gone")
	if err != nil {
		t.Fatalf("socketInstanceOf() for an unknown client: %v", err)
	}
	if got != "" {
		t.Fatalf("instance = %q, want empty", got)
	}
}

// Forgetting must only clear the entry that still names this instance. A
// controller that reconnected elsewhere must not have the new instance's record
// deleted by the old socket's teardown, or the next takeover notice goes to a
// process that no longer holds the socket.
func TestForgetSocketInstanceOnlyClearsItsOwnEntry(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()
	ctx := context.Background()

	rememberSocketInstance(ctx, rdb, "111111111", "cli-a", "instance-a")
	rememberSocketInstance(ctx, rdb, "111111111", "cli-a", "instance-b")

	// instance-a's stale teardown must not remove instance-b's record.
	forgetSocketInstance(ctx, rdb, "111111111", "cli-a", "instance-a")
	got, err := socketInstanceOf(ctx, rdb, "111111111", "cli-a")
	if err != nil {
		t.Fatalf("socketInstanceOf(): %v", err)
	}
	if got != "instance-b" {
		t.Fatalf("instance = %q, want instance-b: a stale teardown cleared a live record", got)
	}

	forgetSocketInstance(ctx, rdb, "111111111", "cli-a", "instance-b")
	got, err = socketInstanceOf(ctx, rdb, "111111111", "cli-a")
	if err != nil {
		t.Fatalf("socketInstanceOf(): %v", err)
	}
	if got != "" {
		t.Fatalf("instance = %q, want empty after the owning instance forgot it", got)
	}
}

// A handler with no bus must behave exactly as before: local delivery only. An
// install that never configured Redis cannot start failing a takeover.
func TestSendToClientWithoutABusDeliversLocallyOnly(t *testing.T) {
	h := &RealtimeHandler{signalClient: map[string]map[string]*signalConn{}}
	sc := newControlTestConn("111111111", "cli-a")
	h.signalClient["111111111"] = map[string]*signalConn{"cli-a": sc}

	h.sendToClient("111111111", "cli-a", controlRevokedFrame("exclusive", "cli-b"))

	frames := drainFrames(t, sc)
	revoked := framesOfType(frames, signalMsgControlRevoked)
	if len(revoked) != 1 {
		t.Fatalf("got %d revoke frames, want 1", len(revoked))
	}
	if revoked[0]["control_holder"] != "cli-b" {
		t.Fatalf("control_holder = %v, want cli-b", revoked[0]["control_holder"])
	}
}

// The local path still works with a bus attached, and the frame a local
// controller sees is unchanged - the addressing is only added when the frame has
// to cross an instance boundary.
func TestSendToClientWithABusStillDeliversLocallyUnchanged(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	h := (&RealtimeHandler{signalClient: map[string]map[string]*signalConn{}}).
		WithControlFrameBus(context.Background(), rdb)
	sc := newControlTestConn("111111111", "cli-a")
	h.signalClient["111111111"] = map[string]*signalConn{"cli-a": sc}

	h.sendToClient("111111111", "cli-a", controlRevokedFrame("exclusive", "cli-b"))

	frames := drainFrames(t, sc)
	revoked := framesOfType(frames, signalMsgControlRevoked)
	if len(revoked) != 1 {
		t.Fatalf("got %d revoke frames, want 1", len(revoked))
	}
	if _, ok := revoked[0]["client_id"]; ok {
		t.Fatalf("a local frame carried routing fields: %v", revoked[0])
	}
}

func waitForSubscriber(t *testing.T, rdb *redis.Client, channel string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		count, err := rdb.PubSubNumSub(context.Background(), channel).Result()
		if err == nil && count[channel] > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no subscriber ever appeared on %s", channel)
}
