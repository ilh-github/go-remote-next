package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

type WebhookService struct {
	db *gorm.DB
	// deliveryObserver fires when a delivery is attempted, so a test can watch
	// the retry budget without standing up an HTTP server.
	deliveryObserver func()
	// deliveryRecorder records which webhook URL the attempt went to. Nil in
	// production; the real delivery path does not consult it.
	deliveryRecorder func(string)
}

func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db}
}

// WithDeliveryObserver and WithDeliveryRecorder are test seams. They exist
// because the only honest way to prove a retry happened is to count attempts,
// and counting them through a real HTTP server makes the test about the server.
func (s *WebhookService) WithDeliveryObserver(fn func()) *WebhookService {
	s.deliveryObserver = fn
	return s
}

func (s *WebhookService) WithDeliveryRecorder(fn func(string)) *WebhookService {
	s.deliveryRecorder = fn
	return s
}

func (s *WebhookService) Create(ctx context.Context, webhook *models.Webhook) error {
	return s.db.WithContext(ctx).Create(webhook).Error
}

func (s *WebhookService) GetAll(ctx context.Context) ([]models.Webhook, error) {
	var webhooks []models.Webhook
	err := s.db.WithContext(ctx).Order("created_at desc").Find(&webhooks).Error
	return webhooks, err
}

func (s *WebhookService) GetByID(ctx context.Context, id uint) (*models.Webhook, error) {
	var webhook models.Webhook
	err := s.db.WithContext(ctx).First(&webhook, id).Error
	return &webhook, err
}

func (s *WebhookService) Update(ctx context.Context, webhook *models.Webhook) error {
	return s.db.WithContext(ctx).Save(webhook).Error
}

func (s *WebhookService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Webhook{}, id).Error
}

func (s *WebhookService) Dispatch(event string, data interface{}) {
	var webhooks []models.Webhook
	if err := s.db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		// This load used to return silently. The user-side fan-out at least
		// logged its failures; this one dropped the event with no trace at all,
		// and the outbox row carrying it was already marked delivered by the
		// time the subscriber ran.
		//
		// Retrying in place is the right shape for the same reason it was there:
		// the events that reach here are session.revoked, turn.config.changed
		// and device.secret.rotated, and a revoked session an operator's SIEM
		// never hears about is the one that matters. The revocation happened,
		// the audit trail has it, and the system that was supposed to be told
		// was not.
		if !s.dispatchWithRetry(event, data) {
			log.Printf("[webhook] load enabled webhooks failed after %d attempts", dispatchAttempts)
		}
		return
	}
	s.fanOut(event, data, webhooks)
}

// dispatchAttempts is how many times the webhook load is retried. The load is
// one indexed SELECT on the shared subscriber worker, so a long budget would
// hold up unrelated events while it burned.
const dispatchAttempts = 3

// dispatchWithRetry re-attempts the load and fan-out after a failure, reporting
// whether any attempt got as far as finding the webhooks.
func (s *WebhookService) dispatchWithRetry(event string, data interface{}) bool {
	for attempt := 1; attempt < dispatchAttempts; attempt++ {
		time.Sleep(dispatchBackoff(attempt))
		var webhooks []models.Webhook
		if err := s.db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
			continue
		}
		s.fanOut(event, data, webhooks)
		return true
	}
	return false
}

func dispatchBackoff(attempt int) time.Duration {
	// 200ms, 400ms - the same shape as the notification fan-out, and short
	// enough that the whole budget finishes inside a second.
	return time.Duration(attempt) * 200 * time.Millisecond
}

// fanOut delivers to every webhook subscribed to the event. It is the part of
// Dispatch that runs once the webhooks are known, shared by the first attempt
// and the retries.
func (s *WebhookService) fanOut(event string, data interface{}, webhooks []models.Webhook) {
	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      data,
	}
	for _, wh := range webhooks {
		var events []string
		if err := json.Unmarshal([]byte(wh.Events), &events); err != nil {
			continue
		}
		subscribed := false
		for _, e := range events {
			if e == event {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}
		go s.deliver(wh, payload)
	}
}

func (s *WebhookService) deliver(webhook models.Webhook, payload map[string]interface{}) {
	// Signalled before the POST rather than after it: what the retry test is
	// asking is whether an attempt was made, and a real POST carries seconds of
	// its own backoff that has nothing to do with the load retry under test.
	if s.deliveryObserver != nil {
		s.deliveryObserver()
	}
	if s.deliveryRecorder != nil {
		s.deliveryRecorder(webhook.URL)
	}
	lastStatus := postWebhook(webhook.URL, webhook.Secret, payload)

	now := time.Now()
	if err := s.db.Model(&webhook).Updates(map[string]interface{}{
		"last_triggered": now,
		"last_status":    lastStatus,
	}).Error; err != nil {
		log.Printf("Failed to update webhook status: %v", err)
	}
}

func (s *WebhookService) Test(ctx context.Context, id uint) error {
	webhook, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"event":     "test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      map[string]string{"message": "This is a test webhook event"},
	}

	go s.deliver(*webhook, payload)
	return nil
}
