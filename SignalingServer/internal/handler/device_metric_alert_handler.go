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

// DeviceMetricAlertHandler serves owner-defined host resource thresholds.
//
//	GET    /v1/me/devices/:device_id/metric-alert
//	PUT    /v1/me/devices/:device_id/metric-alert
//	DELETE /v1/me/devices/:device_id/metric-alert
type DeviceMetricAlertHandler struct {
	devices *service.DeviceService
	alerts  *service.DeviceMetricAlertService
}

func NewDeviceMetricAlertHandler(
	devices *service.DeviceService,
	alerts *service.DeviceMetricAlertService,
) *DeviceMetricAlertHandler {
	return &DeviceMetricAlertHandler{devices: devices, alerts: alerts}
}

type deviceMetricAlertReq struct {
	Enabled          bool       `json:"enabled"`
	CPUPercent       int        `json:"cpu_percent"`
	MemoryPercent    int        `json:"memory_percent"`
	DiskPercent      int        `json:"disk_percent"`
	SustainedMinutes int        `json:"sustained_minutes"`
	RepeatMinutes    int        `json:"repeat_minutes"`
	SnoozedUntil     *time.Time `json:"snoozed_until"`
}

type metricSnapshotView struct {
	CapturedAt    time.Time `json:"captured_at"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent   float64   `json:"disk_percent"`
	DiskMount     string    `json:"disk_mount"`
}

type metricEpisodeView struct {
	ExceededSince *time.Time `json:"exceeded_since"`
	LastAlertedAt *time.Time `json:"last_alerted_at"`
}

type metricAlertStatesView struct {
	CPU    metricEpisodeView `json:"cpu"`
	Memory metricEpisodeView `json:"memory"`
	Disk   metricEpisodeView `json:"disk"`
}

type deviceMetricAlertView struct {
	DeviceID         string                `json:"device_id"`
	Enabled          bool                  `json:"enabled"`
	CPUPercent       int                   `json:"cpu_percent"`
	MemoryPercent    int                   `json:"memory_percent"`
	DiskPercent      int                   `json:"disk_percent"`
	SustainedMinutes int                   `json:"sustained_minutes"`
	RepeatMinutes    int                   `json:"repeat_minutes"`
	SnoozedUntil     *time.Time            `json:"snoozed_until"`
	Current          *metricSnapshotView   `json:"current"`
	States           metricAlertStatesView `json:"states"`
	UpdatedAt        *time.Time            `json:"updated_at"`
}

func (h *DeviceMetricAlertHandler) Get(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	rule, err := h.alerts.Get(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	snapshot, err := h.alerts.Latest(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, metricAlertView(deviceID, rule, snapshot))
}

func (h *DeviceMetricAlertHandler) Put(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	var req deviceMetricAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	rule, err := h.alerts.Upsert(
		c.Request.Context(),
		deviceID,
		middleware.MustUserID(c),
		service.UpsertMetricAlertInput{
			Enabled:          req.Enabled,
			CPUPercent:       req.CPUPercent,
			MemoryPercent:    req.MemoryPercent,
			DiskPercent:      req.DiskPercent,
			SustainedMinutes: req.SustainedMinutes,
			RepeatMinutes:    req.RepeatMinutes,
			SnoozedUntil:     req.SnoozedUntil,
		},
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMetricAlert) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	snapshot, err := h.alerts.Latest(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, metricAlertView(deviceID, rule, snapshot))
}

func (h *DeviceMetricAlertHandler) Delete(c *gin.Context) {
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

func (h *DeviceMetricAlertHandler) ownedDeviceID(c *gin.Context) (string, bool) {
	uid := middleware.MustUserID(c)
	deviceID := c.Param("device_id")
	device, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return "", false
		}
		ProblemInternal(c, err.Error())
		return "", false
	}
	if device.UserID == nil || *device.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return "", false
	}
	return deviceID, true
}

func metricAlertView(
	deviceID string,
	rule *models.DeviceMetricAlertRule,
	snapshot *models.DeviceMetricSnapshot,
) deviceMetricAlertView {
	view := deviceMetricAlertView{
		DeviceID:         deviceID,
		SustainedMinutes: service.DefaultMetricSustainedMinutes,
		States:           metricAlertStatesView{},
	}
	if snapshot != nil {
		view.Current = &metricSnapshotView{
			CapturedAt:    snapshot.CapturedAt,
			CPUPercent:    snapshot.CPUPercent,
			MemoryPercent: snapshot.MemoryPercent,
			DiskPercent:   snapshot.DiskPercent,
			DiskMount:     snapshot.DiskMount,
		}
	}
	if rule == nil {
		return view
	}
	view.Enabled = rule.Enabled
	view.CPUPercent = rule.CPUPercent
	view.MemoryPercent = rule.MemoryPercent
	view.DiskPercent = rule.DiskPercent
	view.SustainedMinutes = rule.SustainedMinutes
	view.RepeatMinutes = rule.RepeatMinutes
	view.SnoozedUntil = rule.SnoozedUntil
	view.States = metricAlertStatesView{
		CPU: metricEpisodeView{
			ExceededSince: rule.CPUExceededSince,
			LastAlertedAt: rule.CPUAlertedAt,
		},
		Memory: metricEpisodeView{
			ExceededSince: rule.MemoryExceededSince,
			LastAlertedAt: rule.MemoryAlertedAt,
		},
		Disk: metricEpisodeView{
			ExceededSince: rule.DiskExceededSince,
			LastAlertedAt: rule.DiskAlertedAt,
		},
	}
	updated := rule.UpdatedAt
	view.UpdatedAt = &updated
	return view
}
