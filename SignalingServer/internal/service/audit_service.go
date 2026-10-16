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

// CapRows trims the oldest audit rows until at most maxRows remain, and reports
// how many it removed. It is the second, age-independent bound on the table:
// retention answers "how long do we keep rows", the cap answers "how big is
// the table allowed to get". A caller loops until it returns 0 because each
// call removes at most auditPurgeBatch rows.
//
// When an archive directory is configured the rows are appended to the daily
// JSONL file first and only deleted once that write is fsynced, exactly like
// age retention - the cap must never be the step that destroys the only copy.
func (s *AuditService) CapRows(ctx context.Context, maxRows int) (int64, error) {
	if maxRows <= 0 {
		return 0, nil
	}
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return 0, err
	}
	overflow := int(total) - maxRows
	if overflow <= 0 {
		return 0, nil
	}
	if overflow > auditPurgeBatch {
		overflow = auditPurgeBatch
	}

	if s.archiveDir == "" {
		// Oldest `overflow` ids by id ASC. id tracks insertion order, so this
		// drops the oldest rows without needing a second ordering column.
		res := s.db.WithContext(ctx).Exec(
			"DELETE FROM audit_logs WHERE id IN (SELECT id FROM audit_logs ORDER BY id ASC LIMIT ?)",
			overflow)
		return res.RowsAffected, res.Error
	}

	var rows []models.AuditLog
	if err := s.db.WithContext(ctx).Order("id ASC").Limit(overflow).Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.archiveRows(rows); err != nil {
		return 0, fmt.Errorf("archive audit logs before row-cap purge: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	res := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.AuditLog{})
	return res.RowsAffected, res.Error
}

// auditCapScopesPerPass bounds how many distinct actors one per-actor cap pass
// looks at. The worker loops until a pass removes nothing, so a deployment with
// many over-limit actors drains over several passes instead of doing all of it
// inside one long statement.
const auditCapScopesPerPass = 20

// Actor kinds for the per-actor cap. The kind is part of the identity: an admin
// with id 7 and an end user with id 7 are different people, and merging them
// into one bucket would let an admin's activity spend a user's budget.
const (
	auditScopeAdmin = "admin"
	auditScopeUser  = "user"
)

// auditScope is one actor whose rows are being capped.
type auditScope struct {
	Kind string
	ID   uint
}

// predicate returns the SQL fragment (and its args) that selects exactly this
// actor's rows. A row never counts as both kinds: the admin columns win when
// they are set, matching how Log and LogActor write the two.
func (s auditScope) predicate() (string, []interface{}) {
	if s.Kind == auditScopeAdmin {
		return "admin_id = ?", []interface{}{s.ID}
	}
	return "admin_id = 0 AND actor_user_id = ?", []interface{}{s.ID}
}

// CapRowsPerActor trims the oldest rows of any single actor until each holds at
// most maxRowsPerActor rows, and reports how many rows it removed.
//
// The table-wide cap (CapRows) answers "how big may this table get"; it cannot
// stop one noisy actor - a user whose failing token refreshes log in a loop, an
// admin running a bulk migration - from filling the table and evicting
// *everybody else's* trail. This is the bound that keeps the audit trail
// attributable: exceeding the per-actor limit costs the noisy actor their own
// old rows instead of everyone's.
//
// Rows with no actor at all (both admin_id and actor_user_id zero - the
// system's own entries) are deliberately exempt: there is nobody to charge
// them to, and the table-wide cap still bounds them.
//
// Like CapRows, one call removes at most auditPurgeBatch rows and inspects at
// most auditCapScopesPerPass actors, so the caller loops until it returns 0.
// When an archive directory is configured the rows are appended to the daily
// JSONL file first and only deleted once that write has been fsynced - the cap
// must never be the step that destroys the only copy of a trail.
func (s *AuditService) CapRowsPerActor(ctx context.Context, maxRowsPerActor int) (int64, error) {
	if maxRowsPerActor <= 0 {
		return 0, nil
	}
	scopes, err := s.overLimitAuditScopes(ctx, maxRowsPerActor)
	if err != nil {
		return 0, err
	}
	var removedTotal int64
	for _, scope := range scopes {
		if removedTotal >= auditPurgeBatch {
			break
		}
		removed, err := s.capOneAuditScope(ctx, scope, maxRowsPerActor, auditPurgeBatch-int(removedTotal))
		if err != nil {
			return removedTotal, err
		}
		removedTotal += removed
	}
	return removedTotal, nil
}

// overLimitAuditScopes lists the actors holding more than maxRowsPerActor rows,
// oldest actor id first so repeated passes walk the same deterministic order.
func (s *AuditService) overLimitAuditScopes(ctx context.Context, maxRowsPerActor int) ([]auditScope, error) {
	var rows []struct {
		ActorKind string
		ActorID   uint
	}
	err := s.db.WithContext(ctx).Raw(`
		SELECT actor_kind, actor_id FROM (
			SELECT CASE WHEN admin_id <> 0 THEN 'admin' ELSE 'user' END AS actor_kind,
			       CASE WHEN admin_id <> 0 THEN admin_id ELSE actor_user_id END AS actor_id
			FROM audit_logs
			WHERE admin_id <> 0 OR actor_user_id <> 0
		) AS attributed
		GROUP BY actor_kind, actor_id
		HAVING COUNT(*) > ?
		ORDER BY actor_kind, actor_id
		LIMIT ?`, maxRowsPerActor, auditCapScopesPerPass).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	scopes := make([]auditScope, 0, len(rows))
	for _, row := range rows {
		scopes = append(scopes, auditScope{Kind: row.ActorKind, ID: row.ActorID})
	}
	return scopes, nil
}

// capOneAuditScope removes this actor's oldest rows down to the cap, by at most
// `budget` rows.
func (s *AuditService) capOneAuditScope(ctx context.Context, scope auditScope, maxRowsPerActor, budget int) (int64, error) {
	where, args := scope.predicate()
	if budget <= 0 {
		return 0, nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where(where, args...).Count(&count).Error; err != nil {
		return 0, err
	}
	overflow := int(count) - maxRowsPerActor
	if overflow <= 0 {
		return 0, nil
	}
	if overflow > budget {
		overflow = budget
	}

	if s.archiveDir == "" {
		deleteArgs := append(append([]interface{}{}, args...), overflow)
		res := s.db.WithContext(ctx).Exec(
			"DELETE FROM audit_logs WHERE id IN (SELECT id FROM audit_logs WHERE "+where+" ORDER BY id ASC LIMIT ?)",
			deleteArgs...)
		return res.RowsAffected, res.Error
	}

	var rows []models.AuditLog
	if err := s.db.WithContext(ctx).
		Where(where, args...).
		Order("id ASC").
		Limit(overflow).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.archiveRows(rows); err != nil {
		return 0, fmt.Errorf("archive audit logs before per-actor purge: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	// Re-apply the scope in the delete so a row cannot be removed by the archive
	// path if its actor changed between the read and the delete.
	res := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Where(where, args...).
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

// SweepArchive deletes daily archive files older than archiveRetentionDays and
// reports how many it removed. 0 means "keep the archives forever"; a
// deployment with no archive directory simply has nothing to sweep.
func (s *AuditService) SweepArchive(archiveRetentionDays int) (int, error) {
	if s == nil {
		return 0, nil
	}
	return sweepArchiveDir(s.archiveDir, "audit-logs", archiveRetentionDays, time.Now().UTC())
}

// RunRetention sweeps expired rows on a ticker until ctx is cancelled.
//
// Four independent bounds are enforced, and any one alone is enough to make the
// sweep run: `retentionDays` deletes rows older than the window, `maxRows`
// trims the oldest rows so at most that many remain whatever their age,
// `maxRowsPerActor` trims the oldest rows of any one actor, and
// `archiveRetentionDays` deletes archive files older than its window. Keeping
// them separate matters - a deployment may want "keep 30 days OR 100k rows,
// whichever bites first" and must be able to set any one to 0 to disable that
// bound without disabling the others.
//
// The per-actor cap runs *before* the table-wide one on purpose: it is the
// targeted rule (charge the noisy actor their own oldest rows), so letting it
// fire first leaves the blunt rule less to do and less chance of evicting an
// innocent actor's trail instead.
//
// interval <= 0 with every bound <= 0 means "keep every audit row (and every
// archive file) forever", matching the recording and notification defaults.
func (s *AuditService) RunRetention(ctx context.Context, interval time.Duration, retentionDays, maxRows, maxRowsPerActor, archiveRetentionDays int) {
	if interval <= 0 || (retentionDays <= 0 && maxRows <= 0 && maxRowsPerActor <= 0 && archiveRetentionDays <= 0) {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if retentionDays > 0 {
				cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
				total := int64(0)
				// Archive-then-delete only handles one batch per call, so loop
				// until a pass removes nothing. The plain-delete path could do
				// it in one statement, but one loop keeps both paths identical.
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
			if maxRowsPerActor > 0 {
				total := int64(0)
				for {
					removed, err := s.CapRowsPerActor(ctx, maxRowsPerActor)
					if err != nil {
						log.Printf("[audit] per-actor row-cap sweep failed: %v", err)
						break
					}
					total += removed
					if removed == 0 {
						break
					}
				}
				if total > 0 {
					log.Printf("[audit] per-actor row cap removed %d audit rows over %d", total, maxRowsPerActor)
				}
			}
			if maxRows > 0 {
				total := int64(0)
				for {
					removed, err := s.CapRows(ctx, maxRows)
					if err != nil {
						log.Printf("[audit] row-cap sweep failed: %v", err)
						break
					}
					total += removed
					if removed == 0 {
						break
					}
				}
				if total > 0 {
					log.Printf("[audit] row cap removed %d audit rows over %d", total, maxRows)
				}
			}
			if archiveRetentionDays > 0 {
				removed, err := s.SweepArchive(archiveRetentionDays)
				if err != nil {
					log.Printf("[audit] archive sweep failed: %v", err)
				} else if removed > 0 {
					log.Printf("[audit] archive sweep removed %d file(s) older than %d day(s)",
						removed, archiveRetentionDays)
				}
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
