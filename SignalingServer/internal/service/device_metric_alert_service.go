package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidMetricAlert = errors.New("invalid metric alert rule")

const (
	DefaultMetricSustainedMinutes = 5
	MaxMetricSustainedMinutes     = 7 * 24 * 60
	MaxMetricRepeatMinutes        = 7 * 24 * 60
)

type UpsertMetricAlertInput struct {
	Enabled          bool
	CPUPercent       int
	MemoryPercent    int
	DiskPercent      int
	SustainedMinutes int
	RepeatMinutes    int
	SnoozedUntil     *time.Time
}

type IngestMetricInput struct {
	CapturedAt    time.Time
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	DiskMount     string
}

type DeviceMetricAlertService struct {
	db     *gorm.DB
	bus    *EventBus
	outbox EventOutboxWriter
}

func NewDeviceMetricAlertService(db *gorm.DB, bus *EventBus) *DeviceMetricAlertService {
	return &DeviceMetricAlertService{db: db, bus: bus}
}

// WithOutbox makes threshold-alert episode claims and their events atomic.
// Sample broadcasts remain best-effort because they are high-frequency state
// updates; the actionable alert path needs the durable commit boundary.
func (s *DeviceMetricAlertService) WithOutbox(outbox EventOutboxWriter) *DeviceMetricAlertService {
	s.outbox = outbox
	return s
}

func (s *DeviceMetricAlertService) Get(ctx context.Context, deviceID string) (*models.DeviceMetricAlertRule, error) {
	var row models.DeviceMetricAlertRule
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *DeviceMetricAlertService) Latest(ctx context.Context, deviceID string) (*models.DeviceMetricSnapshot, error) {
	var row models.DeviceMetricSnapshot
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *DeviceMetricAlertService) Upsert(
	ctx context.Context,
	deviceID string,
	ownerID uint,
	in UpsertMetricAlertInput,
) (*models.DeviceMetricAlertRule, error) {
	if in.SustainedMinutes < 0 || in.SustainedMinutes > MaxMetricSustainedMinutes {
		return nil, ErrInvalidMetricAlert
	}
	if in.RepeatMinutes < 0 || in.RepeatMinutes > MaxMetricRepeatMinutes {
		return nil, ErrInvalidMetricAlert
	}
	for _, threshold := range []int{in.CPUPercent, in.MemoryPercent, in.DiskPercent} {
		if threshold < 0 || threshold > 100 {
			return nil, ErrInvalidMetricAlert
		}
	}
	if in.Enabled && in.CPUPercent == 0 && in.MemoryPercent == 0 && in.DiskPercent == 0 {
		return nil, ErrInvalidMetricAlert
	}

	now := time.Now().UTC()
	row := models.DeviceMetricAlertRule{
		DeviceID:         deviceID,
		OwnerID:          ownerID,
		Enabled:          in.Enabled,
		CPUPercent:       in.CPUPercent,
		MemoryPercent:    in.MemoryPercent,
		DiskPercent:      in.DiskPercent,
		SustainedMinutes: in.SustainedMinutes,
		RepeatMinutes:    in.RepeatMinutes,
		SnoozedUntil:     in.SnoozedUntil,
	}
	updates := map[string]interface{}{
		"owner_id":          ownerID,
		"enabled":           in.Enabled,
		"cpu_percent":       in.CPUPercent,
		"memory_percent":    in.MemoryPercent,
		"disk_percent":      in.DiskPercent,
		"sustained_minutes": in.SustainedMinutes,
		"repeat_minutes":    in.RepeatMinutes,
		"snoozed_until":     in.SnoozedUntil,
		"updated_at":        now,
	}
	if !in.Enabled {
		for _, column := range metricStateColumns {
			updates[column] = nil
		}
	}
	if err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Assign(updates).
		FirstOrCreate(&row).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, deviceID)
}

func (s *DeviceMetricAlertService) Delete(ctx context.Context, deviceID string) error {
	return s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Delete(&models.DeviceMetricAlertRule{}).Error
}

// Ingest stores the latest sample and advances the rule state in the same
// request. Hosts report on a timer, so this keeps the alert latency bounded by
// the reporting interval without a second background evaluator.
func (s *DeviceMetricAlertService) Ingest(
	ctx context.Context,
	deviceID string,
	in IngestMetricInput,
) (*models.DeviceMetricSnapshot, error) {
	if err := validateMetricSample(in); err != nil {
		return nil, err
	}
	if in.CapturedAt.IsZero() {
		in.CapturedAt = time.Now().UTC()
	}
	in.CapturedAt = in.CapturedAt.UTC()
	in.DiskMount = strings.TrimSpace(in.DiskMount)

	snapshot := models.DeviceMetricSnapshot{
		DeviceID:      deviceID,
		CapturedAt:    in.CapturedAt,
		CPUPercent:    in.CPUPercent,
		MemoryPercent: in.MemoryPercent,
		DiskPercent:   in.DiskPercent,
		DiskMount:     in.DiskMount,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"captured_at", "cpu_percent", "memory_percent", "disk_percent",
			"disk_mount", "updated_at",
		}),
	}).Create(&snapshot).Error; err != nil {
		return nil, err
	}

	var device models.Device
	if err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &snapshot, nil
		}
		return nil, err
	}
	if device.UserID == nil || !device.LoggedIn {
		return &snapshot, nil
	}

	if s.bus != nil {
		s.bus.Publish(ctx, Event{
			Type:     EventDeviceMetricsUpdated,
			UserID:   *device.UserID,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"device_id":      deviceID,
				"captured_at":    snapshot.CapturedAt.Format(time.RFC3339),
				"cpu_percent":    snapshot.CPUPercent,
				"memory_percent": snapshot.MemoryPercent,
				"disk_percent":   snapshot.DiskPercent,
				"disk_mount":     snapshot.DiskMount,
			},
		})
	}

	rule, err := s.Get(ctx, deviceID)
	if err != nil || rule == nil || !rule.Enabled || rule.OwnerID != *device.UserID {
		return &snapshot, err
	}

	states := metricRuleStates(rule)
	for _, metric := range []struct {
		name      string
		value     float64
		threshold int
		state     *metricRuleState
	}{
		{"cpu", snapshot.CPUPercent, rule.CPUPercent, states["cpu"]},
		{"memory", snapshot.MemoryPercent, rule.MemoryPercent, states["memory"]},
		{"disk", snapshot.DiskPercent, rule.DiskPercent, states["disk"]},
	} {
		if err := s.evaluateMetric(ctx, *rule, device, snapshot, metric.name, metric.value, metric.threshold, metric.state); err != nil {
			return nil, err
		}
	}
	return &snapshot, nil
}

