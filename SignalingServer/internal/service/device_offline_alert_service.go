package service

import (
	"context"
	"errors"
	"log"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidOfflineAlertThreshold means after_minutes was outside the accepted
// range. A zero (or negative) threshold would mean "alert the instant the
// heartbeat expires", which is just the raw `device.online.changed` event with
// extra steps, so it is rejected instead of quietly reinterpreted.
var ErrInvalidOfflineAlertThreshold = errors.New("after_minutes must be between 1 and 10080")

// Bounds and defaults for the offline-alert threshold.
const (
	DefaultOfflineAlertAfterMinutes = 10
	MinOfflineAlertAfterMinutes     = 1
	MaxOfflineAlertAfterMinutes     = 7 * 24 * 60
)

// offlineAlertBatch bounds one evaluator pass. Rules are one-per-device and
// only exist when an owner asked for one, so this is a safety valve rather
// than an expected limit.
const offlineAlertBatch = 200

// offlineAlertDevice is the per-device projection the evaluator needs.
type offlineAlertDevice struct {
	DeviceID   string `gorm:"column:device_id"`
	DeviceName string `gorm:"column:device_name"`
	LoggedIn   bool   `gorm:"column:logged_in"`
}

// OfflinePresence is the slice of PresenceService the evaluator needs. An
// interface rather than the concrete Redis-backed type keeps the state machine
// testable without a live Redis, which matters because the interesting
// behaviour here is the episode bookkeeping, not the presence lookup.
type OfflinePresence interface {
	IsOnline(ctx context.Context, deviceID string) bool
}

// DeviceOfflineAlertService turns the presence signal into a decision the owner
// actually asked for: "this device has been offline for longer than N minutes".
//
// `device.online.changed` already tells an owner the moment a heartbeat stops,
// but a machine that is rebooting, sleeping or being unplugged for a minute
// produces exactly the same event as one that died. A rule adds the missing
// dimension - how long - and reuses the notification outbox for delivery, so
// the alert inherits durable at-least-once delivery, per-channel fairness,
// dead-letter replay and retention without a second delivery path.
type DeviceOfflineAlertService struct {
	db       *gorm.DB
	presence OfflinePresence
	bus      *EventBus
}

func NewDeviceOfflineAlertService(db *gorm.DB, presence OfflinePresence, bus *EventBus) *DeviceOfflineAlertService {
	return &DeviceOfflineAlertService{db: db, presence: presence, bus: bus}
}

// Get returns the rule for a device, or (nil, nil) when the owner never
// configured one - a missing row means "no alert", not "alert with defaults".
func (s *DeviceOfflineAlertService) Get(ctx context.Context, deviceID string) (*models.DeviceOfflineAlertRule, error) {
	var row models.DeviceOfflineAlertRule
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// Upsert writes the rule for a device, creating the row on first use.
//
// Disabling a rule also clears the episode state: the owner said "stop telling
// me about this device", so the next time they switch it on they get a clean
// slate rather than an alert about an outage that started while the rule was
// off.
func (s *DeviceOfflineAlertService) Upsert(
	ctx context.Context,
	deviceID string,
	ownerID uint,
	enabled bool,
	afterMinutes int,
) (*models.DeviceOfflineAlertRule, error) {
	if afterMinutes < MinOfflineAlertAfterMinutes || afterMinutes > MaxOfflineAlertAfterMinutes {
		return nil, ErrInvalidOfflineAlertThreshold
	}
	now := time.Now().UTC()

	row := models.DeviceOfflineAlertRule{
		DeviceID:     deviceID,
		OwnerID:      ownerID,
		Enabled:      enabled,
		AfterMinutes: afterMinutes,
	}
	updates := map[string]interface{}{
		"owner_id":      ownerID,
		"enabled":       enabled,
		"after_minutes": afterMinutes,
		"updated_at":    now,
	}
	if !enabled {
		updates["offline_since"] = nil
		updates["alerted_at"] = nil
	}
	// The unique index on device_id makes this an upsert without a read-then-
	// write race between two owners' requests.
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Assign(updates).
		FirstOrCreate(&row).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, deviceID)
}

// Delete removes the rule. Deleting a missing rule is not an error: the caller
// asked for "no alert on this device" and that is now true either way.
func (s *DeviceOfflineAlertService) Delete(ctx context.Context, deviceID string) error {
	return s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Delete(&models.DeviceOfflineAlertRule{}).Error
}

// EvaluateOnce advances every enabled rule's episode state and publishes an
// alert for the ones that crossed their threshold. It returns how many alerts
// were published.
//
// Rules are walked in id order in bounded batches so a large install cannot
// turn one tick into an unbounded query.
func (s *DeviceOfflineAlertService) EvaluateOnce(ctx context.Context, now time.Time) (int, error) {
	if s.presence == nil {
		return 0, nil
	}
	now = now.UTC()
	fired := 0
	lastID := uint(0)
	for {
		var rules []models.DeviceOfflineAlertRule
		if err := s.db.WithContext(ctx).
			Where("enabled = ? AND id > ?", true, lastID).
			Order("id ASC").
			Limit(offlineAlertBatch).
			Find(&rules).Error; err != nil {
			return fired, err
		}
		if len(rules) == 0 {
			return fired, nil
		}
		devices, err := s.loadDevices(ctx, rules)
		if err != nil {
			return fired, err
		}
		for _, rule := range rules {
			lastID = rule.ID
			n, err := s.evaluateRule(ctx, rule, devices[rule.DeviceID], now)
			if err != nil {
				return fired, err
			}
			fired += n
		}
		if len(rules) < offlineAlertBatch {
			return fired, nil
		}
	}
}

// loadDevices fetches the rule's devices in one query, so the evaluator does
// not do an N+1 walk over its batch.
func (s *DeviceOfflineAlertService) loadDevices(
	ctx context.Context,
	rules []models.DeviceOfflineAlertRule,
) (map[string]offlineAlertDevice, error) {
	ids := make([]string, 0, len(rules))
	for _, rule := range rules {
		ids = append(ids, rule.DeviceID)
	}
	var rows []offlineAlertDevice
	if err := s.db.WithContext(ctx).
		Table("devices").
		Select("device_id, device_name, logged_in").
		Where("device_id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]offlineAlertDevice, len(rows))
	for _, row := range rows {
		out[row.DeviceID] = row
	}
	return out, nil
}

// evaluateRule is the per-device state machine described on the model. It
// returns 1 when it published an alert.
func (s *DeviceOfflineAlertService) evaluateRule(
	ctx context.Context,
	rule models.DeviceOfflineAlertRule,
	device offlineAlertDevice,
	now time.Time,
) (int, error) {
	// The owner logged the device out (or unbound it). "Not expected online"
	// is not an outage, so clear the episode instead of alerting about a
	// machine the owner deliberately parked.
	if device.DeviceID == "" || !device.LoggedIn {
		return 0, s.clearEpisode(ctx, rule, now)
	}
	if s.presence.IsOnline(ctx, rule.DeviceID) {
		// Back online: end the episode so the next outage alerts again.
		return 0, s.clearEpisode(ctx, rule, now)
	}

	since := rule.OfflineSince
	if since == nil {
		// First offline observation of this episode. The conditional update is
		// what decides ownership: if another instance recorded the start first
		// this affects no rows and we simply come back next tick.
		res := s.db.WithContext(ctx).
			Model(&models.DeviceOfflineAlertRule{}).
			Where("id = ? AND offline_since IS NULL", rule.ID).
			Updates(map[string]interface{}{"offline_since": now, "updated_at": now})
		if res.Error != nil {
			return 0, res.Error
		}
		if res.RowsAffected == 0 {
			return 0, nil
		}
		since = &now
	}

	if now.Sub(*since) < time.Duration(rule.AfterMinutes)*time.Minute {
		return 0, nil
	}
	if rule.AlertedAt != nil {
		// Already told the owner about this episode.
		return 0, nil
	}

	// Claim the alert for this episode. Exactly one caller can move alerted_at
	// from NULL, so a second server instance (or a second tick racing this
	// one) cannot send the same outage twice.
	claimed := s.db.WithContext(ctx).
		Model(&models.DeviceOfflineAlertRule{}).
		Where("id = ? AND alerted_at IS NULL AND offline_since IS NOT NULL", rule.ID).
		Updates(map[string]interface{}{"alerted_at": now, "updated_at": now})
	if claimed.Error != nil {
		return 0, claimed.Error
	}
	if claimed.RowsAffected == 0 {
		return 0, nil
	}

	s.publishOfflineAlert(ctx, rule, device, *since, now)
	return 1, nil
}

// clearEpisode resets the episode state when the device is online again (or is
// no longer expected online).
func (s *DeviceOfflineAlertService) clearEpisode(
	ctx context.Context,
	rule models.DeviceOfflineAlertRule,
	now time.Time,
) error {
	if rule.OfflineSince == nil && rule.AlertedAt == nil {
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&models.DeviceOfflineAlertRule{}).
		Where("id = ?", rule.ID).
		Updates(map[string]interface{}{
			"offline_since": nil,
			"alerted_at":    nil,
			"updated_at":    now,
		}).Error
}

// publishOfflineAlert emits the user-scoped event. It carries no secret and no
// access code, only why the owner is being told.
func (s *DeviceOfflineAlertService) publishOfflineAlert(
	ctx context.Context,
	rule models.DeviceOfflineAlertRule,
	device offlineAlertDevice,
	since time.Time,
	now time.Time,
) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, Event{
		Type:     EventDeviceOfflineSustained,
		UserID:   rule.OwnerID,
		DeviceID: rule.DeviceID,
		Data: map[string]interface{}{
			"device_id":         rule.DeviceID,
			"device_name":       device.DeviceName,
			"offline_since":     since.UTC().Format(time.RFC3339),
			"offline_minutes":   int(now.Sub(since).Minutes()),
			"threshold_minutes": rule.AfterMinutes,
		},
	})
}

// Run evaluates the rules until ctx is cancelled.
func (s *DeviceOfflineAlertService) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.EvaluateOnce(ctx, time.Now()); err != nil {
				log.Printf("[offline-alert] evaluate failed: %v", err)
			}
		}
	}
}
