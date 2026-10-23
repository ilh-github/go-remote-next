package handler

import (
	"context"

	"quickdesk/signaling/internal/service"
)

// Control frames on the signaling channel.
//
// Input itself travels over the WebRTC DataChannel, peer to peer, so this
// server cannot drop a byte a controller should not have sent. What it can do
// is be the single authority on who holds control and say so out loud: the
// holder gets `control_grant`, everybody else who just lost it gets
// `control_revoked`. The WebClient enforces the answer locally, the same way it
// already settles same-origin tabs.
const (
	signalMsgControlGrant   = "control_grant"
	signalMsgControlRevoked = "control_revoked"
)

// controlGrantFrame is what the handshake answers with: the device's mode, who
// holds control in exclusive mode, and whether this particular controller may
// drive the machine.
func controlGrantFrame(grant service.ControlGrant, canControl bool) map[string]interface{} {
	return map[string]interface{}{
		"type":           signalMsgControlGrant,
		"control_mode":   grant.Mode,
		"control_holder": grant.Holder,
		"can_control":    canControl,
	}
}

// controlRevokedFrame tells a controller it lost control and who took it, so
// the UI can name the new holder instead of silently dropping input.
func controlRevokedFrame(mode, holder string) map[string]interface{} {
	return map[string]interface{}{
		"type":           signalMsgControlRevoked,
		"control_mode":   mode,
		"control_holder": holder,
	}
}

// controlRevokedTarget returns the client that must be told it lost control, or
// "" when nobody did.
//
// Only exclusive mode has a holder to lose, and a controller that asks for a
// grant it already holds has not taken anything from anybody. The frame sent to
// the target names the new holder, which is the asker.
func controlRevokedTarget(mode, previousHolder, asker string) string {
	if mode != service.ControlModeExclusive {
		return ""
	}
	if previousHolder == "" || previousHolder == asker {
		return ""
	}
	return previousHolder
}

// canControl reports whether clientID may drive deviceID right now. A handler
// with no grant store answers yes for everybody: that is the behaviour every
// existing install has, and it must survive an instance that was never wired up.
func (h *RealtimeHandler) canControl(deviceID, clientID string) bool {
	if h == nil || h.controlGrants == nil {
		return true
	}
	return h.controlGrants.Get(deviceID).Holds(clientID)
}

// WithControlGrants attaches the per-instance control store.
func (h *RealtimeHandler) WithControlGrants(grants *service.ControlGrants) *RealtimeHandler {
	if h == nil {
		return h
	}
	h.controlGrants = grants
	return h
}

// releaseControl gives up control when a controller's socket goes away. Without
// it a holder that disconnected would keep the machine: the next controller
// would take over only by asking, and a client that waits for its turn instead
// of asking would wait forever.
func (h *RealtimeHandler) releaseControl(deviceID, clientID string) {
	if h == nil || h.controlGrants == nil {
		return
	}
	h.controlGrants.Release(deviceID, clientID)
}

// controlMode reads the device's configured mode. An unreadable or absent
// policy means shared: control of a machine nobody configured must not change
// behaviour because the server could not read a row.
func (h *RealtimeHandler) controlMode(ctx context.Context, deviceID string) string {
	if h == nil || h.accessPolicy == nil {
		return service.ControlModeShared
	}
	policy, err := h.accessPolicy.Get(ctx, deviceID)
	if err != nil || policy == nil {
		return service.ControlModeShared
	}
	return service.NormalizeControlMode(policy.ControlMode)
}

// sendToClient delivers one frame to a single controller on this instance.
// Nothing is sent when that controller is not connected here: the caller is
// announcing a takeover, and a controller on another instance has its own
// process's grants to consult.
func (h *RealtimeHandler) sendToClient(deviceID, clientID string, frame map[string]interface{}) {
	if h == nil || frame == nil {
		return
	}
	h.signalMu.RLock()
	target := h.signalClient[deviceID][clientID]
	h.signalMu.RUnlock()
	if target == nil {
		return
	}
	target.send(mustMarshal(frame))
}
