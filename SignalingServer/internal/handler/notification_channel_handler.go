package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationChannelHandler serves the per-user notification endpoints:
//
//	GET    /v1/me/notification-channels
//	POST   /v1/me/notification-channels
//	PATCH  /v1/me/notification-channels/:id
//	DELETE /v1/me/notification-channels/:id
//	POST   /v1/me/notification-channels/:id/test
//	GET    /v1/me/notification-channels/:id/deliveries
//	POST   /v1/me/notification-channels/:id/deliveries/:deliveryId/retry
//	POST   /v1/me/notification-channels/:id/deliveries:retry-failed
//
// Every query is scoped by user_id, so one user can never read, modify or
// fire another user's endpoint, and the signing secret is never echoed back.
type NotificationChannelHandler struct {
	channels   *service.NotificationChannelService
	deliveries *service.NotificationDeliveryService
}

func NewNotificationChannelHandler(channels *service.NotificationChannelService, deliveries *service.NotificationDeliveryService) *NotificationChannelHandler {
	return &NotificationChannelHandler{channels: channels, deliveries: deliveries}
}

type notificationChannelReq struct {
	Name    string   `json:"name"`
	URL     string   `json:"url" binding:"required"`
	Secret  string   `json:"secret"`
	Events  []string `json:"events" binding:"required"`
	Enabled *bool    `json:"enabled"`
}

func (h *NotificationChannelHandler) List(c *gin.Context) {
	rows, err := h.channels.List(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":            rows,
		"supported_events": service.SupportedUserNotificationEvents,
		"signature_header": "X-Webhook-Signature",
		"signature_scheme": "sha256=<hmac-sha256 of the raw body>",
	})
}

func (h *NotificationChannelHandler) Create(c *gin.Context) {
	var req notificationChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := h.channels.Create(c.Request.Context(), middleware.MustUserID(c),
		service.UpsertNotificationChannelInput{
			Name:    req.Name,
			URL:     req.URL,
			Secret:  req.Secret,
			Events:  req.Events,
			Enabled: enabled,
		})
	if err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (h *NotificationChannelHandler) Update(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	var req notificationChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	uid := middleware.MustUserID(c)

	// `enabled` is a pointer so PATCH can leave the current state alone.
	enabled := true
	if req.Enabled == nil {
		current, err := h.channels.Get(c.Request.Context(), uid, id)
		if err != nil {
			writeNotificationChannelError(c, err)
			return
		}
		enabled = current.Enabled
	} else {
		enabled = *req.Enabled
	}

	row, err := h.channels.Update(c.Request.Context(), uid, id,
		service.UpsertNotificationChannelInput{
			Name:    req.Name,
			URL:     req.URL,
			Secret:  req.Secret,
			Events:  req.Events,
			Enabled: enabled,
		})
	if err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *NotificationChannelHandler) Delete(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	uid := middleware.MustUserID(c)
	if _, err := h.channels.Get(c.Request.Context(), uid, id); err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	if err := h.channels.Delete(c.Request.Context(), uid, id); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Test fires a synthetic event so the user can verify the endpoint works
// without waiting for real device traffic.
func (h *NotificationChannelHandler) Test(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	if err := h.channels.SendTest(c.Request.Context(), middleware.MustUserID(c), id); err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true})
}

