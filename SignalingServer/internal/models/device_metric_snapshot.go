package models

import "time"

// DeviceMetricSnapshot is the most recent resource sample reported by a host.
// The row is one-per-device on purpose: metric alerts only need the latest
// value, and retaining a full time series without a query/retention surface
// would turn a small operational feature into an unbounded metrics database.
type DeviceMetricSnapshot struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DeviceID      string    `gorm:"uniqueIndex;size:20;not null" json:"device_id"`
	CapturedAt    time.Time `gorm:"index;not null" json:"captured_at"`
	CPUPercent    float64   `gorm:"not null;default:0" json:"cpu_percent"`
	MemoryPercent float64   `gorm:"not null;default:0" json:"memory_percent"`
	DiskPercent   float64   `gorm:"not null;default:0" json:"disk_percent"`
	DiskMount     string    `gorm:"size:256;not null;default:''" json:"disk_mount"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (DeviceMetricSnapshot) TableName() string {
	return "device_metric_snapshots"
}
