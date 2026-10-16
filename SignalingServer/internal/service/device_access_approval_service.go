package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// DeviceAccessApprovalDefaultTTL is how long the owner has to decide before a
// pending request expires. Short on purpose: the controller is sitting in
// front of a countdown.
const DeviceAccessApprovalDefaultTTL = 2 * time.Minute

// ClaimOutcome is the state a claim attempt observed. It mirrors the approval
// status plus the two transport-level cases (not found / already consumed) so
// the handler can map straight onto HTTP codes.
type ClaimOutcome int

const (
	// ClaimConfirmed means the approval had been granted and this call just
	// consumed it: the caller may mint a signal token.
	ClaimConfirmed ClaimOutcome = iota
	// ClaimPending means the owner has not decided yet.
	ClaimPending
	// ClaimDenied means the owner refused.
	ClaimDenied
	// ClaimExpired means no decision arrived in time.
	ClaimExpired
	// ClaimConsumed means the approval was already exchanged for a token.
	ClaimConsumed
	// ClaimNotFound means the id/secret pair does not match an approval. It is
	// deliberately indistinguishable from a wrong secret so the endpoint is
	// not an id oracle.
	ClaimNotFound
)

// DeviceAccessApprovalService issues and resolves second-factor approvals.
type DeviceAccessApprovalService struct {
	db *gorm.DB
}

func NewDeviceAccessApprovalService(db *gorm.DB) *DeviceAccessApprovalService {
	return &DeviceAccessApprovalService{db: db}
}

// CreateDeviceAccessApprovalInput is one pending request.
type CreateDeviceAccessApprovalInput struct {
	DeviceID string
	OwnerID  uint
	ClientID string
	ClientIP string
	Source   string
	GrantID  *uint
	TTL      time.Duration
}

// Create stores a pending approval and returns it together with the one-time
// plaintext secret the caller must hand back on claim.
func (s *DeviceAccessApprovalService) Create(
	ctx context.Context,
	in CreateDeviceAccessApprovalInput,
) (*models.DeviceAccessApproval, string, error) {
	if strings.TrimSpace(in.DeviceID) == "" {
		return nil, "", errors.New("device_id is required")
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = models.ApprovalSourceDeviceCode
	}
	if source != models.ApprovalSourceDeviceCode && source != models.ApprovalSourceGrant {
		return nil, "", fmt.Errorf("invalid approval source %q", source)
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = DeviceAccessApprovalDefaultTTL
	}
	secret, err := generateApprovalSecret()
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	row := &models.DeviceAccessApproval{
		DeviceID:   in.DeviceID,
		OwnerID:    in.OwnerID,
		ClientID:   in.ClientID,
		ClientIP:   in.ClientIP,
		Source:     source,
		GrantID:    in.GrantID,
		SecretHash: HashApprovalSecret(secret),
		Status:     models.ApprovalPending,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, "", err
	}
	return row, secret, nil
}

