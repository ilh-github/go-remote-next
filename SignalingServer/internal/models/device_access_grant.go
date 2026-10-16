package models

import "time"

// DeviceAccessGrant is a time-boxed, optionally single-use code that admits a
// controller to one device without handing over the device's own (rotating)
// access code. The owner creates it; VerifyAccessCode accepts either the device
// code or an active grant.
//
// Grants are deliberately additive to the device policy, not a bypass: the
// policy gate (remote switch, IP allow list, windows, exception dates) runs
// before the code comparison, so a grant can never open a device that the owner
// has switched off.
type DeviceAccessGrant struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:20;not null;index" json:"device_id"`
	// OwnerID is the user who issued the grant (audit value; unrelated to the
	// guest who redeems it).
	OwnerID uint `gorm:"index" json:"owner_id"`
	// Code is what the guest types. It is shown to the owner on creation (and
	// in the list) because the owner has to communicate it.
	Code  string `gorm:"size:64;not null;index" json:"code"`
	Label string `gorm:"size:100" json:"label"`
	// ExpiresAt bounds the grant in time; a grant past this instant is refused.
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	// MaxUses caps how many sessions the grant may start. 0 means "no cap
	// within the window".
	MaxUses int `gorm:"not null;default:0" json:"max_uses"`
	// Uses counts sessions actually started with the grant.
	Uses int `gorm:"not null;default:0" json:"uses"`
	// RevokedAt lets the owner cancel a grant before it expires.
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (DeviceAccessGrant) TableName() string {
	return "device_access_grants"
}

// UsableAt reports whether the grant may admit a controller at the given time.
func (g DeviceAccessGrant) UsableAt(now time.Time) bool {
	if g.RevokedAt != nil {
		return false
	}
	if !now.Before(g.ExpiresAt) {
		return false
	}
	return g.MaxUses <= 0 || g.Uses < g.MaxUses
}
