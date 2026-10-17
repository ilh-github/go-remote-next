package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

var (
	ErrUserDeviceGroupNotFound   = errors.New("device group not found")
	ErrUserDeviceGroupNameExists = errors.New("device group name already exists")
	ErrUserDeviceGroupInvalid    = errors.New("device group input is invalid")
	ErrUserDeviceNotOwned        = errors.New("device is not owned by this user")
)

type UserDeviceGroupInput struct {
	Name        string
	Description string
	Color       string
}

type UserDeviceGroupPatch struct {
	Name        *string
	Description *string
	Color       *string
}

type UserDeviceGroupView struct {
	models.UserDeviceGroup
	DeviceIDs   []string `json:"device_ids"`
	DeviceCount int      `json:"device_count"`
}

type UserDeviceGroupService struct {
	db *gorm.DB
}

func NewUserDeviceGroupService(db *gorm.DB) *UserDeviceGroupService {
	return &UserDeviceGroupService{db: db}
}

func (s *UserDeviceGroupService) List(ctx context.Context, userID uint) ([]UserDeviceGroupView, error) {
	var groups []models.UserDeviceGroup
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("name asc, id asc").
		Find(&groups).Error; err != nil {
		return nil, err
	}
	memberships, err := s.memberships(ctx, groupIDs(groups))
	if err != nil {
		return nil, err
	}
	out := make([]UserDeviceGroupView, len(groups))
	for i, group := range groups {
		out[i] = UserDeviceGroupView{
			UserDeviceGroup: group,
			DeviceIDs:       memberships[group.ID],
			DeviceCount:     len(memberships[group.ID]),
		}
	}
	return out, nil
}

func (s *UserDeviceGroupService) Create(ctx context.Context, userID uint, in UserDeviceGroupInput) (*UserDeviceGroupView, error) {
	in, err := normalizeUserDeviceGroupInput(in)
	if err != nil {
		return nil, err
	}
	var existing int64
	if err := s.db.WithContext(ctx).Model(&models.UserDeviceGroup{}).
		Where("user_id = ? AND name = ?", userID, in.Name).
		Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrUserDeviceGroupNameExists
	}

	group := &models.UserDeviceGroup{
		UserID:      userID,
		Name:        in.Name,
		Description: in.Description,
		Color:       in.Color,
	}
	if err := s.db.WithContext(ctx).Create(group).Error; err != nil {
		return nil, fmt.Errorf("create user device group: %w", err)
	}
	return &UserDeviceGroupView{UserDeviceGroup: *group, DeviceIDs: []string{}}, nil
}

func (s *UserDeviceGroupService) Update(ctx context.Context, userID, groupID uint, patch UserDeviceGroupPatch) (*UserDeviceGroupView, error) {
	var out *UserDeviceGroupView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, err := findUserDeviceGroup(tx, userID, groupID)
		if err != nil {
			return err
		}
		if patch.Name != nil {
			name := strings.TrimSpace(*patch.Name)
			if err := validateUserDeviceGroupName(name); err != nil {
				return err
			}
			if name != group.Name {
				var existing int64
				if err := tx.Model(&models.UserDeviceGroup{}).
					Where("user_id = ? AND name = ? AND id <> ?", userID, name, groupID).
					Count(&existing).Error; err != nil {
					return err
				}
				if existing > 0 {
					return ErrUserDeviceGroupNameExists
				}
			}
			group.Name = name
		}
		if patch.Description != nil {
			description := strings.TrimSpace(*patch.Description)
			if utf8.RuneCountInString(description) > 500 {
				return fmt.Errorf("%w: description is longer than 500 characters", ErrUserDeviceGroupInvalid)
			}
			group.Description = description
		}
		if patch.Color != nil {
			color := strings.TrimSpace(*patch.Color)
			if utf8.RuneCountInString(color) > 20 {
				return fmt.Errorf("%w: color is longer than 20 characters", ErrUserDeviceGroupInvalid)
			}
			group.Color = color
		}
		group.UpdatedAt = time.Now().UTC()
		if err := tx.Save(group).Error; err != nil {
			return err
		}
		members, err := loadGroupDeviceIDs(tx, group.ID)
		if err != nil {
			return err
		}
		out = &UserDeviceGroupView{UserDeviceGroup: *group, DeviceIDs: members, DeviceCount: len(members)}
		return nil
	})
	return out, err
}

