package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMetricAlertTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run metric-alert tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	migration, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if err := db.Exec(string(migration)).Error; err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM device_metric_alert_rules").Error; err != nil {
		t.Fatalf("clear rules: %v", err)
	}
	if err := db.Exec("DELETE FROM device_metric_snapshots").Error; err != nil {
		t.Fatalf("clear snapshots: %v", err)
	}
	return db
}

func seedMetricAlertDevice(t *testing.T, db *gorm.DB, deviceID string, userID uint) {
	t.Helper()
	username := fmt.Sprintf("qa-metric-%d", userID)
	if err := db.Exec(
		"INSERT INTO users (id, username, password) VALUES (?, ?, 'test-hash') ON CONFLICT (id) DO NOTHING",
		userID, username,
	).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec("DELETE FROM devices WHERE device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("clear device: %v", err)
	}
	owner := userID
	device := models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		DeviceName: "metric-" + deviceID,
		UserID:     &owner,
		LoggedIn:   true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

func metricInput(at time.Time, cpu, memory, disk float64) IngestMetricInput {
	return IngestMetricInput{
		CapturedAt:    at,
		CPUPercent:    cpu,
		MemoryPercent: memory,
		DiskPercent:   disk,
		DiskMount:     "/",
	}
}

func loadMetricAlertRule(t *testing.T, db *gorm.DB, deviceID string) models.DeviceMetricAlertRule {
	t.Helper()
	var rule models.DeviceMetricAlertRule
	if err := db.Where("device_id = ?", deviceID).First(&rule).Error; err != nil {
		t.Fatalf("load metric rule: %v", err)
	}
	return rule
}

func TestMetricAlertStateMachine(t *testing.T) {
	db := newMetricAlertTestDB(t)
	const deviceID = "910000001"
	seedMetricAlertDevice(t, db, deviceID, 11)

	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 8)}
	bus.Subscribe(captured)
	svc := NewDeviceMetricAlertService(db, bus)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Upsert(ctx, deviceID, 11, UpsertMetricAlertInput{
		Enabled:          true,
		CPUPercent:       90,
		MemoryPercent:    90,
		DiskPercent:      95,
		SustainedMinutes: 5,
		RepeatMinutes:    30,
	}); err != nil {
		t.Fatalf("upsert rule: %v", err)
	}

	// A normal sample stores the snapshot and starts no episodes.
	snapshot, err := svc.Ingest(ctx, deviceID, metricInput(start, 50, 40, 30))
	if err != nil {
		t.Fatalf("normal sample: %v", err)
	}
	if snapshot.DeviceID != deviceID || snapshot.CPUPercent != 50 {
		t.Fatalf("snapshot = %+v, want device %s cpu 50", snapshot, deviceID)
	}
	if fired := drainMetricEvents(captured); fired != 0 {
		t.Fatalf("normal sample fired %d alerts, want 0", fired)
	}

	// The first breach starts the CPU episode without alerting yet.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(time.Minute), 95, 40, 30)); err != nil {
		t.Fatalf("cpu breach: %v", err)
	}
	rule := loadMetricAlertRule(t, db, deviceID)
	if rule.CPUExceededSince == nil || !rule.CPUExceededSince.Equal(start.Add(time.Minute)) {
		t.Fatalf("cpu_exceeded_since = %v, want %v", rule.CPUExceededSince, start.Add(time.Minute))
	}
	if fired := drainMetricEvents(captured); fired != 0 {
		t.Fatalf("first CPU breach fired %d alerts, want 0", fired)
	}

	// Past the sustained threshold one CPU alert fires.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(6*time.Minute), 96, 40, 30)); err != nil {
		t.Fatalf("sustained cpu: %v", err)
	}
	evt := nextMetricEvent(t, captured)
	if evt.Type != EventDeviceMetricAlert {
		t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceMetricAlert)
	}
	if evt.UserID != 11 || evt.DeviceID != deviceID {
		t.Fatalf("event routing = %+v, want user 11 / device %s", evt, deviceID)
	}
	if got := evt.Data["metric"]; got != "cpu" {
		t.Fatalf("metric = %v, want cpu", got)
	}
	if got := evt.Data["value_percent"]; got != 96.0 {
		t.Fatalf("value_percent = %v, want 96", got)
	}
	if got := evt.Data["threshold_percent"]; got != 90 {
		t.Fatalf("threshold_percent = %v, want 90", got)
	}
	if got := evt.Data["reminder"]; got != false {
		t.Fatalf("reminder = %v, want false", got)
	}

	// Still breached but inside the repeat interval: no second alert.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(7*time.Minute), 97, 40, 30)); err != nil {
		t.Fatalf("inside repeat: %v", err)
	}
	if fired := drainMetricEvents(captured); fired != 0 {
		t.Fatalf("inside repeat fired %d alerts, want 0", fired)
	}

	// Past the repeat interval: a reminder, flagged as such.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(37*time.Minute), 98, 40, 30)); err != nil {
		t.Fatalf("repeat reminder: %v", err)
	}
	evt = nextMetricEvent(t, captured)
	if got := evt.Data["reminder"]; got != true {
		t.Fatalf("reminder = %v, want true", got)
	}

	// Recovery clears the CPU episode, so a later breach is a new episode.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(38*time.Minute), 10, 40, 30)); err != nil {
		t.Fatalf("recovery: %v", err)
	}
	rule = loadMetricAlertRule(t, db, deviceID)
	if rule.CPUExceededSince != nil || rule.CPUAlertedAt != nil {
		t.Fatalf("CPU state survived recovery: %+v", rule)
	}

	// Memory is independent: its first breach does not inherit CPU state.
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(39*time.Minute), 10, 95, 30)); err != nil {
		t.Fatalf("memory breach: %v", err)
	}
	rule = loadMetricAlertRule(t, db, deviceID)
	if rule.MemoryExceededSince == nil || rule.CPUExceededSince != nil {
		t.Fatalf("memory episode should be independent: %+v", rule)
	}
}

