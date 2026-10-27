package service

import (
	"context"
	"errors"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ConnectionService implements /v1/me/connections — a per-user activity
// log capturing every connection attempt (success / failed / timeout).
type ConnectionService struct {
	db *gorm.DB
}

func NewConnectionService(db *gorm.DB) *ConnectionService {
	return &ConnectionService{db: db}
}

// RecordInput is the validated body of POST /v1/me/connections.
type RecordInput struct {
	DeviceID   string
	DeviceName string
	ConnectIP  string
	Duration   int
	Status     string
	ErrorMsg   string
}

var ErrInvalidConnectionStatus = errors.New("status must be success|failed|timeout")

// ErrConnectionHistoryNotFound is what a delete of somebody else's row returns.
// The row id alone is not authority: without this, one account could remove
// another account's history entry by guessing an id, and the caller could not
// tell "not yours" from "already gone".
var ErrConnectionHistoryNotFound = errors.New("connection history entry not found")

func (s *ConnectionService) Record(ctx context.Context, userID uint, in RecordInput) (*models.ConnectionHistory, error) {
	switch in.Status {
	case "success", "failed", "timeout":
	default:
		return nil, ErrInvalidConnectionStatus
	}
	if in.DeviceID == "" {
		return nil, errors.New("device_id is required")
	}
	row := &models.ConnectionHistory{
		UserID:     userID,
		DeviceID:   in.DeviceID,
		DeviceName: in.DeviceName,
		ConnectIP:  in.ConnectIP,
		Duration:   in.Duration,
		Status:     in.Status,
		ErrorMsg:   in.ErrorMsg,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	// If success, bump user_devices.last_connect_at / connect_count.
	if in.Status == "success" {
		now := time.Now().UTC()
		s.db.WithContext(ctx).Model(&models.UserDevice{}).
			Where("user_id = ? AND device_id = ?", userID, in.DeviceID).
			Updates(map[string]interface{}{
				"last_connect_at": now,
				"connect_count":   gorm.Expr("connect_count + 1"),
			})
	}
	return row, nil
}

// ListParams carries the cursor params used by GET /v1/me/connections.
type ConnectionListParams struct {
	Since  time.Time // inclusive lower bound; zero = no bound
	Before time.Time // exclusive upper bound (cursor); zero = now
	Limit  int
}

// List returns the user's connection history newest-first, paged by
// `created_at < Before`.
func (s *ConnectionService) List(ctx context.Context, userID uint, p ConnectionListParams) ([]models.ConnectionHistory, error) {
	q := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")
	if !p.Since.IsZero() {
		q = q.Where("created_at >= ?", p.Since)
	}
	if !p.Before.IsZero() {
		q = q.Where("created_at < ?", p.Before)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	var out []models.ConnectionHistory
	if err := q.Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes one entry, scoped to its owner.
//
// The history grows a row on every connection attempt, so without this the list
// only accumulates -- upstream issue #18 asks for "delete single entry" for
// exactly that reason. The owner is part of the WHERE clause rather than a
// check afterwards: a delete that fetched by id first and then compared owners
// would still remove the row in the window between the two statements.
func (s *ConnectionService) Delete(ctx context.Context, userID uint, id uint) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.ConnectionHistory{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConnectionHistoryNotFound
	}
	return nil
}

// Clear removes the owner's whole history and reports how many rows went.
//
// The count is what the UI confirms with, so it is the real number rather than a
// generic "cleared". Scoped by user id: a clear-all that matched on nothing else
// would empty every account's history.
func (s *ConnectionService) Clear(ctx context.Context, userID uint) (int64, error) {
	res := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.ConnectionHistory{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
