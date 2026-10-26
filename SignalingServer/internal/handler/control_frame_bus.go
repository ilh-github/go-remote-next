package handler

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ControlFrameBus carries control frames between signaling instances.
//
// It exists for one notice: `control_revoked`. A takeover in exclusive mode is
// recorded wherever the asker connected, but the controller that just lost
// control may hold its socket in a different process - in a fleet that is the
// common case, not the exception. Delivering the notice only to local sockets
// would drop exactly the frame the feature is for, and the displaced holder
// would keep typing into a machine it no longer owns.
//
// The channel is addressed rather than broadcast, the same shape
// SessionRevokeBus uses: the payload names the instance, so only the process
// that can act receives it.
type ControlFrameBus struct {
	rdb        *redis.Client
	instanceID string
	onFrame    func(payload []byte)
}

func NewControlFrameBus(rdb *redis.Client, instanceID string, onFrame func(payload []byte)) *ControlFrameBus {
	return &ControlFrameBus{
		rdb:        rdb,
		instanceID: strings.TrimSpace(instanceID),
		onFrame:    onFrame,
	}
}

func (b *ControlFrameBus) channel(instanceID string) string {
	return "qd:sessions:instance:" + instanceID + ":control-frame"
}

// Publish hands one already-encoded frame to the instance that owns a socket.
func (b *ControlFrameBus) Publish(ctx context.Context, instanceID string, payload []byte) error {
	if b == nil || b.rdb == nil {
		return errors.New("control frame bus is not configured")
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return errors.New("instance_id is required")
	}
	return b.rdb.Publish(ctx, b.channel(instanceID), payload).Err()
}

// Run subscribes to this instance's own channel until the context ends,
// resubscribing with a backoff if the connection drops - a subscription that
// died silently would turn every cross-instance takeover notice into a dropped
// frame.
func (b *ControlFrameBus) Run(ctx context.Context) {
	if b == nil || b.rdb == nil || b.instanceID == "" {
		return
	}
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := b.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[control-frame] subscription to %s lost: %v (retry in %s)", b.channel(b.instanceID), err, backoff)
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
	}
}

func (b *ControlFrameBus) runOnce(ctx context.Context) error {
	sub := b.rdb.Subscribe(ctx, b.channel(b.instanceID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		return err
	}
	ch := sub.Channel()
	log.Printf("[control-frame] subscribed to %s", b.channel(b.instanceID))
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return errors.New("pubsub channel closed")
			}
			if len(msg.Payload) > 0 && b.onFrame != nil {
				b.onFrame([]byte(msg.Payload))
			}
		}
	}
}
