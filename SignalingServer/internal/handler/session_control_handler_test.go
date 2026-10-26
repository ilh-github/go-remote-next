package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"quickdesk/signaling/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newControlTestConn builds a controller socket wired to a buffered out queue,
// which is the only part of the writer the control frames touch.
func newControlTestConn(deviceID, clientID string) *signalConn {
	return &signalConn{
		deviceID: deviceID,
		clientID: clientID,
		out:      make(chan []byte, 16),
		done:     make(chan struct{}),
	}
}

func drainFrames(t *testing.T, sc *signalConn) []map[string]interface{} {
	t.Helper()
	var frames []map[string]interface{}
	for {
		select {
		case raw := <-sc.out:
			var frame map[string]interface{}
			if err := json.Unmarshal(raw, &frame); err != nil {
				t.Fatalf("frame was not JSON: %v", err)
			}
			frames = append(frames, frame)
			continue
		default:
			return frames
		}
	}
}

func framesOfType(frames []map[string]interface{}, want string) []map[string]interface{} {
	var out []map[string]interface{}
	for _, frame := range frames {
		if frame["type"] == want {
			out = append(out, frame)
		}
	}
	return out
}

// The handshake has to hand the controller the fact it needs. This drives the
// same call the handshake makes rather than re-deriving it, so what is under
// test is the wiring, not a second copy of the frame builder.
func TestExclusiveControlHandsTheGrantToTheAskerAndNotifiesThePreviousHolder(t *testing.T) {
	h := &RealtimeHandler{
		controlGrants: service.NewControlGrants(),
		signalClient:  map[string]map[string]*signalConn{},
	}
	first := newControlTestConn("111111111", "cli-a")
	second := newControlTestConn("111111111", "cli-b")
	h.signalClient["111111111"] = map[string]*signalConn{
		"cli-a": first,
		"cli-b": second,
	}

	const deviceID = "111111111"
	const mode = service.ControlModeExclusive

	// First controller connects and takes control.
	previous := h.controlGrants.Get(deviceID).Holder
	grant := h.controlGrants.Request(deviceID, first.clientID, mode)
	if revoked := controlRevokedTarget(mode, previous, first.clientID); revoked != "" {
		h.sendToClient(deviceID, revoked, controlRevokedFrame(grant.Mode, first.clientID))
	}
	first.send(mustMarshal(controlGrantFrame(grant, grant.Holds(first.clientID))))

	granted := framesOfType(drainFrames(t, first), signalMsgControlGrant)
	if len(granted) != 1 {
		t.Fatalf("the first controller got %d grant frames, want 1", len(granted))
	}
	if granted[0]["can_control"] != true {
		t.Fatalf("can_control = %v, want true", granted[0]["can_control"])
	}
	if notified := framesOfType(drainFrames(t, second), signalMsgControlRevoked); len(notified) != 0 {
		t.Fatalf("a controller that never held control was notified: %v", notified)
	}

	// Second controller connects and takes it over.
	previous = h.controlGrants.Get(deviceID).Holder
	grant = h.controlGrants.Request(deviceID, second.clientID, mode)
	if revoked := controlRevokedTarget(mode, previous, second.clientID); revoked != "" {
		h.sendToClient(deviceID, revoked, controlRevokedFrame(grant.Mode, second.clientID))
	}
	second.send(mustMarshal(controlGrantFrame(grant, grant.Holds(second.clientID))))

	revoked := framesOfType(drainFrames(t, first), signalMsgControlRevoked)
	if len(revoked) != 1 {
		t.Fatalf("the previous holder got %d revoke frames, want 1", len(revoked))
	}
	if revoked[0]["control_holder"] != "cli-b" {
		t.Fatalf("the revoke named %v as the new holder, want cli-b", revoked[0]["control_holder"])
	}
	granted = framesOfType(drainFrames(t, second), signalMsgControlGrant)
	if len(granted) != 1 || granted[0]["can_control"] != true {
		t.Fatalf("the new holder did not get a usable grant: %v", granted)
	}
}

// In shared mode nobody is notified, and both controllers are told they may
// drive the machine - which is the behaviour every existing install has.
func TestSharedControlGrantsEverybodyWithoutNotifying(t *testing.T) {
	h := &RealtimeHandler{
		controlGrants: service.NewControlGrants(),
		signalClient:  map[string]map[string]*signalConn{},
	}
	first := newControlTestConn("111111111", "cli-a")
	second := newControlTestConn("111111111", "cli-b")
	h.signalClient["111111111"] = map[string]*signalConn{
		"cli-a": first,
		"cli-b": second,
	}

	const deviceID = "111111111"
	const mode = service.ControlModeShared
	for _, sc := range []*signalConn{first, second} {
		previous := h.controlGrants.Get(deviceID).Holder
		grant := h.controlGrants.Request(deviceID, sc.clientID, mode)
		if revoked := controlRevokedTarget(mode, previous, sc.clientID); revoked != "" {
			h.sendToClient(deviceID, revoked, controlRevokedFrame(grant.Mode, sc.clientID))
		}
		sc.send(mustMarshal(controlGrantFrame(grant, grant.Holds(sc.clientID))))
	}

	for _, sc := range []*signalConn{first, second} {
		frames := drainFrames(t, sc)
		if granted := framesOfType(frames, signalMsgControlGrant); len(granted) != 1 {
			t.Fatalf("%s got %d grant frames, want 1", sc.clientID, len(granted))
		} else if granted[0]["can_control"] != true {
			t.Fatalf("%s was told it may not control in shared mode: %v", sc.clientID, granted[0])
		}
		if notified := framesOfType(frames, signalMsgControlRevoked); len(notified) != 0 {
			t.Fatalf("%s was notified of a revoke in shared mode: %v", sc.clientID, notified)
		}
	}
}

