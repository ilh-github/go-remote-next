package handler

import (
	"fmt"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminNotificationDeliveryHandler exposes the user-notification outbox to
// administrators:
//
//	GET  /v1/admin/notification-deliveries:stats
//	POST /v1/admin/notification-deliveries:sweep
//
// The periodic retention worker calls the same service method as `:sweep`, so
// an operator can run the purge on demand (and see what it did) instead of
// waiting for the next tick.
type AdminNotificationDeliveryHandler struct {
	deliveries    *service.NotificationDeliveryService
	audit         *service.AuditService
	retentionDays int
}

func NewAdminNotificationDeliveryHandler(
	deliveries *service.NotificationDeliveryService,
	audit *service.AuditService,
	retentionDays int,
) *AdminNotificationDeliveryHandler {
	return &AdminNotificationDeliveryHandler{
		deliveries:    deliveries,
		audit:         audit,
		retentionDays: retentionDays,
	}
}

// Stats reports how many outbox rows sit in each delivery state.
func (h *AdminNotificationDeliveryHandler) Stats(c *gin.Context) {
	if h.deliveries == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "delivery store is not configured")
		return
	}
	counts, err := h.deliveries.CountByStatus(c.Request.Context())
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	var total int64
	for _, n := range counts {
		total += n
	}
	c.JSON(http.StatusOK, gin.H{
		"total":          total,
		"by_status":      counts,
		"retention_days": h.retentionDays,
		"archive_dir":    h.deliveries.ArchiveDir(),
	})
}

// notificationSweepReq lets an operator run one purge with a different window
// than the configured retention (handy for testing without changing config).
type notificationSweepReq struct {
	OlderThanDays *int `json:"older_than_days"`
}

// Sweep runs the retention pass on demand.
func (h *AdminNotificationDeliveryHandler) Sweep(c *gin.Context) {
	if h.deliveries == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "delivery store is not configured")
		return
	}
	var req notificationSweepReq
	// An empty body means "use the configured retention"; only parse when the
	// caller actually sent something.
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
			"retention is not configured; set NOTIFICATION_RETENTION_DAYS or pass older_than_days")
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	purged, err := h.deliveries.PurgeTerminal(c.Request.Context(), cutoff)
	if err != nil {
		// An archive failure lands here: nothing was deleted, so say so.
		ProblemInternal(c, "purge failed, nothing deleted: "+err.Error())
		return
	}

	archiveDir := h.deliveries.ArchiveDir()
	if h.audit != nil {
		h.audit.Log(c.Request.Context(), middleware.MustAdminID(c), "", "notification_delivery.sweep",
			"notification_delivery", "",
			fmt.Sprintf("older_than_days=%d cutoff=%s purged=%d archive=%s",
				days, cutoff.Format(time.RFC3339), purged, archiveDir),
			c.ClientIP())
	}
	c.JSON(http.StatusOK, gin.H{
		"purged":          purged,
		"older_than_days": days,
		"cutoff":          cutoff.Format(time.RFC3339),
		"archive_dir":     archiveDir,
	})
}