type metricRuleState struct {
	sinceColumn   string
	alertedColumn string
	since         *time.Time
	alertedAt     *time.Time
}

var metricStateColumns = []string{
	"cpu_exceeded_since", "cpu_alerted_at",
	"memory_exceeded_since", "memory_alerted_at",
	"disk_exceeded_since", "disk_alerted_at",
}

func metricRuleStates(rule *models.DeviceMetricAlertRule) map[string]*metricRuleState {
	return map[string]*metricRuleState{
		"cpu": {
			sinceColumn: "cpu_exceeded_since", alertedColumn: "cpu_alerted_at",
			since: rule.CPUExceededSince, alertedAt: rule.CPUAlertedAt,
		},
		"memory": {
			sinceColumn: "memory_exceeded_since", alertedColumn: "memory_alerted_at",
			since: rule.MemoryExceededSince, alertedAt: rule.MemoryAlertedAt,
		},
		"disk": {
			sinceColumn: "disk_exceeded_since", alertedColumn: "disk_alerted_at",
			since: rule.DiskExceededSince, alertedAt: rule.DiskAlertedAt,
		},
	}
}

func (s *DeviceMetricAlertService) evaluateMetric(
	ctx context.Context,
	rule models.DeviceMetricAlertRule,
	device models.Device,
	snapshot models.DeviceMetricSnapshot,
	metric string,
	value float64,
	threshold int,
	state *metricRuleState,
) error {
	if threshold <= 0 || value < float64(threshold) {
		if state.since == nil && state.alertedAt == nil {
			return nil
		}
		return s.db.WithContext(ctx).Model(&models.DeviceMetricAlertRule{}).
			Where("device_id = ?", rule.DeviceID).
			Updates(map[string]interface{}{
				state.sinceColumn:   nil,
				state.alertedColumn: nil,
				"updated_at":        time.Now().UTC(),
			}).Error
	}

	now := snapshot.CapturedAt
	if state.since == nil {
		res := s.db.WithContext(ctx).Model(&models.DeviceMetricAlertRule{}).
			Where("device_id = ? AND "+state.sinceColumn+" IS NULL", rule.DeviceID).
			Updates(map[string]interface{}{
				state.sinceColumn:   now,
				state.alertedColumn: nil,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Another process advanced the episode first. Leave this sample to
			// the winner instead of emitting a duplicate alert.
			return nil
		}
		since := now
		state.since = &since
		if rule.SustainedMinutes > 0 {
			return nil
		}
	}
	if now.Before(state.since.Add(time.Duration(rule.SustainedMinutes) * time.Minute)) {
		return nil
	}
	if rule.SnoozedUntil != nil && now.Before(*rule.SnoozedUntil) {
		return nil
	}

	reminder := state.alertedAt != nil
	if reminder {
		if rule.RepeatMinutes <= 0 {
			return nil
		}
		if now.Before(state.alertedAt.Add(time.Duration(rule.RepeatMinutes) * time.Minute)) {
			return nil
		}
	}

	data := map[string]interface{}{
		"device_id":         rule.DeviceID,
		"device_name":       device.DeviceName,
		"metric":            metric,
		"value_percent":     value,
		"threshold_percent": threshold,
		"sustained_minutes": rule.SustainedMinutes,
		"captured_at":       snapshot.CapturedAt.Format(time.RFC3339),
		"reminder":          reminder,
	}
	evt := Event{
		Type:     EventDeviceMetricAlert,
		UserID:   rule.OwnerID,
		DeviceID: rule.DeviceID,
		Data:     data,
	}
	publishDirect := false
	claimedAlert := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.DeviceMetricAlertRule{}).
			Where("device_id = ? AND "+state.sinceColumn+" = ?", rule.DeviceID, *state.since)
		if state.alertedAt == nil {
			claim = claim.Where(state.alertedColumn + " IS NULL")
		} else {
			claim = claim.Where(state.alertedColumn+" = ?", *state.alertedAt)
		}
		res := claim.Update(state.alertedColumn, now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		claimedAlert = true
		if s.outbox != nil {
			_, err := s.outbox.EnqueueTx(ctx, tx, evt)
			return err
		}
		publishDirect = s.bus != nil
		return nil
	})
	if err != nil {
		return err
	}
	if claimedAlert && publishDirect {
		s.bus.Publish(ctx, evt)
	}
	return nil
}

func validateMetricSample(in IngestMetricInput) error {
	for _, value := range []float64{in.CPUPercent, in.MemoryPercent, in.DiskPercent} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return ErrInvalidMetricAlert
		}
	}
	return nil
}
