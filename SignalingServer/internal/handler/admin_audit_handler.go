package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAuditHandler serves GET /v1/admin/audit-logs.
type AdminAuditHandler struct {
	audit                *service.AuditService
	retentionDays        int
	maxRows              int
	archiveRetentionDays int
}

func NewAdminAuditHandler(a *service.AuditService, retentionDays, maxRows, archiveRetentionDays int) *AdminAuditHandler {
	return &AdminAuditHandler{
		audit:                a,
		retentionDays:        retentionDays,
		maxRows:              maxRows,
		archiveRetentionDays: archiveRetentionDays,
	}
}

func (h *AdminAuditHandler) List(c *gin.Context) {
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	actorUserID, err := parseActorUserID(c.Query("actor_user_id"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	logs, total, err := h.audit.List(c.Request.Context(), service.AuditListParams{
		AfterID:       cur.OffsetID,
		Limit:         p.Limit + 1,
		Action:        c.Query("action"),
		AdminUsername: c.Query("admin"),
		ActorUserID:   actorUserID,
		DateFrom:      c.Query("date_from"),
		DateTo:        c.Query("date_to"),
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	next := ""
	if len(logs) > p.Limit {
		last := logs[p.Limit-1]
		next = EncodeCursor(CursorPayload{OffsetID: last.ID})
		logs = logs[:p.Limit]
	}
	c.JSON(http.StatusOK, NewCursorPage(logs, next, total))
}

// parseActorUserID reads the optional `actor_user_id` filter. A blank value
// means "no filter" (the admin-activity view); zero is also "no filter" because
// no audit row is attributed to user 0. Anything else that is not a positive
// integer is an error rather than a silent fallback: a filter that quietly
// matched everything would look like "this user did nothing wrong".
func parseActorUserID(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("actor_user_id must be a positive integer")
	}
	return uint(parsed), nil
}

// auditSweepReq lets an operator run one purge with a different window or row
// cap than the configured retention, which is how either bound is exercised
// without changing config and restarting.
type auditSweepReq struct {
	OlderThanDays        *int `json:"older_than_days"`
	MaxRows              *int `json:"max_rows"`
	ArchiveRetentionDays *int `json:"archive_retention_days"`
}

// Sweep handles POST /v1/admin/audit-logs:sweep.
//
// The periodic worker calls the same service method, so this endpoint exists so
// an operator can run the purge on demand and see what it did. It refuses to
// guess when no window is available: deleting audit rows is irreversible for
// anyone who did not configure an archive, and "retention is not set" must not
// silently mean "keep zero days".
func (h *AdminAuditHandler) Sweep(c *gin.Context) {
	if h.audit == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "audit store is not configured")
		return
	}
	var req auditSweepReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
	}
	days := h.retentionDays
	if req.OlderThanDays != nil {
		days = *req.OlderThanDays
	}
	maxRows := h.maxRows
	if req.MaxRows != nil {
		maxRows = *req.MaxRows
	}
	archiveDays := h.archiveRetentionDays
	if req.ArchiveRetentionDays != nil {
		archiveDays = *req.ArchiveRetentionDays
	}
	if days <= 0 && maxRows <= 0 && archiveDays <= 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"retention is not configured; set AUDIT_RETENTION_DAYS / AUDIT_MAX_ROWS / AUDIT_ARCHIVE_RETENTION_DAYS or pass older_than_days / max_rows / archive_retention_days")
		return
	}

	var purged int64
	cutoff := ""
	if days > 0 {
		cut := time.Now().UTC().AddDate(0, 0, -days)
		cutoff = cut.Format(time.RFC3339)
		for {
			removed, err := h.audit.PurgeBefore(c.Request.Context(), cut)
			if err != nil {
				// An archive failure lands here: nothing was deleted, so say so.
				ProblemInternal(c, "purge failed, nothing deleted: "+err.Error())
				return
			}
			purged += removed
			if removed == 0 {
				break
			}
		}
	}
	// The row cap is a second, age-independent bound; it runs after the age
	// purge so a row that both rules would drop is only counted once.
	var capRemoved int64
	if maxRows > 0 {
		for {
			removed, err := h.audit.CapRows(c.Request.Context(), maxRows)
			if err != nil {
				ProblemInternal(c, "row cap failed, nothing deleted: "+err.Error())
				return
			}
			capRemoved += removed
			if removed == 0 {
				break
			}
		}
	}
	// The archive files are a third, independent bound: rows may be kept
	// forever while the JSONL they were purged into is not.
	var archiveRemoved int
	if archiveDays > 0 {
		if h.audit.ArchiveDir() == "" {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"archive retention was requested but AUDIT_ARCHIVE_DIR is not configured")
			return
		}
		removed, err := h.audit.SweepArchive(archiveDays)
		if err != nil {
			ProblemInternal(c, "archive sweep failed: "+err.Error())
			return
		}
		archiveRemoved = removed
	}
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "audit.sweep",
			"audit_log", "",
			fmt.Sprintf("older_than_days=%d cutoff=%s purged=%d max_rows=%d cap_removed=%d archive_retention_days=%d archive_removed=%d archive=%s",
				days, cutoff, purged, maxRows, capRemoved, archiveDays, archiveRemoved, h.audit.ArchiveDir()),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"purged":                 purged,
		"older_than_days":        days,
		"cutoff":                 cutoff,
		"max_rows":               maxRows,
		"cap_removed":            capRemoved,
		"archive_retention_days": archiveDays,
		"archive_removed":        archiveRemoved,
		"archive_dir":            h.audit.ArchiveDir(),
	})
}

