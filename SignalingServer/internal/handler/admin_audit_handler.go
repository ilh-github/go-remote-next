package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAuditHandler serves GET /v1/admin/audit-logs.
type AdminAuditHandler struct {
	audit *service.AuditService
}

func NewAdminAuditHandler(a *service.AuditService) *AdminAuditHandler {
	return &AdminAuditHandler{audit: a}
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
