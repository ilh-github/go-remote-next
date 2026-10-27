package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SupportedUserNotificationEvents are the user-scoped events a user-owned
// channel may subscribe to. The list is explicit on purpose: admin webhooks
// only carry system-scope events (§2.17 M4), so anything tied to one user's
// devices has to be opted into here, and a typo in an event name should fail
// loudly at create time instead of silently never firing.
var SupportedUserNotificationEvents = []string{
	EventDeviceAccessDenied,
	EventDeviceAccessGranted,
	EventDeviceAccessSession,
	EventDeviceOnlineChanged,
	EventDeviceOfflineSustained,
	EventDeviceMetricAlert,
	EventDeviceBound,
	EventDeviceUnbound,
	EventDeviceAccessApprovalRequested,
	EventDeviceAccessApprovalDecided,
	// A scheduled task that has spent its retry budget is an operational
	// failure the owner asked to be told about by scheduling it.
	EventScheduledTaskFailed,
	EventDeviceConnectPrompted,
	// A scheduled connection prompt. The owner asked to be told at a time they
	// chose; without this the prompt reaches the realtime socket and nowhere the
	// owner configured, which is the half-wired version of the feature.
}

// IsSupportedUserNotificationEvent reports whether evtType may be subscribed
// to by a user notification channel.
func IsSupportedUserNotificationEvent(evtType string) bool {
	for _, e := range SupportedUserNotificationEvents {
		if e == evtType {
			return true
		}
	}
	return false
}

var (
	// ErrInvalidNotificationURL means the endpoint is not a usable http(s) URL.
	ErrInvalidNotificationURL = errors.New("notification URL must be an absolute http(s) URL")
	// ErrUnsupportedNotificationEvent means the caller subscribed to an event
	// outside SupportedUserNotificationEvents.
	ErrUnsupportedNotificationEvent = errors.New("unsupported notification event")
)

// ValidateNotificationURL rejects anything that is not an absolute http(s)
// endpoint. Note for operators: this is not an SSRF guard by itself — a
// self-hosted deployment still lets a user point a channel at an internal
// address, which is acceptable for a single-tenant install and should be
// revisited before this is exposed as multi-tenant SaaS.
func ValidateNotificationURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ErrInvalidNotificationURL
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return ErrInvalidNotificationURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidNotificationURL
	}
	if u.Host == "" {
		return ErrInvalidNotificationURL
	}
	return nil
}

// NotificationChannelService owns per-user webhook endpoints: CRUD plus the
// delivery path that turns user-scoped bus events into outbound HTTP calls.
type NotificationChannelService struct {
	db         *gorm.DB
	deliveries *NotificationDeliveryService
}

func NewNotificationChannelService(db *gorm.DB, deliveries *NotificationDeliveryService) *NotificationChannelService {
	return &NotificationChannelService{db: db, deliveries: deliveries}
}

// UpsertNotificationChannelInput is the validated body of the create/update
// endpoints.
type UpsertNotificationChannelInput struct {
	Name    string
	URL     string
	Secret  string
	Events  []string
	Enabled bool
}

func (s *NotificationChannelService) Create(ctx context.Context, userID uint, in UpsertNotificationChannelInput) (*models.UserNotificationChannel, error) {
	if err := validateChannelInput(in); err != nil {
		return nil, err
	}
	eventsJSON, _ := json.Marshal(in.Events)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "channel"
	}
	row := &models.UserNotificationChannel{
		UserID:  userID,
		Name:    name,
		URL:     strings.TrimSpace(in.URL),
		Secret:  in.Secret,
		Events:  string(eventsJSON),
		Enabled: in.Enabled,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *NotificationChannelService) List(ctx context.Context, userID uint) ([]models.UserNotificationChannel, error) {
	var rows []models.UserNotificationChannel
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}

func (s *NotificationChannelService) Get(ctx context.Context, userID, id uint) (*models.UserNotificationChannel, error) {
	var row models.UserNotificationChannel
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NotificationChannelService) Update(ctx context.Context, userID, id uint, in UpsertNotificationChannelInput) (*models.UserNotificationChannel, error) {
	if err := validateChannelInput(in); err != nil {
		return nil, err
	}
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	eventsJSON, _ := json.Marshal(in.Events)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "channel"
	}
	row.Name = name
	row.URL = strings.TrimSpace(in.URL)
	if in.Secret != "" {
		row.Secret = in.Secret
	}
	row.Events = string(eventsJSON)
	row.Enabled = in.Enabled
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *NotificationChannelService) Delete(ctx context.Context, userID, id uint) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.UserNotificationChannel{}).Error
}

// DispatchEvent fans one user-scoped event out to that user's matching
// channels. Called from the event bus subscriber, so it must not block: each
// delivery runs in its own goroutine and only the delivery itself writes the
// last_triggered / last_status bookkeeping.
func (s *NotificationChannelService) DispatchEvent(ctx context.Context, evt Event) {
	if s.dispatchOnce(ctx, evt) {
		return
	}
	// The channel load is the only thing between the event and its channels,
	// and a failure here used to drop the event with a log line as the only
	// trace. The outbox row that carried it was already marked delivered by
	// then, so nothing anywhere retried it: the user's webhook went quiet and
	// the only evidence was one line in a log nobody was reading.
	//
	// Retrying in place is the right shape rather than writing a channel-less
	// delivery row. That row cannot work: the delivery worker resolves
	// ChannelID with First() and parks anything unresolvable as `failed`, so
	// "unrouted" would be indistinguishable from "the channel was deleted". A
	// read that failed is not a delivery that failed.
	if !s.dispatchWithRetry(ctx, evt) {
		log.Printf("[notification] dispatch %s for user %d failed after %d attempts",
			evt.Type, evt.UserID, dispatchLoadAttempts)
	}
}

