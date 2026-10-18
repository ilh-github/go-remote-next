package models

import "time"

// EventOutbox is the durable handoff between a database transaction and the
// realtime event bus. Producers insert the row in the same transaction as the
// business change; a worker later publishes it to Redis/subscribers and marks
// it delivered.
//
// The row is intentionally transport-neutral: Payload is the complete Event
// envelope before server_rev is assigned. That keeps replay deterministic if
// the first publish attempt happens after a process restart.
type EventOutbox struct {
	ID        string `gorm:"primaryKey;size:64" json:"id"`
	EventType string `gorm:"size:64;not null;index" json:"event_type"`
	UserID    uint   `gorm:"not null;default:0;index" json:"user_id"`
	DeviceID  string `gorm:"size:20;not null;default:'';index" json:"device_id"`
	Payload   string `gorm:"type:text;not null" json:"payload"`

	Status string `gorm:"size:16;not null;default:'pending';index" json:"status"`
	// Attempts counts successful claims, including an attempt that is still in
	// flight. A stale claim therefore carries enough information to back off.
	Attempts       int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt  time.Time  `gorm:"not null;index" json:"next_attempt_at"`
	ClaimID        string     `gorm:"size:64;not null;default:'';index" json:"claim_id"`
	ClaimExpiresAt *time.Time `gorm:"index" json:"claim_expires_at"`
	LastError      string     `gorm:"size:255;not null;default:''" json:"last_error"`
	DeliveredAt    *time.Time `json:"delivered_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (EventOutbox) TableName() string {
	return "event_outbox"
}
