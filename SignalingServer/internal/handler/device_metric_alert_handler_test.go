package handler

import (
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

func TestMetricAlertViewDefaultsWithoutRule(t *testing.T) {
	view := metricAlertView("123456789", nil, nil)
	if view.DeviceID != "123456789" {
		t.Fatalf("device_id = %q, want requested device", view.DeviceID)
	}
	if view.Enabled {
		t.Fatal("a missing rule must be disabled")
	}
	if view.SustainedMinutes != 5 {
		t.Fatalf("sustained_minutes = %d, want 5", view.SustainedMinutes)
	}
	if view.Current != nil {
		t.Fatalf("current = %+v, want nil before the first sample", view.Current)
	}
	if view.States.CPU.ExceededSince != nil || view.States.Memory.ExceededSince != nil ||
		view.States.Disk.ExceededSince != nil {
		t.Fatal("missing rule must not report metric episodes")
	}
}

func TestMetricAlertViewCarriesCurrentSampleAndIndependentStates(t *testing.T) {
	captured := time.Now().UTC().Truncate(time.Microsecond)
	cpuSince := captured.Add(-10 * time.Minute)
	memorySince := captured.Add(-3 * time.Minute)
	cpuAlerted := captured.Add(-5 * time.Minute)
	rule := &models.DeviceMetricAlertRule{
		DeviceID:            "123456789",
		Enabled:             true,
		CPUPercent:          90,
		MemoryPercent:       80,
		SustainedMinutes:    5,
		RepeatMinutes:       30,
		CPUExceededSince:    &cpuSince,
		CPUAlertedAt:        &cpuAlerted,
		MemoryExceededSince: &memorySince,
	}
	snapshot := &models.DeviceMetricSnapshot{
		DeviceID:      "123456789",
		CapturedAt:    captured,
		CPUPercent:    96.5,
		MemoryPercent: 84,
		DiskPercent:   42,
		DiskMount:     "/",
	}

	view := metricAlertView("123456789", rule, snapshot)
	if !view.Enabled || view.CPUPercent != 90 || view.MemoryPercent != 80 {
		t.Fatalf("view = %+v, want enabled CPU 90 / memory 80", view)
	}
	if view.Current == nil || view.Current.CPUPercent != 96.5 || view.Current.DiskMount != "/" {
		t.Fatalf("current = %+v, want latest snapshot", view.Current)
	}
	if view.States.CPU.ExceededSince == nil || !view.States.CPU.ExceededSince.Equal(cpuSince) {
		t.Fatalf("CPU state = %+v, want since %v", view.States.CPU, cpuSince)
	}
	if view.States.CPU.LastAlertedAt == nil || !view.States.CPU.LastAlertedAt.Equal(cpuAlerted) {
		t.Fatalf("CPU alerted state = %+v, want %v", view.States.CPU, cpuAlerted)
	}
	if view.States.Memory.ExceededSince == nil || !view.States.Memory.ExceededSince.Equal(memorySince) {
		t.Fatalf("memory state = %+v, want since %v", view.States.Memory, memorySince)
	}
	if view.States.Disk.ExceededSince != nil {
		t.Fatalf("disk state = %+v, want idle", view.States.Disk)
	}
}