// Get returns one approval by id, or (nil, nil) when it does not exist.
func (s *DeviceAccessApprovalService) Get(ctx context.Context, approvalID uint) (*models.DeviceAccessApproval, error) {
	var row models.DeviceAccessApproval
	err := s.db.WithContext(ctx).Where("id = ?", approvalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// List returns a device's approvals newest-first. A non-empty statuses filter
// narrows to those statuses; overdue pending rows are expired first so a
// "pending" list never shows something that already timed out. status filter
// `expired` therefore also matches rows that were pending a moment ago.
func (s *DeviceAccessApprovalService) List(
	ctx context.Context,
	deviceID string,
	statuses []string,
	limit int,
) ([]models.DeviceAccessApproval, error) {
	if err := s.ExpireOverdue(ctx, deviceID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultApprovalListLimit
	}
	if limit > maxApprovalListLimit {
		limit = maxApprovalListLimit
	}
	q := s.db.WithContext(ctx).Model(&models.DeviceAccessApproval{}).Where("device_id = ?", deviceID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	var rows []models.DeviceAccessApproval
	err := q.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ExpireOverdue flips this device's unanswered, past-deadline rows to
// `expired`. It is safe to call repeatedly and is the only way a pending row
// changes without an owner action.
func (s *DeviceAccessApprovalService) ExpireOverdue(ctx context.Context, deviceID string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("device_id = ? AND status = ? AND expires_at <= ?", deviceID, models.ApprovalPending, now).
		Updates(map[string]interface{}{
			"status":     models.ApprovalExpired,
			"decided_at": now,
			"reason":     "expired without a decision",
			"updated_at": now,
		}).Error
}

// Decide records the owner's verdict. It returns the updated row and whether
// this call changed anything: deciding an already-decided row is a no-op, so a
// double-click cannot flip an approval.
func (s *DeviceAccessApprovalService) Decide(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	approve bool,
	decidedBy uint,
	reason string,
) (*models.DeviceAccessApproval, bool, error) {
	now := time.Now().UTC()
	status := models.ApprovalDenied
	if approve {
		status = models.ApprovalApproved
	}
	if len(reason) > 255 {
		reason = reason[:255]
	}
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("id = ? AND device_id = ? AND status = ? AND expires_at > ?",
			approvalID, deviceID, models.ApprovalPending, now).
		Updates(map[string]interface{}{
			"status":     status,
			"decided_by": decidedBy,
			"decided_at": now,
			"reason":     reason,
			"updated_at": now,
		})
	if result.Error != nil {
		return nil, false, result.Error
	}
	row, err := s.Get(ctx, approvalID)
	if err != nil {
		return nil, false, err
	}
	if row == nil || row.DeviceID != deviceID {
		return nil, false, nil
	}
	return row, result.RowsAffected == 1, nil
}

// Peek verifies the one-time secret and reports the current state without
// consuming anything. The handler uses it to check that the host is still
// online before it burns the approval.
//
// A missing id and a wrong secret both return ClaimNotFound on purpose: the
// endpoint must not become an id oracle.
func (s *DeviceAccessApprovalService) Peek(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	secret string,
) (*models.DeviceAccessApproval, ClaimOutcome, error) {
	row, err := s.Get(ctx, approvalID)
	if err != nil {
		return nil, ClaimNotFound, err
	}
	if row == nil || row.DeviceID != deviceID || !ApprovalSecretMatches(row.SecretHash, secret) {
		return nil, ClaimNotFound, nil
	}
	now := time.Now().UTC()
	switch row.EffectiveStatusAt(now) {
	case models.ApprovalPending:
		return row, ClaimPending, nil
	case models.ApprovalDenied:
		return row, ClaimDenied, nil
	case models.ApprovalExpired:
		// Best-effort persistence; the caller already has the answer.
		_ = s.db.WithContext(ctx).
			Model(&models.DeviceAccessApproval{}).
			Where("id = ? AND status = ?", row.ID, models.ApprovalPending).
			Updates(map[string]interface{}{
				"status": models.ApprovalExpired, "decided_at": now,
				"updated_at": now, "reason": "expired without a decision",
			}).Error
		return row, ClaimExpired, nil
	case models.ApprovalConsumed:
		return row, ClaimConsumed, nil
	case models.ApprovalApproved:
		return row, ClaimConfirmed, nil
	default:
		return row, ClaimDenied, nil
	}
}

// Claim verifies the one-time secret and, when the approval was granted,
// performs the `approved -> consumed` transition in the UPDATE itself, so two
// controllers racing the same approval cannot both mint a signal token:
// exactly one sees ClaimConfirmed, the other sees ClaimConsumed.
func (s *DeviceAccessApprovalService) Claim(
	ctx context.Context,
	deviceID string,
	approvalID uint,
	secret string,
) (*models.DeviceAccessApproval, ClaimOutcome, error) {
	row, outcome, err := s.Peek(ctx, deviceID, approvalID, secret)
	if err != nil || outcome != ClaimConfirmed {
		return row, outcome, err
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).
		Model(&models.DeviceAccessApproval{}).
		Where("id = ? AND status = ? AND consumed_at IS NULL", row.ID, models.ApprovalApproved).
		Updates(map[string]interface{}{
			"status":      models.ApprovalConsumed,
			"consumed_at": now,
			"updated_at":  now,
		})
	if result.Error != nil {
		return nil, ClaimNotFound, result.Error
	}
	if result.RowsAffected != 1 {
		return row, ClaimConsumed, nil
	}
	row.Status = models.ApprovalConsumed
	row.ConsumedAt = &now
	return row, ClaimConfirmed, nil
}

const (
	defaultApprovalListLimit = 50
	maxApprovalListLimit     = 200
)

// generateApprovalSecret returns 32 bytes of hex (64 chars). It is the bearer
// proof a controller presents to poll and claim its own approval.
func generateApprovalSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate approval secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// HashApprovalSecret is the stored form of an approval secret.
func HashApprovalSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// ApprovalSecretMatches compares the presented secret against a stored hash in
// constant time.
func ApprovalSecretMatches(storedHash, presented string) bool {
	if storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(HashApprovalSecret(presented))) == 1
}