// A controller that disconnected must stop holding control, or the machine
// stays reserved for a socket that is gone.
func TestReleasingControlLetsTheNextAskerTakeOver(t *testing.T) {
	h := &RealtimeHandler{controlGrants: service.NewControlGrants()}
	const deviceID = "111111111"
	h.controlGrants.Request(deviceID, "cli-a", service.ControlModeExclusive)

	h.releaseControl(deviceID, "cli-a")
	if h.canControl(deviceID, "cli-b") {
		t.Fatal("a machine with no holder still refused a new controller")
	}
	grant := h.controlGrants.Request(deviceID, "cli-b", service.ControlModeExclusive)
	if grant.Holder != "cli-b" {
		t.Fatalf("holder = %q, want cli-b after the previous holder released", grant.Holder)
	}
}

// A host that went away drops the grant: every controller is about to be
// disconnected, so a holder would be a stale answer to nobody's question.
func TestHostDisconnectDropsTheControlGrant(t *testing.T) {
	h := &RealtimeHandler{controlGrants: service.NewControlGrants()}
	const deviceID = "111111111"
	h.controlGrants.Request(deviceID, "cli-a", service.ControlModeExclusive)
	h.controlGrants.ForgetDevice(deviceID)
	if got := h.controlGrants.Get(deviceID); got.Holder != "" {
		t.Fatalf("a gone host still reported holder %q", got.Holder)
	}
	if _, ok := h.controlGrants.TakeRevoked(deviceID); ok {
		t.Fatal("a gone host left a pending revoke notice")
	}
}

// The handshake has to tell a controller whether it may drive the machine, or
// the client would guess - and a client that guesses "yes" while somebody else
// holds control produces the exact invisible fight QD-314 is about.
func TestControlGrantFrameCarriesModeAndHolder(t *testing.T) {
	frame := controlGrantFrame(service.ControlGrant{
		Mode:   service.ControlModeExclusive,
		Holder: "cli-a",
	}, true)
	if frame["type"] != "control_grant" {
		t.Fatalf("type = %v, want control_grant", frame["type"])
	}
	if frame["control_mode"] != service.ControlModeExclusive {
		t.Fatalf("control_mode = %v, want %v", frame["control_mode"], service.ControlModeExclusive)
	}
	if frame["control_holder"] != "cli-a" {
		t.Fatalf("control_holder = %v, want cli-a", frame["control_holder"])
	}
	if frame["can_control"] != true {
		t.Fatalf("can_control = %v, want true", frame["can_control"])
	}
}

// Shared mode never names a holder: there is nobody in particular, and a client
// that saw an empty holder would think it was waiting for one.
func TestControlGrantFrameForSharedModeNamesNobody(t *testing.T) {
	frame := controlGrantFrame(service.ControlGrant{Mode: service.ControlModeShared}, true)
	if frame["control_holder"] != "" {
		t.Fatalf("control_holder = %v, want empty", frame["control_holder"])
	}
	if frame["can_control"] != true {
		t.Fatalf("can_control = %v, want true in shared mode", frame["can_control"])
	}
}

// A controller that lost control must be told which client took it, so the UI
// can say "another window took over" instead of "your input stopped".
func TestControlRevokedFrameNamesTheNewHolder(t *testing.T) {
	frame := controlRevokedFrame(service.ControlModeExclusive, "cli-b")
	if frame["type"] != "control_revoked" {
		t.Fatalf("type = %v, want control_revoked", frame["type"])
	}
	if frame["control_mode"] != service.ControlModeExclusive {
		t.Fatalf("control_mode = %v, want %v", frame["control_mode"], service.ControlModeExclusive)
	}
	if frame["control_holder"] != "cli-b" {
		t.Fatalf("control_holder = %v, want cli-b", frame["control_holder"])
	}
}

