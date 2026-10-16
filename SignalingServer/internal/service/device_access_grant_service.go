package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidAccessGrant means the requested grant is malformed or would never
// admit anyone (non-positive lifetime, negative use cap).
var ErrInvalidAccessGrant = errors.New("access grants need expires_in_minutes > 0 and max_uses >= 0")

// grantCodeAlphabet avoids characters that are easy to confuse when someone
// reads a code aloud or retypes it (0/O, 1/I/L).
const grantCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// DeviceAccessGrantService issues and redeems time-boxed access grants.
type DeviceAccessGrantService struct {
	db *gorm.DB
}

func NewDeviceAccessGrantService(db *gorm.DB) *DeviceAccessGrantService {
	return &DeviceAccessGrantService{db: db}
}

// CreateDeviceAccessGrantInput is the validated body of the create endpoint.
type CreateDeviceAccessGrantInput struct {
	Label            string
	ExpiresInMinutes int
	MaxUses          int
	Code             string
}

// Create issues a grant for one device.
func (s *DeviceAccessGrantService) Create(ctx context.Context, deviceID string, ownerID uint, in CreateDeviceAccessGrantInput) (*models.DeviceAccessGrant, error) {
	if in.ExpiresInMinutes <= 0 || in.MaxUses < 0 {
		return nil, ErrInvalidAccessGrant
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		generated, err := generateGrantCode()
		if err != nil {
			return nil, err
		}
		code = generated
	}
	row := &models.DeviceAccessGrant{
		DeviceID:  deviceID,
		OwnerID:   ownerID,
		Code:      code,
		Label:     strings.TrimSpace(in.Label),
		ExpiresAt: time.Now().UTC().Add(time.Duration(in.ExpiresInMinutes) * time.Minute),
		MaxUses:   in.MaxUses,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List returns every grant for a device, newest first.
func (s *DeviceAccessGrantService) List(ctx context.Context, deviceID string) ([]models.DeviceAccessGrant, error) {
	var rows []models.DeviceAccessGrant
	err := s.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}

// Get returns one grant scoped to its device, or (nil, nil) when it is not
// there.
func (s *DeviceAccessGrantService) Get(ctx context.Context, deviceID string, grantID uint) (*models.DeviceAccessGrant, error) {
	var row models.DeviceAccessGrant
	err := s.db.WithContext(ctx).
		Where("id = ? AND device_id = ?", grantID, deviceID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Revoke cancels a grant. Revoking an already-revoked grant is a no-op.
func (s *DeviceAccessGrantService) Revoke(ctx context.Context, deviceID string, grantID uint) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&models.DeviceAccessGrant{}).
		Where("id = ? AND device_id = ? AND revoked_at IS NULL", grantID, deviceID).
		Updates(map[string]interface{}{"revoked_at": now, "updated_at": now}).Error
}

// Match returns the usable grant whose code equals the presented one, or nil
// when none does.
//
// The candidates are loaded for the device and compared with
// subtle.ConstantTimeCompare rather than looked up by code, so the redeem path
// keeps the same constant-time habit as the device access code (and does not
// turn the code index into an oracle).
func (s *DeviceAccessGrantService) Match(ctx context.Context, deviceID, code string) (*models.DeviceAccessGrant, error) {
	if strings.TrimSpace(code) == "" {
		return nil, nil
	}
	var rows []models.DeviceAccessGrant
	if err := s.db.WithContext(ctx).
		Where("device_id = ? AND revoked_at IS NULL AND expires_at > ?", deviceID, time.Now().UTC()).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	presented := []byte(code)
	for i := range rows {
		if subtle.ConstantTimeCompare([]byte(rows[i].Code), presented) == 1 && rows[i].UsableAt(time.Now().UTC()) {
			return &rows[i], nil
		}
	}
	return nil, nil
}

// Consume records one started session against the grant. It re-checks the
// grant in the UPDATE itself, so two controllers racing the last use of a
// single-use grant cannot both win: the loser sees ok=false.
func (s *DeviceAccessGrantService) Consume(ctx context.Context, grantID uint) (bool, error) {
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessGrant{}).
		Where("id = ? AND revoked_at IS NULL AND expires_at > ? AND (max_uses = 0 OR uses < max_uses)",
			grantID, now).
		Updates(map[string]interface{}{
			"uses":         gorm.Expr("uses + 1"),
			"last_used_at": now,
			"updated_at":   now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func generateGrantCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate grant code: %w", err)
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = grantCodeAlphabet[int(b)%len(grantCodeAlphabet)]
	}
	return string(out), nil
}