// dispatchOnce performs one complete fan-out pass and reports whether it ran to
// the end. It is the retryable unit: everything that can fail is inside it, and
// the caller decides what a failure means.
func (s *NotificationChannelService) dispatchOnce(ctx context.Context, evt Event) bool {
	if evt.UserID == 0 || !IsSupportedUserNotificationEvent(evt.Type) {
		return true // nothing to do is not a failure
	}
	var channels []models.UserNotificationChannel
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND enabled = ?", evt.UserID, true).
		Find(&channels).Error; err != nil {
		return false
	}
	if len(channels) == 0 {
		return true
	}
	payload := notificationEventPayload(evt)
	complete := true
	for _, ch := range channels {
		if !channelSubscribes(ch.Events, evt.Type) {
			continue
		}
		if s.deliveries == nil {
			log.Printf("[notification] no delivery queue configured; dropping %s for channel %d",
				evt.Type, ch.ID)
			continue
		}
		if err := s.deliveries.Enqueue(ctx, ch, evt.Type, evt.DeviceID, payload); err != nil {
			log.Printf("[notification] enqueue %s for channel %d failed: %v", evt.Type, ch.ID, err)
			// One channel's failure must not fail the pass: the others are
			// still worth delivering to, and a retry would re-deliver the ones
			// that already succeeded. The log line is the trace for this one.
			complete = false
		}
	}
	return complete
}

// dispatchLoadAttempts is how many times the channel load is retried before the
// event is given up on. Small on purpose: the load is one indexed SELECT on the
// subscriber worker, and the worker is shared with every other subscriber, so a
// long retry budget would hold up unrelated events while it burned.
const dispatchLoadAttempts = 3

// dispatchWithRetry re-attempts the whole fan-out after the channel load failed.
// It reports whether any attempt succeeded, so the caller knows whether there is
// anything left to log.
//
// The retry covers the load and the enqueues together rather than just the read,
// because the failure that matters is "this pass could not complete", and a
// second pass that fails on a different step has learned something the first
// one could not have.
func (s *NotificationChannelService) dispatchWithRetry(ctx context.Context, evt Event) bool {
	for attempt := 1; attempt < dispatchLoadAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(dispatchRetryBackoff(attempt)):
		}
		if s.dispatchOnce(ctx, evt) {
			return true
		}
	}
	return false
}

func dispatchRetryBackoff(attempt int) time.Duration {
	// 200ms, 400ms. Long enough for a connection pool to hand out a fresh
	// connection, short enough that a three-attempt budget finishes inside a
	// second - the same order as the tick the delivery worker runs on.
	return time.Duration(attempt) * 200 * time.Millisecond
}

// notificationEventPayload is the body posted to a user's channel. The top
// level `id` is the event envelope's stable identity: delivery is
// at-least-once, so a receiver needs a value it can key a dedupe table on even
// when the event data itself carries no unique field. It is echoed back in the
// X-Webhook-Event-Id header on the outbox POST.
func notificationEventPayload(evt Event) map[string]interface{} {
	payload := map[string]interface{}{
		"event":     evt.Type,
		"timestamp": evt.Ts.UTC().Format(time.RFC3339),
		"data":      evt.Data,
	}
	if evt.ID != "" {
		payload["id"] = evt.ID
	}
	return payload
}

// SendTest delivers a synthetic `test` event so a user can verify their
// endpoint without waiting for real device traffic.
func (s *NotificationChannelService) SendTest(ctx context.Context, userID, id uint) error {
	row, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	payload := notificationEventPayload(Event{
		ID:   "evt_" + uuid.New().String(),
		Type: "test",
		Ts:   time.Now().UTC(),
		Data: map[string]interface{}{
			"message": "This is a test notification from QuickDesk",
		},
	})
	if s.deliveries == nil {
		return errors.New("delivery queue is not configured")
	}
	return s.deliveries.Enqueue(ctx, *row, "test", "", payload)
}

func validateChannelInput(in UpsertNotificationChannelInput) error {
	if err := ValidateNotificationURL(in.URL); err != nil {
		return err
	}
	if len(in.Events) == 0 {
		return fmt.Errorf("%w: at least one event is required", ErrUnsupportedNotificationEvent)
	}
	for _, e := range in.Events {
		if !IsSupportedUserNotificationEvent(e) {
			return fmt.Errorf("%w: %q (supported: %s)",
				ErrUnsupportedNotificationEvent, e, strings.Join(SupportedUserNotificationEvents, ", "))
		}
	}
	return nil
}

// channelSubscribes reports whether the stored JSON event list contains
// evtType. A malformed list never matches.
func channelSubscribes(eventsJSON, evtType string) bool {
	var events []string
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return false
	}
	for _, e := range events {
		if e == evtType {
			return true
		}
	}
	return false
}