func TestMetricSnapshotKeepsOnlyLatestSample(t *testing.T) {
	db := newMetricAlertTestDB(t)
	const deviceID = "910000002"
	seedMetricAlertDevice(t, db, deviceID, 12)

	svc := NewDeviceMetricAlertService(db, NewEventBus(nil))
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Ingest(ctx, deviceID, metricInput(start, 10, 20, 30)); err != nil {
		t.Fatalf("first sample: %v", err)
	}
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start.Add(time.Minute), 40, 50, 60)); err != nil {
		t.Fatalf("second sample: %v", err)
	}

	var count int64
	if err := db.Model(&models.DeviceMetricSnapshot{}).
		Where("device_id = ?", deviceID).Count(&count).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if count != 1 {
		t.Fatalf("snapshot rows = %d, want 1", count)
	}
	latest, err := svc.Latest(ctx, deviceID)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest == nil || latest.CPUPercent != 40 || latest.MemoryPercent != 50 || latest.DiskPercent != 60 {
		t.Fatalf("latest = %+v, want 40/50/60", latest)
	}
}

func TestMetricSnapshotPublishesUpdatedEvent(t *testing.T) {
	db := newMetricAlertTestDB(t)
	const deviceID = "910000005"
	seedMetricAlertDevice(t, db, deviceID, 15)

	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 2)}
	bus.Subscribe(captured)
	svc := NewDeviceMetricAlertService(db, bus)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Ingest(ctx, deviceID, metricInput(start, 12.5, 34.5, 78.0)); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	select {
	case evt := <-captured.events:
		if evt.Type != EventDeviceMetricsUpdated {
			t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceMetricsUpdated)
		}
		if evt.UserID != 15 || evt.DeviceID != deviceID {
			t.Fatalf("event routing = %+v, want user 15 / device %s", evt, deviceID)
		}
		if evt.Data["cpu_percent"] != 12.5 ||
			evt.Data["memory_percent"] != 34.5 ||
			evt.Data["disk_percent"] != 78.0 {
			t.Fatalf("updated event data = %+v", evt.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a device.metrics.updated event")
	}
}

