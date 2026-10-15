package models

import "time"

// NotificationDelivery is one outbox row: a user-scoped event that must reach
// one notification channel. Rows are written when the event happens and are
// retried by the delivery worker until they succeed or run out of attempts,
// so a channel that is briefly down no longer loses the event.
type NotificationDelivery struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ChannelID uint   `gorm:"not null;index" json:"channel_id"`
	UserID    uint   `gorm:"not null;index" json:"user_id"`
	EventType string `gorm:"size:64;not null" json:"event_type"`
	Payload   string `gorm:"type:text;not null" json:"payload"` // JSON body actually posted

	// Status is pending | delivering | delivered | failed.
	Status string `gorm:"size:16;not null;index" json:"status"`
	// Attempts counts finished delivery attempts (not claims).
	Attempts int `gorm:"not null;default:0" json:"attempts"`
	// NextAttemptAt is when the worker may pick the row up again.
	NextAttemptAt time.Time `gorm:"not null;index" json:"next_attempt_at"`

	LastStatus  int        `json:"last_status"` // HTTP status of the last attempt, 0 = transport error
	LastError   string     `gorm:"size:255" json:"last_error"`
	DeliveredAt *time.Time `json:"delivered_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (NotificationDelivery) TableName() string {
	return "notification_deliveries"
}