// parseAuditArchiveLimit reads the optional `limit` query. Zero means "use the
// default"; anything negative or non-numeric is a 400 rather than a silent
// fallback, because "I asked for everything" and "you returned 50" must not be
// confused. Values above the cap are clamped and echoed back.
func parseAuditArchiveLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be a number")
	}
	if parsed < 0 {
		return 0, fmt.Errorf("limit must be zero (default) or positive")
	}
	return parsed, nil
}

// Archive handles GET /v1/admin/audit-archive:search.
//
// Retention appends expired rows to a daily JSONL file before deleting them, so
// the live table stays bounded; this endpoint reads that file back so "what
// happened last week" survives the purge. It is a bounded reverse scan like
// `system/logs:search`, and it refuses to run at all when no archive directory
// is configured - an empty result must never mean "retention deleted it and you
// never turned on the archive".
func (h *AdminAuditHandler) Archive(c *gin.Context) {
	if h.audit == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "audit store is not configured")
		return
	}
	if h.audit.ArchiveDir() == "" {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"audit archive is not configured; set AUDIT_ARCHIVE_DIR")
		return
	}
	actorUserID, err := parseActorUserID(c.Query("actor_user_id"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	limit, err := parseAuditArchiveLimit(c.Query("limit"))
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	result, err := h.audit.QueryArchive(c.Request.Context(), service.AuditArchiveQuery{
		Date:          c.Query("date"),
		Query:         c.Query("q"),
		Action:        c.Query("action"),
		AdminUsername: c.Query("admin"),
		ActorUserID:   actorUserID,
		Limit:         limit,
	})
	if errors.Is(err, service.ErrAuditArchiveNotFound) {
		ProblemNotFound(c, ProblemCodeNotFound, fmt.Sprintf("no archive file for %s", result.Date))
		return
	}
	if errors.Is(err, service.ErrAuditArchiveBadDate) {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}

	h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "audit.archive.search",
		"audit_log", result.Date,
		fmt.Sprintf("date=%s q=%q action=%q admin=%q actor_user_id=%d limit=%d returned=%d truncated=%t",
			result.Date, c.Query("q"), c.Query("action"), c.Query("admin"), actorUserID, limit,
			len(result.Items), result.Truncated),
		c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"items":         result.Items,
		"count":         len(result.Items),
		"date":          result.Date,
		"file":          result.File,
		"scanned_bytes": result.ScannedBytes,
		"scanned_lines": result.ScannedLines,
		"truncated":     result.Truncated,
		"has_more":      result.HasMore,
		"archive_dir":   h.audit.ArchiveDir(),
	})
}