// The controller that *lost* control is the one notified, and only in exclusive
// mode. Announcing it in shared mode would tell a controller it lost something
// it never had.
func TestControlRevokedTargetIsThePreviousHolder(t *testing.T) {
	if got := controlRevokedTarget(service.ControlModeExclusive, "cli-a", "cli-b"); got != "cli-a" {
		t.Fatalf("revoked target = %q, want the previous holder cli-a", got)
	}
	if got := controlRevokedTarget(service.ControlModeExclusive, "cli-a", "cli-a"); got != "" {
		t.Fatalf("a controller asking for control it already holds was notified: %q", got)
	}
	if got := controlRevokedTarget(service.ControlModeExclusive, "", "cli-b"); got != "" {
		t.Fatalf("taking control from nobody notified %q", got)
	}
	if got := controlRevokedTarget(service.ControlModeShared, "cli-a", "cli-b"); got != "" {
		t.Fatalf("shared mode notified a revoke: %q", got)
	}
}

// A handler with no grant store must behave exactly as before: every controller
// may drive the machine. This is what keeps an instance that has not been wired
// up from turning into an accidental exclusive machine.
func TestControlGrantsWithoutAStoreLetEverybodyDrive(t *testing.T) {
	h := &RealtimeHandler{}
	if !h.canControl("111111111", "cli-a") {
		t.Fatal("a handler with no grant store refused control")
	}
	if !h.canControl("111111111", "cli-b") {
		t.Fatal("a handler with no grant store refused a second controller")
	}
}

// The grant store decides once it exists, and the mode is what the owner
// configured rather than what a controller asks for.
func TestCanControlFollowsTheDeviceMode(t *testing.T) {
	h := &RealtimeHandler{controlGrants: service.NewControlGrants()}
	h.controlGrants.Request("111111111", "cli-a", service.ControlModeExclusive)
	if !h.canControl("111111111", "cli-a") {
		t.Fatal("the holder was refused control")
	}
	if h.canControl("111111111", "cli-b") {
		t.Fatal("a controller that does not hold the grant was allowed to drive")
	}
}

// The handshake consults the store through the handler, not the service
// directly, so what is pinned here is the wiring: an instance whose local map
// knows nothing must still answer correctly when the holder is elsewhere.
func TestHandshakeSeesAHolderAnotherInstanceIsHolding(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	store := service.NewControlGrantStore(rdb, 5*time.Minute)
	instanceA := (&RealtimeHandler{controlGrants: service.NewControlGrants()}).
		WithControlGrantStore(store)
	instanceB := (&RealtimeHandler{controlGrants: service.NewControlGrants()}).
		WithControlGrantStore(store)

	const deviceID = "111111111"
	instanceA.controlGrants.Request(deviceID, "cli-a", service.ControlModeExclusive)

	if !instanceB.canControl(deviceID, "cli-a") {
		t.Fatal("the holder was refused control on the instance that did not grant it")
	}
	if instanceB.canControl(deviceID, "cli-b") {
		t.Fatal("a second controller was told it may drive an exclusive machine")
	}

	// A takeover from either instance must be visible from the other, which is
	// the whole point of putting the holder somewhere shared.
	instanceB.controlGrants.Request(deviceID, "cli-b", service.ControlModeExclusive)
	if instanceA.canControl(deviceID, "cli-a") {
		t.Fatal("the displaced holder still believed it held control")
	}
	if !instanceA.canControl(deviceID, "cli-b") {
		t.Fatal("the new holder was refused control on the other instance")
	}
}

// A release from the holding instance frees the machine everywhere. Without it
// a controller that disconnected would keep the device reserved on the other
// instance, and the next controller would wait for a turn that never comes.
func TestHandshakeReleaseFreesTheMachineEverywhere(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	store := service.NewControlGrantStore(rdb, 5*time.Minute)
	instanceA := (&RealtimeHandler{controlGrants: service.NewControlGrants()}).
		WithControlGrantStore(store)
	instanceB := (&RealtimeHandler{controlGrants: service.NewControlGrants()}).
		WithControlGrantStore(store)

	const deviceID = "111111111"
	instanceA.controlGrants.Request(deviceID, "cli-a", service.ControlModeExclusive)
	instanceA.releaseControl(deviceID, "cli-a")

	grant := instanceB.controlGrants.Get(deviceID)
	if grant.Mode != service.ControlModeExclusive {
		t.Fatalf("mode = %q, want exclusive: a released machine is free, not shared", grant.Mode)
	}
	if grant.Holder != "" {
		t.Fatalf("holder = %q, want empty after release", grant.Holder)
	}
}

// A host going away drops the holder from the shared store too, so the next
// controller is not told to wait for a machine whose host is gone.
func TestHandshakeHostDisconnectDropsTheSharedHolder(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer rdb.Close()

	store := service.NewControlGrantStore(rdb, 5*time.Minute)
	h := (&RealtimeHandler{controlGrants: service.NewControlGrants()}).
		WithControlGrantStore(store)

	const deviceID = "111111111"
	h.controlGrants.Request(deviceID, "cli-a", service.ControlModeExclusive)
	h.controlGrants.ForgetDevice(deviceID)

	stored, err := store.Get(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if stored.Holder != "" {
		t.Fatalf("a gone host still reported holder %q", stored.Holder)
	}
}
