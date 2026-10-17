package handler

import (
	"errors"
	"net/http"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
)

type HostMetricHandler struct {
	metrics *service.DeviceMetricAlertService
}

func NewHostMetricHandler(metrics *service.DeviceMetricAlertService) *HostMetricHandler {
	return &HostMetricHandler{metrics: metrics}
}

type hostMetricReq struct {
	CapturedAt    time.Time `json:"captured_at"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent   float64   `json:"disk_percent"`
	DiskMount     string    `json:"disk_mount"`
}

func (h *HostMetricHandler) Report(c *gin.Context) {
	var req hostMetricReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	snapshot, err := h.metrics.Ingest(c.Request.Context(), middleware.MustDeviceID(c), service.IngestMetricInput{
		CapturedAt:    req.CapturedAt,
		CPUPercent:    req.CPUPercent,
		MemoryPercent: req.MemoryPercent,
		DiskPercent:   req.DiskPercent,
		DiskMount:     req.DiskMount,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidMetricAlert) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"device_id":   snapshot.DeviceID,
		"captured_at": snapshot.CapturedAt,
	})
}