func (s *UserDeviceGroupService) Delete(ctx context.Context, userID, groupID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, err := findUserDeviceGroup(tx, userID, groupID)
		if err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", group.ID).Delete(&models.UserDeviceGroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.UserDeviceGroup{}, group.ID).Error
	})
}

func (s *UserDeviceGroupService) ReplaceDevices(ctx context.Context, userID, groupID uint, deviceIDs []string) (*UserDeviceGroupView, error) {
	deviceIDs = normalizeDeviceIDs(deviceIDs)
	var out *UserDeviceGroupView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, err := findUserDeviceGroup(tx, userID, groupID)
		if err != nil {
			return err
		}
		if len(deviceIDs) > 0 {
			var owned []string
			if err := tx.Model(&models.Device{}).
				Clauses(lockForUpdate()).
				Where("device_id IN ? AND user_id = ?", deviceIDs, userID).
				Pluck("device_id", &owned).Error; err != nil {
				return err
			}
			if len(owned) != len(deviceIDs) {
				return ErrUserDeviceNotOwned
			}
		}
		if err := tx.Where("group_id = ?", group.ID).
			Delete(&models.UserDeviceGroupMember{}).Error; err != nil {
			return err
		}
		if len(deviceIDs) > 0 {
			members := make([]models.UserDeviceGroupMember, 0, len(deviceIDs))
			for _, deviceID := range deviceIDs {
				members = append(members, models.UserDeviceGroupMember{
					GroupID:  group.ID,
					DeviceID: deviceID,
				})
			}
			if err := tx.Create(&members).Error; err != nil {
				return err
			}
		}
		group.UpdatedAt = time.Now().UTC()
		if err := tx.Model(group).Update("updated_at", group.UpdatedAt).Error; err != nil {
			return err
		}
		out = &UserDeviceGroupView{UserDeviceGroup: *group, DeviceIDs: deviceIDs, DeviceCount: len(deviceIDs)}
		return nil
	})
	return out, err
}

func normalizeUserDeviceGroupInput(in UserDeviceGroupInput) (UserDeviceGroupInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Color = strings.TrimSpace(in.Color)
	if err := validateUserDeviceGroupName(in.Name); err != nil {
		return UserDeviceGroupInput{}, err
	}
	if utf8.RuneCountInString(in.Description) > 500 {
		return UserDeviceGroupInput{}, fmt.Errorf("%w: description is longer than 500 characters", ErrUserDeviceGroupInvalid)
	}
	if utf8.RuneCountInString(in.Color) > 20 {
		return UserDeviceGroupInput{}, fmt.Errorf("%w: color is longer than 20 characters", ErrUserDeviceGroupInvalid)
	}
	return in, nil
}

func validateUserDeviceGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrUserDeviceGroupInvalid)
	}
	if utf8.RuneCountInString(name) > 100 {
		return fmt.Errorf("%w: name is longer than 100 characters", ErrUserDeviceGroupInvalid)
	}
	return nil
}

func findUserDeviceGroup(tx *gorm.DB, userID, groupID uint) (*models.UserDeviceGroup, error) {
	var group models.UserDeviceGroup
	err := tx.Clauses(lockForUpdate()).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserDeviceGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func groupIDs(groups []models.UserDeviceGroup) []uint {
	ids := make([]uint, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids
}

func (s *UserDeviceGroupService) memberships(ctx context.Context, groupIDs []uint) (map[uint][]string, error) {
	out := make(map[uint][]string, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	var members []models.UserDeviceGroupMember
	if err := s.db.WithContext(ctx).
		Where("group_id IN ?", groupIDs).
		Order("id asc").
		Find(&members).Error; err != nil {
		return nil, err
	}
	for _, member := range members {
		out[member.GroupID] = append(out[member.GroupID], member.DeviceID)
	}
	return out, nil
}

func loadGroupDeviceIDs(tx *gorm.DB, groupID uint) ([]string, error) {
	var members []models.UserDeviceGroupMember
	if err := tx.Where("group_id = ?", groupID).Order("id asc").Find(&members).Error; err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.DeviceID)
	}
	return ids, nil
}

func normalizeDeviceIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