func TestMetricAlertZeroSustainedAlertsOnFirstSample(t *testing.T) {
	db := newMetricAlertTestDB(t)
	const deviceID = "910000004"
	seedMetricAlertDevice(t, db, deviceID, 14)

	bus := NewEventBus(nil)
	captured := &captureSubscriber{events: make(chan Event, 2)}
	bus.Subscribe(captured)
	svc := NewDeviceMetricAlertService(db, bus)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := svc.Upsert(ctx, deviceID, 14, UpsertMetricAlertInput{
		Enabled: true, CPUPercent: 80, SustainedMinutes: 0,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start, 85, 20, 20)); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	evt := nextMetricEvent(t, captured)
	if got := evt.Data["metric"]; got != "cpu" {
		t.Fatalf("metric = %v, want cpu", got)
	}
}

func TestMetricAlertValidationAndDisable(t *testing.T) {
	db := newMetricAlertTestDB(t)
	const deviceID = "910000003"
	seedMetricAlertDevice(t, db, deviceID, 13)

	svc := NewDeviceMetricAlertService(db, NewEventBus(nil))
	ctx := context.Background()

	for _, bad := range []UpsertMetricAlertInput{
		{Enabled: true, SustainedMinutes: 5},
		{Enabled: true, CPUPercent: -1, SustainedMinutes: 5},
		{Enabled: true, MemoryPercent: 101, SustainedMinutes: 5},
		{Enabled: true, DiskPercent: 90, SustainedMinutes: -1},
		{Enabled: true, DiskPercent: 90, SustainedMinutes: 5, RepeatMinutes: -1},
	} {
		_, err := svc.Upsert(ctx, deviceID, 13, bad)
		if !errors.Is(err, ErrInvalidMetricAlert) {
			t.Fatalf("input %+v: err = %v, want ErrInvalidMetricAlert", bad, err)
		}
	}

	start := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := svc.Upsert(ctx, deviceID, 13, UpsertMetricAlertInput{
		Enabled: true, CPUPercent: 90, SustainedMinutes: 1,
	}); err != nil {
		t.Fatalf("upsert valid rule: %v", err)
	}
	if _, err := svc.Ingest(ctx, deviceID, metricInput(start, 95, 10, 10)); err != nil {
		t.Fatalf("start CPU episode: %v", err)
	}
	if rule := loadMetricAlertRule(t, db, deviceID); rule.CPUExceededSince == nil {
		t.Fatal("expected CPU episode before disabling")
	}

	if _, err := svc.Upsert(ctx, deviceID, 13, UpsertMetricAlertInput{
		Enabled: false, CPUPercent: 90, SustainedMinutes: 1,
	}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	rule := loadMetricAlertRule(t, db, deviceID)
	if rule.Enabled {
		t.Fatal("rule should be disabled")
	}
	if rule.CPUExceededSince != nil || rule.CPUAlertedAt != nil ||
		rule.MemoryExceededSince != nil || rule.MemoryAlertedAt != nil ||
		rule.DiskExceededSince != nil || rule.DiskAlertedAt != nil {
		t.Fatalf("disabling must clear all metric episodes: %+v", rule)
	}

	if err := svc.Delete(ctx, deviceID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, deviceID); err != nil {
		t.Fatalf("second delete should be a no-op: %v", err)
	}
	if rule, err := svc.Get(ctx, deviceID); err != nil || rule != nil {
		t.Fatalf("Get after delete = %+v (err=%v), want nil", rule, err)
	}
}

func drainMetricEvents(captured *captureSubscriber) int {
	count := 0
	for {
		select {
		case evt := <-captured.events:
			if evt.Type == EventDeviceMetricAlert {
				count++
			}
		default:
			return count
		}
	}
}

func nextMetricEvent(t *testing.T, captured *captureSubscriber) Event {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case evt := <-captured.events:
			if evt.Type == EventDeviceMetricAlert {
				return evt
			}
		case <-deadline:
			t.Fatal("expected a metric alert event")
			return Event{}
		}
	}
}
