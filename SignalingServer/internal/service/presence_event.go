package service

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ErrPresenceEventNotDurable is returned when the durable write for a presence
// transition failed, so the caller can decide what the owner is told. It is a
// distinct error rather than a wrapped bus error because the two failures mean
// different things: the bus one is "nobody was told", this one is "the record
// that would have told them later does not exist".
var ErrPresenceEventNotDurable = errors.New("presence event could not be made durable")

// PresenceEventPublisher publishes `device.online.changed` for a transition
// this process observed.
//
// The offline direction already had a durable path (PresenceWatcher.WithOutbox
// commits the event before it drops the remembered-online candidate). The online
// direction did not: the WS handshake and the heartbeat route published straight
// to the bus, so a Redis failure between "the device came online" and "the
// owner's client was told" lost the transition with no row to retry from. The
// owner's device list kept showing offline until the next transition, and the
// offline-alert evaluator - which reads presence rather than events - had
// already moved on.
//
// A transition the owner is told about is the same promise in both directions,
// so both directions now share this one implementation.
type PresenceEventPublisher struct {
	db     *gorm.DB
	outbox EventOutboxWriter
	bus    *EventBus
}

func NewPresenceEventPublisher(db *gorm.DB, bus *EventBus) *PresenceEventPublisher {
	return &PresenceEventPublisher{db: db, bus: bus}
}

// WithOutbox makes online transitions durable. Without one the publisher stays
// best-effort, which is the behaviour every existing deployment already has.
func (p *PresenceEventPublisher) WithOutbox(outbox EventOutboxWriter) *PresenceEventPublisher {
	p.outbox = outbox
	return p
}

// PublishOnline enqueues or publishes one online transition.
func (p *PresenceEventPublisher) PublishOnline(ctx context.Context, deviceID string, userID uint, loggedIn bool) error {
	return p.publish(ctx, deviceID, userID, true, loggedIn)
}

// PublishOffline enqueues or publishes one offline transition.
//
// It exists so the WS-disconnect producer stops being a fourth, non-durable
// copy of the same event. The keyspace-expiry and fallback-scanner producers
// go through PresenceWatcher.publishOfflineIfRemembered, which is durable; the
// disconnect path in RealtimeHandler.unregisterSignalConn published straight
// to the bus, so the transition that happens most often in practice - a host
// closing its socket - was the one most likely to be lost.
func (p *PresenceEventPublisher) PublishOffline(ctx context.Context, deviceID string, userID uint) error {
	return p.publish(ctx, deviceID, userID, false, false)
}

func (p *PresenceEventPublisher) publish(
	ctx context.Context, deviceID string, userID uint, online, loggedIn bool,
) error {
	if p == nil {
		return nil
	}
	if userID == 0 || deviceID == "" {
		// An unbound device has no owner to route a user-scoped event to. The
		// system-scope presence state is maintained separately, so there is
		// nothing to publish and nothing to make durable.
		return nil
	}
	evt := Event{
		Type:     EventDeviceOnlineChanged,
		UserID:   userID,
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"device_id": deviceID,
			"online":    online,
			"logged_in": loggedIn,
		},
	}
	if p.outbox == nil {
		p.bus.Publish(ctx, evt)
		return nil
	}
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := p.outbox.EnqueueTx(ctx, tx, evt)
		return err
	})
	if err != nil {
		return errors.Join(ErrPresenceEventNotDurable, err)
	}
	return nil
}