// Deliveries returns the outbox history for one channel: what was queued, how
// many attempts it took, and whether it is still pending, delivered or parked
// as failed. This is the operator-facing view that makes at-least-once
// delivery auditable instead of a black box.
func (h *NotificationChannelHandler) Deliveries(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	uid := middleware.MustUserID(c)
	if _, err := h.channels.Get(c.Request.Context(), uid, id); err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	if h.deliveries == nil {
		ProblemInternal(c, "delivery store is not configured")
		return
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	rows, err := h.deliveries.ListByChannel(c.Request.Context(), id, limit)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// RetryDelivery re-arms a parked (`failed`) delivery so the worker tries it
// again without waiting for a new event. Owner-scoped via the parent channel.
func (h *NotificationChannelHandler) RetryDelivery(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	deliveryID, err := strconv.ParseUint(c.Param("deliveryId"), 10, 32)
	if err != nil || deliveryID == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid delivery id")
		return
	}
	uid := middleware.MustUserID(c)
	if _, err := h.channels.Get(c.Request.Context(), uid, id); err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	if h.deliveries == nil {
		ProblemInternal(c, "delivery store is not configured")
		return
	}
	row, err := h.deliveries.Retry(c.Request.Context(), id, uint(deliveryID))
	if err != nil {
		if errors.Is(err, service.ErrDeliveryNotRetryable) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, "only failed deliveries can be retried")
			return
		}
		writeNotificationChannelError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// retryFailedReq is the optional body of both bulk-replay endpoints. An empty
// body keeps the original behaviour: replay everything in scope.
type retryFailedReq struct {
	EventType string `json:"event_type"`
	DeviceID  string `json:"device_id"`
	Limit     int    `json:"limit"`
	DryRun    bool   `json:"dry_run"`
}

// parseRetryFailedRequest reads the optional body and clamps the batch cap.
// Returns ok=false after writing a 400/500 of its own.
func parseRetryFailedRequest(c *gin.Context) (service.RetryFailedFilters, bool, bool) {
	var req retryFailedReq
	if c.Request != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return service.RetryFailedFilters{}, false, false
		}
	}
	if req.Limit < 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "limit must be zero (no cap) or positive")
		return service.RetryFailedFilters{}, false, false
	}
	limit := req.Limit
	if limit > service.MaxRetryFailedLimit {
		// Clamp rather than reject: the caller asked to drain the backlog, and
		// the response reports the effective cap so nothing is hidden.
		limit = service.MaxRetryFailedLimit
	}
	filters := service.RetryFailedFilters{
		EventType: strings.TrimSpace(req.EventType),
		DeviceID:  strings.TrimSpace(req.DeviceID),
		Limit:     limit,
	}
	return filters, req.DryRun, true
}

func retryFailedResponse(filters service.RetryFailedFilters, dryRun bool, res service.RetryFailedResult) gin.H {
	return gin.H{
		"success":  true,
		"matched":  res.Matched,
		"requeued": res.Requeued,
		"dry_run":  dryRun,
		"limit":    filters.Limit,
		"filters": gin.H{
			"event_type": filters.EventType,
			"device_id":  filters.DeviceID,
		},
	}
}

// RetryFailedDeliveries re-arms parked (`failed`) deliveries for a channel, for
// an operator who fixed the endpoint. Optional filters narrow the replay and a
// dry run reports what would happen without waking the delivery worker.
// Owner-scoped via the parent channel.
func (h *NotificationChannelHandler) RetryFailedDeliveries(c *gin.Context) {
	id, ok := notificationChannelID(c)
	if !ok {
		return
	}
	uid := middleware.MustUserID(c)
	if _, err := h.channels.Get(c.Request.Context(), uid, id); err != nil {
		writeNotificationChannelError(c, err)
		return
	}
	if h.deliveries == nil {
		ProblemInternal(c, "delivery store is not configured")
		return
	}
	filters, dryRun, ok := parseRetryFailedRequest(c)
	if !ok {
		return
	}
	res, err := h.deliveries.RetryFailedForChannel(c.Request.Context(), id, filters, dryRun)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, retryFailedResponse(filters, dryRun, res))
}

// RetryFailedDeliveriesForUser re-arms parked (`failed`) deliveries the caller
// owns, across all of their channels. One call is enough after an outage that
// took several endpoints down at once; the same filters and dry run apply.
func (h *NotificationChannelHandler) RetryFailedDeliveriesForUser(c *gin.Context) {
	if h.deliveries == nil {
		ProblemInternal(c, "delivery store is not configured")
		return
	}
	uid := middleware.MustUserID(c)
	filters, dryRun, ok := parseRetryFailedRequest(c)
	if !ok {
		return
	}
	res, err := h.deliveries.RetryFailedForUser(c.Request.Context(), uid, filters, dryRun)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, retryFailedResponse(filters, dryRun, res))
}

func notificationChannelID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid id")
		return 0, false
	}
	return uint(id), true
}

func writeNotificationChannelError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidNotificationURL),
		errors.Is(err, service.ErrUnsupportedNotificationEvent):
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		ProblemNotFound(c, ProblemCodeNotFound, "Notification channel not found")
	default:
		ProblemInternal(c, err.Error())
	}
}
