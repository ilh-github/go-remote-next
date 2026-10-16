package handler

import (
	"errors"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceOfflineAlertHandler serves the owner-controlled "tell me when this
// device stays offline" rule:
//
//	GET    /v1/me/devices/:device_id/offline-alert
//	PUT    /v1/me/devices/:device_id/offline-alert
//	DELETE /v1/me/devices/:device_id/offline-alert
//
// It is a rules endpoint rather than a "send an alert" endpoint: the evaluator
// runs in the background, and the alert itself is delivered through the user's
// notification channels.
type DeviceOfflineAlertHandler struct {
	devices *service.DeviceService
	alerts  *service.DeviceOfflineAlertService
}

func NewDeviceOfflineAlertHandler(devices *service.DeviceService, alerts *service.DeviceOfflineAlertService) *DeviceOfflineAlertHandler {
	return &DeviceOfflineAlertHandler{devices: devices, alerts: alerts}
}

type deviceOfflineAlertReq struct {
	Enabled      bool `json:"enabled"`
	AfterMinutes int  `json:"after_minutes"`
}

// deviceOfflineAlertView is the wire shape. `offline_since` and
// `last_alerted_at` are the evaluator's episode state, exposed so an owner can
// see why an alert has or has not fired yet.
type deviceOfflineAlertView struct {
	DeviceID      string     `json:"device_id"`
	Enabled       bool       `json:"enabled"`
	AfterMinutes  int        `json:"after_minutes"`
	OfflineSince  *time.Time `json:"offline_since,omitempty"`
	LastAlertedAt *time.Time `json:"last_alerted_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

func (h *DeviceOfflineAlertHandler) Get(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	rule, err := h.alerts.Get(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, offlineAlertView(deviceID, rule))
}

func (h *DeviceOfflineAlertHandler) Put(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	var req deviceOfflineAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if req.AfterMinutes == 0 {
		// An omitted field is far more likely than a deliberate zero, and a
		// zero threshold would mean "alert the moment the heartbeat expires".
		req.AfterMinutes = service.DefaultOfflineAlertAfterMinutes
	}
	uid := middleware.MustUserID(c)
	rule, err := h.alerts.Upsert(c.Request.Context(), deviceID, uid, req.Enabled, req.AfterMinutes)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOfflineAlertThreshold) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, offlineAlertView(deviceID, rule))
}

func (h *DeviceOfflineAlertHandler) Delete(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	if err := h.alerts.Delete(c.Request.Context(), deviceID); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DeviceOfflineAlertHandler) ownedDeviceID(c *gin.Context) (string, bool) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	d, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return "", false
		}
		ProblemInternal(c, err.Error())
		return "", false
	}
	if d.UserID == nil || *d.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return "", false
	}
	return deviceID, true
}

// offlineAlertView renders the rule, filling in the "no rule yet" defaults so
// a client can show the form before the owner has saved anything.
func offlineAlertView(deviceID string, rule *models.DeviceOfflineAlertRule) deviceOfflineAlertView {
	view := deviceOfflineAlertView{
		DeviceID:     deviceID,
		Enabled:      false,
		AfterMinutes: service.DefaultOfflineAlertAfterMinutes,
	}
	if rule == nil {
		return view
	}
	view.Enabled = rule.Enabled
	view.AfterMinutes = rule.AfterMinutes
	view.OfflineSince = rule.OfflineSince
	view.LastAlertedAt = rule.AlertedAt
	updated := rule.UpdatedAt
	view.UpdatedAt = &updated
	return view
}
