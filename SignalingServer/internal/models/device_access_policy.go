package models

import "time"

// DeviceAccessPolicy is the device owner's control over *who may open a
// session on this machine*, enforced at the same choke point that writes the
// access log (`access-code:verify`). The access log answers "who tried"; this
// answers "who is allowed".
//
// One row per device; a missing row means "no policy" (remote access allowed
// from anywhere), so existing devices keep working after the upgrade.
type DeviceAccessPolicy struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:9;not null;uniqueIndex" json:"device_id"`
	// OwnerID is the user who last changed the policy; kept for audit value
	// when a device changes hands.
	OwnerID uint `gorm:"index" json:"owner_id"`
	// RemoteDisabled blocks every new session regardless of the access code.
	RemoteDisabled bool `gorm:"not null;default:false" json:"remote_disabled"`
	// AllowIPs is a JSON array of exact IPs and/or CIDR blocks. Empty means
	// "any client IP". Entries are matched against the controller's address
	// as seen by the server.
	AllowIPs  string    `gorm:"type:text" json:"allow_ips"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceAccessPolicy) TableName() string {
	return "device_access_policies"
}
