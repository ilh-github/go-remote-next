package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
	// archiveDir, when set, makes retention append expired rows to a daily
	// JSONL file before deleting them. Audit rows are evidence, so this is the
	// recommended posture once retention is switched on. Empty = plain delete.
	archiveDir string
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// WithArchiveDir enables archive-before-delete for retention. Returns the
// receiver so it can be chained onto the constructor.
func (s *AuditService) WithArchiveDir(dir string) *AuditService {
	s.archiveDir = strings.TrimSpace(dir)
	return s
}

// ArchiveDir reports the configured archive directory ("" = plain delete).
func (s *AuditService) ArchiveDir() string {
	if s == nil {
		return ""
	}
	return s.archiveDir
}

// auditPurgeBatch bounds how many rows one archive-then-delete pass reads into
// memory. The periodic worker repeats until a pass comes back empty.
const auditPurgeBatch = 1000

// PurgeBefore deletes audit rows created before cutoff and reports how many
// went. When an archive directory is configured the rows are appended to a
// daily JSONL file first and only deleted once that write has been fsynced, so
// retention can never be the step that silently destroys the only copy of a
// trail.
func (s *AuditService) PurgeBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoff = cutoff.UTC()
	if s.archiveDir == "" {
		res := s.db.WithContext(ctx).
			Where("created_at < ?", cutoff).
			Delete(&models.AuditLog{})
		return res.RowsAffected, res.Error
	}

	var rows []models.AuditLog
	if err := s.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Order("id").
		Limit(auditPurgeBatch).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.archiveRows(rows); err != nil {
		return 0, fmt.Errorf("archive audit logs before purge: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	// Re-apply the cutoff in the delete so a row cannot be removed by the
	// archive path if it was somehow re-dated between the read and the delete.
	res := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where("created_at < ?", cutoff).
		Delete(&models.AuditLog{})
	return res.RowsAffected, res.Error
}

// archiveRows appends one JSON object per row to the day's archive file and
// fsyncs it, so the bytes are durable before the caller deletes them.
func (s *AuditService) archiveRows(rows []models.AuditLog) error {
	if err := os.MkdirAll(s.archiveDir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("audit-logs-%s.jsonl", time.Now().UTC().Format("2006-01-02"))
	path := filepath.Join(s.archiveDir, name)

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(buf.Bytes()); err != nil {
		return err
	}
	return file.Sync()
}

// RunRetention sweeps expired rows on a ticker until ctx is cancelled.
// retentionDays <= 0 or interval <= 0 keeps history forever, matching the
// recording and notification retention settings, so the default is
// "never delete an audit row".
func (s *AuditService) RunRetention(ctx context.Context, interval time.Duration, retentionDays int) {
	if retentionDays <= 0 || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			total := int64(0)
			// Archive-then-delete only handles one batch per call, so loop
			// until a pass removes nothing. The plain-delete path could do it
			// in one statement, but one loop keeps both paths identical.
			for {
				removed, err := s.PurgeBefore(ctx, cutoff)
				if err != nil {
					log.Printf("[audit] retention sweep failed: %v", err)
					break
				}
				total += removed
				if removed == 0 {
					break
				}
			}
			if total > 0 {
				log.Printf("[audit] retention purged %d audit rows older than %s",
					total, cutoff.Format(time.RFC3339))
			}
		}
	}
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

// LogActorDeduped writes a user-side audit row only when the same actor has not
// already produced an identical (action, resource) row within window, and
// reports whether it wrote. It exists for media reads: a browser fetches a
// recording in many range requests and re-renders thumbnails as the list
// scrolls, so one row per HTTP request would bury the trail it is supposed to
// be. A window <= 0 means "always write".
//
// The check is deliberately a plain read-then-insert rather than a lock: the
// worst case for two concurrent reads is a duplicate row, which is exactly what
// the caller would have written without dedupe, so there is nothing to protect.
// A failed lookup is also treated as "no recent row", so a database hiccup can
// only produce more rows, never lose one.
func (s *AuditService) LogActorDeduped(
	ctx context.Context,
	actorUserID uint,
	action, resourceType, resourceID, details, ip string,
	window time.Duration,
) bool {
	if window <= 0 {
		s.LogActor(ctx, actorUserID, action, resourceType, resourceID, details, ip)
		return true
	}
	var recent int64
	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("actor_user_id = ? AND action = ? AND resource_type = ? AND resource_id = ?",
			actorUserID, action, resourceType, resourceID).
		Where("created_at >= ?", time.Now().Add(-window)).
		Count(&recent).Error; err == nil && recent > 0 {
		return false
	}
	s.LogActor(ctx, actorUserID, action, resourceType, resourceID, details, ip)
	return true
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
