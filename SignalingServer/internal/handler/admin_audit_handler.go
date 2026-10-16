package handler

import (
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
	audit         *service.AuditService
	retentionDays int
}

func NewAdminAuditHandler(a *service.AuditService, retentionDays int) *AdminAuditHandler {
	return &AdminAuditHandler{audit: a, retentionDays: retentionDays}
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

// auditSweepReq lets an operator run one purge with a different window than the
// configured retention, which is how retention is exercised without changing
// config and restarting.
type auditSweepReq struct {
	OlderThanDays *int `json:"older_than_days"`
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
	if days <= 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest,
			"retention is not configured; set AUDIT_RETENTION_DAYS or pass older_than_days")
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	var purged int64
	for {
		removed, err := h.audit.PurgeBefore(c.Request.Context(), cutoff)
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
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "audit.sweep",
			"audit_log", "",
			fmt.Sprintf("older_than_days=%d cutoff=%s purged=%d archive=%s",
				days, cutoff.Format(time.RFC3339), purged, h.audit.ArchiveDir()),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"purged":          purged,
		"older_than_days": days,
		"cutoff":          cutoff.Format(time.RFC3339),
		"archive_dir":     h.audit.ArchiveDir(),
	})
}
