package service

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Delivery states for a notification outbox row.
const (
	DeliveryStatusPending    = "pending"
	DeliveryStatusDelivering = "delivering"
	DeliveryStatusDelivered  = "delivered"
	DeliveryStatusFailed     = "failed"
)

const (
	// notificationMaxAttempts is how many failed posts a row gets before it
	// is parked as `failed` (still visible through the API).
	notificationMaxAttempts = 8
	// notificationBackoffCap bounds the exponential backoff.
	notificationBackoffCap = 5 * time.Minute
	// notificationClaimBatch is how many due rows one worker pass takes.
	notificationClaimBatch = 20
	// notificationWorkerInterval is the queue poll interval.
	notificationWorkerInterval = 2 * time.Second
)

// NotificationDeliveryService is the durable half of user notifications: it
// stores one outbox row per (event, channel) and retries until the endpoint
// answers 2xx. Delivery is at-least-once — a receiver that errors after
// processing may see the event twice, so consumers should treat
// `event` + `timestamp` + `data.log_id` as an idempotency key.
type NotificationDeliveryService struct {
	db *gorm.DB
}

func NewNotificationDeliveryService(db *gorm.DB) *NotificationDeliveryService {
	return &NotificationDeliveryService{db: db}
}

// Enqueue stores one delivery attempt for a channel. Called from the event
// subscriber, so it must stay cheap: no HTTP here.
func (s *NotificationDeliveryService) Enqueue(ctx context.Context, ch models.UserNotificationChannel, eventType string, payload interface{}) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	row := &models.NotificationDelivery{
		ChannelID:     ch.ID,
		UserID:        ch.UserID,
		EventType:     eventType,
		Payload:       string(encoded),
		Status:        DeliveryStatusPending,
		NextAttemptAt: time.Now().UTC(),
	}
	return s.db.WithContext(ctx).Create(row).Error
}

// Run polls the outbox until ctx is cancelled.
func (s *NotificationDeliveryService) Run(ctx context.Context) {
	ticker := time.NewTicker(notificationWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.DrainOnce(ctx)
		}
	}
}

// DrainOnce claims and delivers every row that is due right now. Exported so
// tests and maintenance commands can drive the queue directly.
func (s *NotificationDeliveryService) DrainOnce(ctx context.Context) {
	claimed, err := s.claimDue(ctx, notificationClaimBatch)
	if err != nil {
		log.Printf("[notification] claim failed: %v", err)
		return
	}
	for _, row := range claimed {
		s.deliver(ctx, row)
	}
}

// claimDue atomically flips up to limit due rows to `delivering`. The
// FOR UPDATE SKIP LOCKED claim keeps two workers (or two server instances)
// from sending the same row twice.
func (s *NotificationDeliveryService) claimDue(ctx context.Context, limit int) ([]models.NotificationDelivery, error) {
	var claimed []models.NotificationDelivery
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT * FROM notification_deliveries
			WHERE status = ? AND next_attempt_at <= ?
			ORDER BY id
			LIMIT ?
			FOR UPDATE SKIP LOCKED`,
			DeliveryStatusPending, time.Now().UTC(), limit).Scan(&claimed).Error; err != nil {
			return err
		}
		if len(claimed) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(claimed))
		for _, row := range claimed {
			ids = append(ids, row.ID)
		}
		return tx.Model(&models.NotificationDelivery{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     DeliveryStatusDelivering,
				"updated_at": time.Now().UTC(),
			}).Error
	})
	return claimed, err
}

// deliver performs one HTTP attempt for a claimed row and records the
// outcome: delivered, rescheduled with backoff, or parked as failed.
func (s *NotificationDeliveryService) deliver(ctx context.Context, row models.NotificationDelivery) {
	var channel models.UserNotificationChannel
	if err := s.db.WithContext(ctx).First(&channel, row.ChannelID).Error; err != nil {
		// The channel was deleted while the event was queued: nothing to send.
		s.finishDelivery(ctx, row.ID, 0, DeliveryStatusFailed, 0, "channel no longer exists")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, 0, "payload is not valid JSON")
		return
	}

	status, err := postWebhookOnce(channel.URL, channel.Secret, payload)
	attempts := row.Attempts + 1
	now := time.Now().UTC()

	if err == nil && status >= 200 && status < 300 {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusDelivered, status, "")
		return
	}

	message := messageFor(status, err)
	if attempts >= notificationMaxAttempts {
		s.finishDelivery(ctx, row.ID, channel.ID, DeliveryStatusFailed, status, message)
		return
	}
	if uerr := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":          DeliveryStatusPending,
			"attempts":        attempts,
			"last_status":     status,
			"last_error":      message,
			"next_attempt_at": now.Add(notificationBackoff(attempts)),
			"updated_at":      now,
		}).Error; uerr != nil {
		log.Printf("[notification] reschedule delivery %d failed: %v", row.ID, uerr)
	}
	s.touchChannel(ctx, channel.ID, status, now)
}

// finishDelivery writes a terminal state and records the outcome on the
// channel so the API can show whether the endpoint is healthy.
func (s *NotificationDeliveryService) finishDelivery(ctx context.Context, id, channelID uint, status string, httpStatus int, message string) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":      status,
		"last_status": httpStatus,
		"last_error":  message,
		"updated_at":  now,
		"attempts":    gorm.Expr("attempts + 1"),
	}
	if status == DeliveryStatusDelivered {
		updates["delivered_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&models.NotificationDelivery{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		log.Printf("[notification] finish delivery %d failed: %v", id, err)
	}
	if channelID != 0 {
		s.touchChannel(ctx, channelID, httpStatus, now)
	}
}

// touchChannel records the last outcome on the channel row.
func (s *NotificationDeliveryService) touchChannel(ctx context.Context, channelID uint, status int, at time.Time) {
	if err := s.db.WithContext(ctx).Model(&models.UserNotificationChannel{}).
		Where("id = ?", channelID).
		Updates(map[string]interface{}{
			"last_triggered": at,
			"last_status":    status,
		}).Error; err != nil {
		log.Printf("[notification] update channel %d status: %v", channelID, err)
	}
}

// ListByChannel returns the delivery log for one channel, newest first.
func (s *NotificationDeliveryService) ListByChannel(ctx context.Context, channelID uint, limit int) ([]models.NotificationDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []models.NotificationDelivery
	err := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// notificationBackoff grows 2s, 4s, 8s … up to the cap.
func notificationBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	backoff := time.Duration(1<<uint(min(attempts, 10))) * time.Second
	if backoff > notificationBackoffCap {
		return notificationBackoffCap
	}
	return backoff
}

func messageFor(status int, err error) string {
	if err != nil {
		return err.Error()
	}
	if status == 0 {
		return "transport error"
	}
	return "HTTP " + strconv.Itoa(status)
}
