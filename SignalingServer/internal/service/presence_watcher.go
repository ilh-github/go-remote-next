package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"quickdesk/signaling/internal/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PresenceWatcher subscribes to Redis keyspace notifications and emits
// `device.online.changed` events when a device's heartbeat key expires —
// the "hb TTL 到期 → device.online.changed (异步)" producer from §2.17.
// It also runs a small fallback scanner over devices observed online, so
// deployments that forgot `notify-keyspace-events Ex` still push offline
// events instead of waiting for the user to refresh the device list.
//
// The goroutine requires the Redis server to have been started with
//
//	notify-keyspace-events Ex
//
// (or equivalent `CONFIG SET`). If the subscription fails we log once and
// keep retrying — the server stays usable; devices just rely on the
// synchronous wsconn DEL path for online-changed events.
type PresenceWatcher struct {
	db           *gorm.DB
	rdb          *redis.Client
	bus          *EventBus
	presence     *PresenceService
	deviceRepo   *repository.DeviceRepository
	scanInterval time.Duration
	outbox       EventOutboxWriter
}

func NewPresenceWatcher(
	db *gorm.DB,
	rdb *redis.Client,
	bus *EventBus,
	presence *PresenceService,
	deviceRepo *repository.DeviceRepository,
) *PresenceWatcher {
	return &PresenceWatcher{
		db:           db,
		rdb:          rdb,
		bus:          bus,
		presence:     presence,
		deviceRepo:   deviceRepo,
		scanInterval: offlineScanInterval,
	}
}

// WithOutbox makes the offline transition durable: the remembered-online
// candidate is dropped and its `device.online.changed` event is enqueued in
// one transaction, so a crash between them cannot silently lose the event.
func (w *PresenceWatcher) WithOutbox(outbox EventOutboxWriter) *PresenceWatcher {
	w.outbox = outbox
	return w
}

// Start launches the subscription goroutine. Cancel ctx to stop.
//
// We only subscribe to the default DB's expiry channel
// (`__keyevent@0__:expired`). If the deploy uses a non-zero DB index, the
// caller should override before calling Start.
func (w *PresenceWatcher) Start(ctx context.Context) {
	if w.rdb == nil {
		return
	}
	go w.runForever(ctx)
	go w.runOfflineScanner(ctx)
}

const (
	presenceHbPrefix    = "qd:presence:device:"
	presenceHbSuffix    = ":hb"
	keyspaceExpiryTopic = "__keyevent@0__:expired"
	offlineScanInterval = 5 * time.Second
)

// runForever is the resilient pubsub loop: if the server drops the
// subscription (network blip, Redis restart, CONFIG change) we back off
// briefly and resubscribe. We never give up — the worker is a singleton
// for the process lifetime.
func (w *PresenceWatcher) runForever(ctx context.Context) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := w.runOnce(ctx); err != nil {
			log.Printf("[PresenceWatcher] subscription lost: %v (retry in %s)", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = time.Second
	}
}

func (w *PresenceWatcher) runOnce(ctx context.Context) error {
	sub := w.rdb.Subscribe(ctx, keyspaceExpiryTopic)
	defer sub.Close()
	// Wait for subscription to be accepted.
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	ch := sub.Channel()
	log.Printf("[PresenceWatcher] subscribed to %s (expecting notify-keyspace-events=Ex)", keyspaceExpiryTopic)
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return errors.New("pubsub channel closed")
			}
			w.handleExpiry(ctx, msg.Payload)
		}
	}
}

// handleExpiry extracts device_id from the expired key and publishes
// device.online.changed for the device's current owner. If the device has
// no owner (unbound), nothing to publish to — the wsconn path still
// maintains system-scope state.
func (w *PresenceWatcher) handleExpiry(ctx context.Context, key string) {
	if !strings.HasPrefix(key, presenceHbPrefix) || !strings.HasSuffix(key, presenceHbSuffix) {
		return
	}
	deviceID := key[len(presenceHbPrefix) : len(key)-len(presenceHbSuffix)]
	if deviceID == "" {
		return
	}

	// Double-check online state. Heartbeat expired but a WS connection
	// may still hold the wsconn key, so "online" is still live by our
	// derivation — no event necessary.
	if w.presence.IsOnline(ctx, deviceID) {
		log.Printf("[presence] heartbeat expired but device still online device=%s presence={%s}",
			deviceID, w.presence.State(ctx, deviceID).String())
		return
	}

	w.publishOfflineIfRemembered(ctx, deviceID)
}

func (w *PresenceWatcher) runOfflineScanner(ctx context.Context) {
	interval := w.scanInterval
	if interval <= 0 {
		interval = offlineScanInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, deviceID := range w.presence.KnownOnlineCandidates(ctx) {
				if deviceID == "" || w.presence.IsOnline(ctx, deviceID) {
					continue
				}
				w.publishOfflineIfRemembered(ctx, deviceID)
			}
		}
	}
}

func (w *PresenceWatcher) publishOfflineIfRemembered(ctx context.Context, deviceID string) {
	if !w.presence.ForgetOnlineCandidate(ctx, deviceID) {
		return
	}
	log.Printf("[presence] device offline device=%s presence={%s}",
		deviceID, w.presence.State(ctx, deviceID).String())

	// Resolve owner so we can route the user-scoped event.
	d, err := w.deviceRepo.GetByDeviceID(ctx, deviceID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[PresenceWatcher] lookup device %s failed: %v", deviceID, err)
		}
		return
	}
	if d.UserID == nil {
		return
	}

	evt := Event{
		Type:     EventDeviceOnlineChanged,
		UserID:   *d.UserID,
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"device_id": deviceID,
			"online":    false,
			"logged_in": false, // derived: intent AND online → false since online=false
		},
	}

	// Without an outbox this stays best-effort. With one, the durable event is
	// committed before the in-memory candidate is dropped, so a crash between
	// "device went offline" and "event exists" cannot lose the transition.
	if w.outbox == nil {
		w.bus.Publish(ctx, evt)
		return
	}
	if err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := w.outbox.EnqueueTx(ctx, tx, evt)
		return err
	}); err != nil {
		log.Printf("[PresenceWatcher] enqueue offline event for %s failed: %v", deviceID, err)
	}
}
