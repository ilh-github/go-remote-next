package service

import (
	"context"

	"gorm.io/gorm"
)

// SessionRevokedEmitter makes the account-session revocation event
// recoverable.
//
// The producers of `session.revoked` — password change, password reset,
// refresh-family break, admin force logout and the scheduled sign-out action —
// all revoke token families in Redis and then publish. The revocation itself
// cannot join a database transaction because it lives in Redis, so this type
// does not claim atomicity between the two. What it does guarantee is that the
// *publish* is durable: a failed or interrupted publish stays pending in the
// outbox and is retried, instead of being lost and leaving the connected
// events WebSocket unaware that it was kicked.
//
// Without an outbox configured it falls back to a direct publish, matching the
// previous behaviour.
type SessionRevokedEmitter struct {
	outbox EventOutboxWriter
	bus    *EventBus
	db     *gorm.DB
}

func NewSessionRevokedEmitter(db *gorm.DB, bus *EventBus) *SessionRevokedEmitter {
	return &SessionRevokedEmitter{bus: bus, db: db}
}

// WithOutbox overrides the outbox, letting callers share one worker instance.
func (p *SessionRevokedEmitter) WithOutbox(outbox EventOutboxWriter) *SessionRevokedEmitter {
	p.outbox = outbox
	return p
}

// Publish records the revocation event. data carries the reason and, for a
// scoped family break, the family_id that alone may receive it.
func (p *SessionRevokedEmitter) Publish(ctx context.Context, userID uint, data map[string]interface{}) error {
	if p == nil || userID == 0 {
		return nil
	}
	evt := Event{
		Type:   EventSessionRevoked,
		UserID: userID,
		Data:   data,
	}
	if p.outbox == nil || p.db == nil {
		if p.bus == nil {
			return nil
		}
		p.bus.Publish(ctx, evt)
		return nil
	}
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := p.outbox.EnqueueTx(ctx, tx, evt)
		return err
	})
}
