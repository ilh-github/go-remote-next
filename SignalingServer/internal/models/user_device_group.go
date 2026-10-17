package models

import "time"

// UserDeviceGroup is a private device grouping owned by one user. It is kept
// separate from DeviceGroup because the latter is the administrator-wide
// inventory model and has no owner column.
type UserDeviceGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index;uniqueIndex:uq_user_device_group_name,priority:1" json:"user_id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex:uq_user_device_group_name,priority:2" json:"name"`
	Description string    `gorm:"size:500" json:"description"`
	Color       string    `gorm:"size:20" json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (UserDeviceGroup) TableName() string {
	return "user_device_groups"
}

type UserDeviceGroupMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index;uniqueIndex:uq_user_device_group_member,priority:1" json:"group_id"`
	DeviceID  string    `gorm:"size:20;not null;uniqueIndex:uq_user_device_group_member,priority:2" json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserDeviceGroupMember) TableName() string {
	return "user_device_group_members"
}
