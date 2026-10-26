package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"quickdesk/signaling/internal/service"

	"github.com/redis/go-redis/v9"
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

// controlFrameFor stamps the addressing a forwarded frame needs. It is the same
// frame the local path sends, plus the two fields that say which controller it
// belongs to; a local delivery simply ignores them.
//
// The type is preserved verbatim rather than rebuilt, so what crosses the
// instance boundary is byte-for-byte what a local controller would have seen.
func controlFrameFor(deviceID, clientID string, frame map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(frame)+2)
	for key, value := range frame {
		out[key] = value
	}
	out["device_id"] = deviceID
	out["client_id"] = clientID
	return out
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

// WithControlGrantStore gives the per-instance grants a cross-instance backing.
//
// The store is optional: a single-instance install keeps the in-process map as
// the whole truth, and a fleet needs the holder somewhere every instance can
// read - two instances each answering "you may drive it" is the invisible fight
// exclusive mode exists to prevent.
func (h *RealtimeHandler) WithControlGrantStore(store *service.ControlGrantStore) *RealtimeHandler {
	if h == nil {
		return h
	}
	if h.controlGrants == nil {
		h.controlGrants = service.NewControlGrants()
	}
	h.controlGrants.WithShared(store)
	return h
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

// sendToClient delivers one frame to a single controller, wherever its socket
// happens to live.
//
// The takeover notice is the whole point of QD-314: a holder that is never told
// it lost control keeps typing into a machine it no longer owns. In a fleet the
// displaced holder is frequently on *another* instance - the asker connected to
// a different process - so delivering only to local sockets would drop exactly
// the notice the feature exists to send. The frame is therefore published to the
// instance that owns the socket, on the same addressed channel the revoke bus
// already uses; a controller that is not connected anywhere simply gets nothing,
// which is correct because there is nobody left to tell.
func (h *RealtimeHandler) sendToClient(deviceID, clientID string, frame map[string]interface{}) {
	if h == nil || frame == nil {
		return
	}
	h.signalMu.RLock()
	target := h.signalClient[deviceID][clientID]
	h.signalMu.RUnlock()
	if target != nil {
		target.send(mustMarshal(frame))
		return
	}
	h.forwardFrameToOwningInstance(context.Background(), deviceID, clientID,
		controlFrameFor(deviceID, clientID, frame))
}

// forwardFrameToOwningInstance publishes one control frame to the instance
// holding a controller's socket. Silently does nothing when there is no bus or
// no way to know which instance that is: the caller is announcing a fact the
// controller can also learn from its own next handshake, so a missed notice must
// never break the takeover itself.
func (h *RealtimeHandler) forwardFrameToOwningInstance(
	ctx context.Context,
	deviceID, clientID string,
	frame map[string]interface{},
) {
	if h.controlFrameBus == nil || h.rdb == nil {
		return
	}
	instanceID, err := socketInstanceOf(ctx, h.rdb, deviceID, clientID)
	if err != nil || instanceID == "" {
		return
	}
	if err := h.controlFrameBus.Publish(ctx, instanceID, mustMarshal(frame)); err != nil {
		log.Printf("[control-frame] publish to instance %s failed: %v", instanceID, err)
	}
}

// socketInstanceKey is the fleet-wide index of which instance holds a given
// controller's socket. Presence already tracks instances per *device*, which is
// the right granularity for "is this machine online" and the wrong one here: a
// takeover notice has to reach one specific controller, and several instances
// can hold sockets for the same device at once.
func socketInstanceKey(deviceID, clientID string) string {
	return "qd:sessions:control:socket:" + strings.TrimSpace(deviceID) + ":" + strings.TrimSpace(clientID)
}

// socketInstanceOf reads that index. An empty answer means "not connected
// anywhere this instance knows about", which is a legitimate state for a
// controller that has already gone away.
func socketInstanceOf(ctx context.Context, rdb *redis.Client, deviceID, clientID string) (string, error) {
	if rdb == nil {
		return "", nil
	}
	value, err := rdb.Get(ctx, socketInstanceKey(deviceID, clientID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// rememberSocketInstance records which instance holds a controller's socket, and
// forgetSocketInstance drops it. Both are best effort: the index only decides
// where a takeover notice is delivered, and a stale entry costs one wasted
// publish, never a wrong grant.
func rememberSocketInstance(ctx context.Context, rdb *redis.Client, deviceID, clientID, instanceID string) {
	if rdb == nil {
		return
	}
	if err := rdb.Set(ctx, socketInstanceKey(deviceID, clientID), instanceID, socketInstanceTTL).Err(); err != nil {
		log.Printf("[control-frame] remember socket for %s/%s failed: %v", deviceID, clientID, err)
	}
}

func forgetSocketInstance(ctx context.Context, rdb *redis.Client, deviceID, clientID, instanceID string) {
	if rdb == nil {
		return
	}
	// Only clear the entry if it still names this instance: a controller that
	// reconnected elsewhere must not have the new instance's record deleted by
	// the old socket's teardown.
	key := socketInstanceKey(deviceID, clientID)
	current, err := rdb.Get(ctx, key).Result()
	if err != nil || strings.TrimSpace(current) != strings.TrimSpace(instanceID) {
		return
	}
	if err := rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("[control-frame] forget socket for %s/%s failed: %v", deviceID, clientID, err)
	}
}

// socketInstanceTTL bounds a stale index entry. It is only a ceiling on debris:
// a socket cannot outlive its process, so this is not a promise the controller
// is still connected, and the teardown deletes the key long before it expires.
const socketInstanceTTL = 10 * time.Minute

// WithControlFrameBus attaches the cross-instance frame channel and starts this
// instance's subscription.
func (h *RealtimeHandler) WithControlFrameBus(ctx context.Context, rdb *redis.Client) *RealtimeHandler {
	if h == nil || rdb == nil {
		return h
	}
	h.controlFrameBus = NewControlFrameBus(rdb, h.InstanceID(), func(payload []byte) {
		h.deliverFrameToLocalClient(payload)
	})
	go h.controlFrameBus.Run(ctx)
	return h
}

// deliverFrameToLocalClient routes a frame that arrived from another instance to
// the controller it names. An unknown client is dropped: the frame was addressed
// here because the index said its socket was here, and a socket that is gone
// means there is nobody left to tell.
func (h *RealtimeHandler) deliverFrameToLocalClient(payload []byte) {
	var frame map[string]interface{}
	if err := json.Unmarshal(payload, &frame); err != nil {
		log.Printf("[control-frame] dropping malformed frame: %v", err)
		return
	}
	deviceID, _ := frame["device_id"].(string)
	clientID, _ := frame["client_id"].(string)
	if deviceID == "" || clientID == "" {
		return
	}
	h.signalMu.RLock()
	target := h.signalClient[deviceID][clientID]
	h.signalMu.RUnlock()
	if target == nil {
		return
	}
	target.send(payload)
}
