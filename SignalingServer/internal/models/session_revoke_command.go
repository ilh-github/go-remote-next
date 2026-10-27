package models

import "time"

// SessionRevokeCommand is one "close this signaling socket" instruction, written
// before it is delivered.
//
// The channel between signaling instances is Redis pub/sub, which is
// fire-and-forget: a process that dies between reading the session row and
// publishing to the owning instance's channel leaves nothing behind. The session
// row still gets closed about a minute later by the presence-lease sweep, so the
// command is never reported as a success that did not happen - but for that
// minute a controller the operator kicked is still holding the socket, and the
// only thing that eventually acts is a statement about the instance dying, not
// about this revoke.
//
// Recording the command first closes that window. A row in the database outlives
// the process that wrote it, so a revoke that was decided but not yet delivered
// survives the crash and is delivered by whichever process picks it up. That is
// the same transactional-outbox shape the event pipeline uses, applied to a
// command rather than an event: the two are not the same thing and deliberately
// do not share a table - an event is fanned out to subscribers, a command is
// addressed at exactly one instance and acts once.
type SessionRevokeCommand struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// SessionID is the socket to close. It is the join key back to the session
	// row, which is what decides whether the command is still worth delivering.
	SessionID string `gorm:"size:64;not null;uniqueIndex" json:"session_id"`
	// InstanceID names the process that holds the socket. Empty means the row was
	// written before the instance column existed; those cannot be addressed and
	// are never delivered.
	InstanceID string `gorm:"size:64;not null;default:'';index" json:"instance_id"`
	// Reason records who asked, so a delivered command and the session's own
	// end_reason can be read together instead of guessed at.
	Reason string `gorm:"size:32;not null;default:''" json:"reason"`

	// Delivered is set once the command reached the owning instance. It is not
	// deleted on delivery: a command that was delivered but whose socket close
	// was never confirmed has to stay distinguishable from one nobody attempted,
	// or the retry loop cannot tell the two apart.
	Delivered   bool       `gorm:"not null;default:false;index" json:"delivered"`
	DeliveredAt *time.Time `json:"delivered_at"`
	// Attempts counts delivery attempts, including one that is still in flight.
	Attempts int `gorm:"not null;default:0" json:"attempts"`
	// ClaimExpiresAt lets a delivery attempt that died mid-flight be retried
	// instead of waiting for a claim that will never be released.
	ClaimExpiresAt *time.Time `json:"claim_expires_at"`
	LastError      string     `gorm:"size:255;not null;default:''" json:"last_error"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SessionRevokeCommand) TableName() string {
	return "session_revoke_commands"
}

// Revoke command reasons. Persisted and read next to the session's end_reason.
const (
	SessionRevokeReasonOwner   = "owner_revoke"
	SessionRevokeReasonAdmin   = "admin_revoke"
	SessionRevokeReasonSched   = "scheduled_revoke"
	SessionRevokeReasonUnknown = ""
)
