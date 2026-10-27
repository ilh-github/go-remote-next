package handler

import (
	"net/http"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// OwnerAuditHandler serves GET /v1/me/audit-logs: the owner's read side of the
// data-channel audit trail.
//
// The write endpoints (file-transfer-audits, clipboard-audits) already refuse a
// device the caller does not own, so the actor on every row they produce is that
// device's owner. That makes the actor id a safe scope for the read: this handler
// narrows to the caller's own rows and nothing else. Without it the owner could
// report a transfer but not read it back anywhere except the admin console, which
// leaves the one person who can act on the answer unable to ask the question.
//
// The action filter is an allow-list rather than a free-form match, for the same
// reason the admin export refuses an unknown format: answering "you did nothing"
// to a typo closes an investigation on the wrong evidence.
type OwnerAuditHandler struct {
	audit *service.AuditService
}

func NewOwnerAuditHandler(audit *service.AuditService) *OwnerAuditHandler {
	return &OwnerAuditHandler{audit: audit}
}

// ownerAuditActions is the set of data-channel audit actions this read covers.
//
// Both are the payloads that travel peer to peer on the Remoting data channel,
// which is why they are reported by the controller and why they live apart from a
// generic log. An action outside this set is refused rather than treated as "no
// filter": the caller asked about something this trail does not hold.
var ownerAuditActions = map[string]struct{}{
	// One row per transfer, whichever way it went. The direction used to be part
	// of the action, which made a transfer id reported with both directions
	// produce two rows; it lives in the row's details now.
	"file.transfer":  {},
	"clipboard.sync": {},
	// The direction-scoped names stay accepted, because "what did I upload" is a
	// real question - they are just no longer row identities.
	"file.transfer.upload":    {},
	"file.transfer.download":  {},
	"clipboard.sync.upload":   {},
	"clipboard.sync.download": {},
}

// directionActions maps a direction-scoped filter onto the aggregate action plus
// the direction the row must have recorded in its details.
var directionActions = map[string]struct {
	action    string
	direction string
}{
	"file.transfer.upload":    {action: "file.transfer", direction: "upload"},
	"file.transfer.download":  {action: "file.transfer", direction: "download"},
	"clipboard.sync.upload":   {action: "clipboard.sync", direction: "upload"},
	"clipboard.sync.download": {action: "clipboard.sync", direction: "download"},
}

func (h *OwnerAuditHandler) List(c *gin.Context) {
	if h.audit == nil {
		ProblemInternal(c, "audit store is not configured")
		return
	}
	uid := middleware.MustUserID(c)
	if uid == 0 {
		ProblemUnauthorized(c, ProblemCodeUnauthorized, "authentication required")
		return
	}
	// ParseCursor already clamps limit to the generic v1 cap, so a busy account
	// gets a bounded page rather than the whole table; no second ceiling here.
	p := ParseCursor(c)
	cur, err := DecodeCursor(p.Cursor)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	action := strings.TrimSpace(c.Query("action"))
	if action != "" {
		if _, ok := ownerAuditActions[action]; !ok {
			ProblemBadRequest(c, ProblemCodeInvalidRequest,
				"action must be one of file.transfer, clipboard.sync, file.transfer.upload, file.transfer.download, clipboard.sync.upload, clipboard.sync.download")
			return
		}
	}
	// A direction-scoped filter resolves to the aggregate action plus that
	// direction, which is matched against the details JSON.
	wantDirection := ""
	if mapped, ok := directionActions[action]; ok {
		action = mapped.action
		wantDirection = mapped.direction
	}

	// The actor id is the scope. Every row this endpoint can return was written
	// by a caller who had already proved they own the device, so "what I did" and
	// "what happened on my machines" are the same set here - and nobody else's
	// rows can enter it.
	logs, total, err := h.audit.List(c.Request.Context(), service.AuditListParams{
		AfterID:     cur.OffsetID,
		Limit:       p.Limit + 1,
		Action:      action,
		ActorUserID: uid,
		// Quoted so a direction cannot match a filename or an error message that
		// happens to contain the word.
		DetailsContains: directionDetailsFilter(wantDirection),
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

// directionDetailsFilter builds the details filter for a direction-scoped audit
// query. Empty when no direction was asked for, so the filter is not applied at
// all rather than matching everything.
func directionDetailsFilter(direction string) string {
	if direction == "" {
		return ""
	}
	// The quotes are part of the match: `"upload"` as it appears in the details
	// JSON array, not the bare word.
	return `"` + direction + `"`
}
