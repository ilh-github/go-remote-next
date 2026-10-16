package service

import (
	"context"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Log(ctx context.Context, adminID uint, adminUsername, action, resourceType, resourceID, details, ip string) {
	entry := &models.AuditLog{
		AdminID:       adminID,
		AdminUsername: adminUsername,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Details:       details,
		IP:            ip,
		CreatedAt:     time.Now(),
	}
	s.db.WithContext(ctx).Create(entry)
}

// LogActor records an audit row for an action taken by an end user rather than
// an administrator: the actor lands in actor_user_id, and admin_id stays zero
// so an "admin activity" query does not pick it up.
//
// This exists because the interesting user-side actions are destructive ones -
// deleting a recording, changing who may reach a device - and until now they
// reached the audit trail either not at all or with no way to say who did it.
//
// ip is the actor's observed client address, or "" when the caller has none.
// It is separate from the admin path's ip only because the actor arrives
// through the event bus; the column is the same one, so a trail that mixes
// admin and user actions answers "from where" uniformly.
func (s *AuditService) LogActor(ctx context.Context, actorUserID uint, action, resourceType, resourceID, details, ip string) {
	entry := &models.AuditLog{
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IP:           ip,
		CreatedAt:    time.Now(),
	}
	s.db.WithContext(ctx).Create(entry)
}

// AuditListParams is the cursor-keyset filter used by GET /v1/admin/audit-logs.
type AuditListParams struct {
	AfterID               uint
	Limit                 int
	Action, AdminUsername string
	DateFrom, DateTo      string
	// ActorUserID narrows to one end-user actor. Zero means "no filter", which
	// is what the admin-activity view wants; user-side rows carry actor ids and
	// leave admin_id zero, so this is the only way to ask "what did user N do".
	ActorUserID uint
}

func (s *AuditService) List(ctx context.Context, p AuditListParams) ([]models.AuditLog, int64, error) {
	query := s.db.WithContext(ctx).Model(&models.AuditLog{})

	if p.Action != "" {
		query = query.Where("action = ?", p.Action)
	}
	if p.AdminUsername != "" {
		query = query.Where("admin_username LIKE ?", "%"+p.AdminUsername+"%")
	}
	if p.ActorUserID > 0 {
		query = query.Where("actor_user_id = ?", p.ActorUserID)
	}
	if p.DateFrom != "" {
		if t, err := time.Parse(time.RFC3339, p.DateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if p.DateTo != "" {
		if t, err := time.Parse(time.RFC3339, p.DateTo); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if p.AfterID > 0 {
		query = query.Where("id < ?", p.AfterID)
	}
	var logs []models.AuditLog
	err := query.Order("id DESC").Limit(p.Limit).Find(&logs).Error
	return logs, total, err
}
