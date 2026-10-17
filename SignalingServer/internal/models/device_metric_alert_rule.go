package models

import "time"

// DeviceMetricAlertRule is one owner-scoped CPU/memory/disk threshold rule.
// Each metric keeps its own episode state so a recovered CPU reading does not
// erase an ongoing memory or disk episode. A zero threshold disables that
// metric; at least one threshold must be positive when the rule is enabled.
type DeviceMetricAlertRule struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"uniqueIndex;size:20;not null" json:"device_id"`
	OwnerID  uint   `gorm:"index;not null" json:"owner_id"`
	Enabled  bool   `gorm:"not null;default:false" json:"enabled"`

	CPUPercent       int `gorm:"not null;default:0" json:"cpu_percent"`
	MemoryPercent    int `gorm:"not null;default:0" json:"memory_percent"`
	DiskPercent      int `gorm:"not null;default:0" json:"disk_percent"`
	SustainedMinutes int `gorm:"not null" json:"sustained_minutes"`
	RepeatMinutes    int `gorm:"not null;default:0" json:"repeat_minutes"`

	SnoozedUntil *time.Time `json:"snoozed_until"`

	CPUExceededSince    *time.Time `json:"-"`
	CPUAlertedAt        *time.Time `json:"-"`
	MemoryExceededSince *time.Time `json:"-"`
	MemoryAlertedAt     *time.Time `json:"-"`
	DiskExceededSince   *time.Time `json:"-"`
	DiskAlertedAt       *time.Time `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DeviceMetricAlertRule) TableName() string {
	return "device_metric_alert_rules"
}
