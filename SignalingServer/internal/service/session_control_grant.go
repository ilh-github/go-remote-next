package service

import (
	"errors"
	"strings"
	"sync"
)

// How several controllers on one device share control of it.
const (
	// ControlModeShared is the default and the historical behaviour: every
	// connected controller may drive the machine, and two people fighting over
	// the mouse is the owner's problem, not the server's.
	ControlModeShared = "shared"
	// ControlModeExclusive gives control to one controller at a time. A second
	// controller that asks takes it over, and the previous holder is told it
	// lost it.
	ControlModeExclusive = "exclusive"
)

// ErrInvalidControlMode is returned for anything that is not one of the two
// modes above, including the empty string.
var ErrInvalidControlMode = errors.New("control_mode must be shared|exclusive")

// NormalizeControlMode maps an unset mode to the default. An unknown non-empty
// value is left alone so the caller's validation can reject it instead of
// silently picking a mode the owner did not ask for.
func NormalizeControlMode(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return ControlModeShared
	}
	return mode
}

// ValidateControlMode rejects anything but the two known modes, whitespace
// included: trimming here would make `"exclusive "` a valid mode, and a client
// that round-trips the value through a form would then store padded data.
// Callers that accept operator input trim before calling.
func ValidateControlMode(mode string) error {
	switch mode {
	case ControlModeShared, ControlModeExclusive:
		return nil
	default:
		return ErrInvalidControlMode
	}
}

// ControlGrant is who may drive one device right now.
type ControlGrant struct {
	// Mode is the device's configured mode at the time of the grant.
	Mode string
	// Holder is the client_id holding control in exclusive mode. Empty in
	// shared mode, where nobody holds it in particular.
	Holder string
}

// Holds reports whether clientID may send input right now.
func (g ControlGrant) Holds(clientID string) bool {
	if g.Mode != ControlModeExclusive {
		return true
	}
	return g.Holder == clientID
}

// ControlGrants tracks control per device for one signaling instance.
//
// The media path is peer to peer, so this server never sees the input bytes and
// cannot drop them: a controller that ignores the grant would still reach the
// host. What the server can do is be the one authority on who holds control and
// tell every other controller that it does not, which is what makes a takeover
// visible instead of two people typing into the same machine invisibly. The
// WebClient enforces the grant locally, exactly as it already does for
// same-origin tabs.
type ControlGrants struct {
	mu      sync.Mutex
	grants  map[string]ControlGrant
	revoked map[string]string // device_id -> client_id that just lost control
}

func NewControlGrants() *ControlGrants {
	return &ControlGrants{
		grants:  map[string]ControlGrant{},
		revoked: map[string]string{},
	}
}

// Request asks for control on behalf of clientID and returns the grant that
// applies. In exclusive mode the asker becomes the holder; the previous holder,
// if there was one and it was somebody else, is queued for a revoke notice.
func (c *ControlGrants) Request(deviceID, clientID, mode string) ControlGrant {
	if c == nil {
		return ControlGrant{Mode: NormalizeControlMode(mode)}
	}
	mode = NormalizeControlMode(mode)
	c.mu.Lock()
	defer c.mu.Unlock()
	grant := ControlGrant{Mode: mode}
	if mode == ControlModeExclusive {
		if previous, ok := c.grants[deviceID]; ok && previous.Holder != "" && previous.Holder != clientID {
			c.revoked[deviceID] = previous.Holder
		}
		grant.Holder = clientID
	}
	c.grants[deviceID] = grant
	return grant
}

// Get returns the current grant, or a grant that holds for everybody when the
// device has no mode recorded yet.
//
// A device in exclusive mode whose holder just released still reports exclusive
// with no holder: that is "the machine is free", not "the machine is shared".
// Reading it as shared would hand control to whoever asked next without the
// owner ever changing the setting.
func (c *ControlGrants) Get(deviceID string) ControlGrant {
	if c == nil {
		return ControlGrant{Mode: ControlModeShared}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.grants[deviceID]
}

// TakeRevoked returns the client_id that lost control since the last call, and
// whether there was one. Draining is what keeps a notice from being replayed at
// a controller that already acknowledged it.
func (c *ControlGrants) TakeRevoked(deviceID string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	holder, ok := c.revoked[deviceID]
	if ok {
		delete(c.revoked, deviceID)
	}
	return holder, ok
}

// Release gives up control. Only the holder's own release is honoured, so a
// controller that is merely connected cannot free a machine it never held.
func (c *ControlGrants) Release(deviceID, clientID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	grant, ok := c.grants[deviceID]
	if !ok || grant.Mode != ControlModeExclusive {
		return
	}
	if grant.Holder != clientID {
		return
	}
	// The mode stays; only the holder goes. A released exclusive machine is
	// free, not shared.
	c.grants[deviceID] = ControlGrant{Mode: ControlModeExclusive}
}

// ForgetDevice drops a device's grant and any pending notice for it. The host
// going away is the case: every controller is about to be disconnected, so
// telling one of them it lost control would be noise.
func (c *ControlGrants) ForgetDevice(deviceID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.grants, deviceID)
	delete(c.revoked, deviceID)
}
