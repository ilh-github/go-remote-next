package models

import "time"

// UserFavorite records a user's favorite (bookmarked) device for quick access.
type UserFavorite struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index;uniqueIndex:idx_user_favorite_user_device" json:"user_id"`
	DeviceID       string    `gorm:"size:9;not null;index;uniqueIndex:idx_user_favorite_user_device" json:"device_id"`
	DeviceName     string    `gorm:"size:100" json:"device_name"`     // user-defined label/remark
	AccessPassword string    `gorm:"size:255" json:"access_password"` // stored plaintext for quick-connect
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// BelongsTo
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName overrides the default table name.
func (UserFavorite) TableName() string {
	return "user_favorites"
}
