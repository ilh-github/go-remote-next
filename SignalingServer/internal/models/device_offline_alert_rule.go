package models

import "time"

// DeviceOfflineAlertRule is one owner-scoped rule: "tell me when this device
// has been offline for longer than after_minutes".
//
// The row also carries the evaluator's episode state, because "offline since
// when" is not something presence can answer. Redis only says whether the
// heartbeat and WS keys exist *right now*; a rule that asked "was this device
// offline for ten minutes?" without recording the start of the episode would
// either fire immediately on the first offline observation or never fire at
// all.
//
//	offline_since - set the first time the evaluator saw the device offline,
//	                cleared when it comes back. NULL means "not in an episode".
//	alerted_at    - set when the alert for the current episode was published.
//	                Cleared together with offline_since, so the next outage
//	                alerts again instead of being suppressed forever.
//
// Both fields are written with conditional UPDATEs so two server instances
// cannot both start an episode or both fire one alert.
type DeviceOfflineAlertRule struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	DeviceID     string `gorm:"uniqueIndex;size:20;not null" json:"device_id"`
	OwnerID      uint   `gorm:"index;not null" json:"owner_id"`
	Enabled      bool   `gorm:"not null;default:true" json:"enabled"`
	AfterMinutes int    `gorm:"not null;default:10" json:"after_minutes"`

	OfflineSince *time.Time `json:"offline_since"`
	AlertedAt    *time.Time `json:"last_alerted_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceOfflineAlertRule) TableName() string {
	return "device_offline_alert_rules"
}
