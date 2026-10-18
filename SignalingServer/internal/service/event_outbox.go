package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	eventOutboxPending   = "pending"
	eventOutboxDelivered = "delivered"
	eventOutboxFailed    = "failed"

	defaultEventOutboxClaimTTL   = 2 * time.Minute
	defaultEventOutboxBatchSize  = 100
	defaultEventOutboxPollPeriod = time.Second
)

// EventOutbox turns "business DB commit" and "event bus publish" into an
// at-least-once handoff. The row goes into the same transaction as the business
// change; a later worker claims it and publishes the Event envelope.
type EventOutbox struct {
	db       *gorm.DB
	bus      *EventBus
	claimTTL time.Duration
	batch    int
}

func NewEventOutbox(db *gorm.DB, bus *EventBus) *EventOutbox {
	return &EventOutbox{
		db:       db,
		bus:      bus,
		claimTTL: defaultEventOutboxClaimTTL,
		batch:    defaultEventOutboxBatchSize,
	}
}

// EnqueueTx must be called with the same *gorm.DB transaction that writes the
// business row. A rollback removes the outbox row along with the change.
func (o *EventOutbox) EnqueueTx(ctx context.Context, tx *gorm.DB, evt Event) (Event, error) {
	if tx == nil {
		return Event{}, errors.New("event outbox requires a transaction")
	}
	evt = prepareEvent(evt)
	payload, err := json.Marshal(evt)
	if err != nil {
		return Event{}, fmt.Errorf("marshal outbox event: %w", err)
	}
	row := models.EventOutbox{
		ID:            evt.ID,
		EventType:     evt.Type,
		UserID:        evt.UserID,
		DeviceID:      evt.DeviceID,
		Payload:       string(payload),
		Status:        eventOutboxPending,
		NextAttemptAt: evt.Ts.UTC(),
	}
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return Event{}, fmt.Errorf("enqueue outbox event: %w", err)
	}
	return evt, nil
}

// Run drains due rows until ctx is cancelled. It is safe to run in every
// server instance: claims are conditional updates, so only one worker wins a
// row and an expired claim is recoverable after a crash.
func (o *EventOutbox) Run(ctx context.Context) {
	if o == nil || o.db == nil || o.bus == nil {
		return
	}
	ticker := time.NewTicker(defaultEventOutboxPollPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if _, err := o.Drain(ctx, now.UTC(), o.batch); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("[EventOutbox] drain failed: %v", err)
			}
		}
	}
}

// Drain processes one batch of due rows. It is exported so tests and
// maintenance commands can run without waiting for the ticker.
func (o *EventOutbox) Drain(ctx context.Context, now time.Time, limit int) (int, error) {
	if o == nil || o.db == nil || o.bus == nil {
		return 0, errors.New("event outbox is not configured")
	}
	if limit <= 0 {
		limit = o.batch
	}
	now = now.UTC()

	var rows []models.EventOutbox
	if err := o.db.WithContext(ctx).
		Where("status = ? AND next_attempt_at <= ? AND (claim_expires_at IS NULL OR claim_expires_at <= ?)",
			eventOutboxPending, now, now).
		Order("next_attempt_at, created_at, id").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("list due outbox events: %w", err)
	}

	claimed := 0
	for i := range rows {
		claimID := uuid.NewString()
		claimExpires := now.Add(o.claimTTL)
		result := o.db.WithContext(ctx).
			Model(&models.EventOutbox{}).
			Where("id = ? AND status = ? AND (claim_expires_at IS NULL OR claim_expires_at <= ?)",
				rows[i].ID, eventOutboxPending, now).
			Updates(map[string]interface{}{
				"claim_id":         claimID,
				"claim_expires_at": claimExpires,
				"attempts":         gorm.Expr("attempts + 1"),
				"updated_at":       now,
			})
		if result.Error != nil {
			return claimed, fmt.Errorf("claim outbox event %s: %w", rows[i].ID, result.Error)
		}
		if result.RowsAffected != 1 {
			continue
		}
		claimed++

		var evt Event
		if err := json.Unmarshal([]byte(rows[i].Payload), &evt); err != nil {
			o.markFailed(ctx, rows[i].ID, claimID, now, fmt.Errorf("decode outbox event: %w", err))
			continue
		}
		if err := o.bus.publish(ctx, evt, true); err != nil {
			o.releaseForRetry(ctx, rows[i].ID, claimID, now, rows[i].Attempts+1, err)
			continue
		}
		if err := o.db.WithContext(ctx).
			Model(&models.EventOutbox{}).
			Where("id = ? AND claim_id = ?", rows[i].ID, claimID).
			Updates(map[string]interface{}{
				"status":           eventOutboxDelivered,
				"claim_id":         "",
				"claim_expires_at": nil,
				"last_error":       "",
				"delivered_at":     now,
				"updated_at":       now,
			}).Error; err != nil {
			return claimed, fmt.Errorf("mark outbox event %s delivered: %w", rows[i].ID, err)
		}
	}
	return claimed, nil
}

func (o *EventOutbox) releaseForRetry(ctx context.Context, id, claimID string, now time.Time, attempts int, cause error) {
	delay := eventOutboxRetryDelay(attempts)
	errText := truncateEventOutboxError(cause.Error())
	if err := o.db.WithContext(ctx).
		Model(&models.EventOutbox{}).
		Where("id = ? AND claim_id = ?", id, claimID).
		Updates(map[string]interface{}{
			"claim_id":         "",
			"claim_expires_at": nil,
			"next_attempt_at":  now.Add(delay),
			"last_error":       errText,
			"updated_at":       now,
		}).Error; err != nil {
		log.Printf("[EventOutbox] release %s for retry failed: %v", id, err)
	}
}

func (o *EventOutbox) markFailed(ctx context.Context, id, claimID string, now time.Time, cause error) {
	if err := o.db.WithContext(ctx).
		Model(&models.EventOutbox{}).
		Where("id = ? AND claim_id = ?", id, claimID).
		Updates(map[string]interface{}{
			"status":           eventOutboxFailed,
			"claim_id":         "",
			"claim_expires_at": nil,
			"last_error":       truncateEventOutboxError(cause.Error()),
			"updated_at":       now,
		}).Error; err != nil {
		log.Printf("[EventOutbox] mark %s failed: %v", id, err)
	}
}

func truncateEventOutboxError(message string) string {
	if len(message) > 255 {
		return message[:255]
	}
	return message
}

func eventOutboxRetryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := 2 * time.Second
	for i := 1; i < attempts && delay < 2*time.Minute; i++ {
		delay *= 2
	}
	if delay > 2*time.Minute {
		return 2 * time.Minute
	}
	return delay
}
