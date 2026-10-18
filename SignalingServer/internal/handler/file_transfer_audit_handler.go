package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FileTransferAuditHandler records control-client reports about file transfers
// that happen on the Remoting data channel. The report is attribution, not
// cryptographic proof from the host.
type FileTransferAuditHandler struct {
	devices *service.DeviceService
	audit   *service.AuditService
}

func NewFileTransferAuditHandler(
	devices *service.DeviceService,
	audit *service.AuditService,
) *FileTransferAuditHandler {
	return &FileTransferAuditHandler{devices: devices, audit: audit}
}

type fileTransferAuditRequest struct {
	DeviceID         string `json:"device_id" binding:"required"`
	TransferID       string `json:"transfer_id" binding:"required"`
	Direction        string `json:"direction" binding:"required"`
	Status           string `json:"status" binding:"required"`
	Filename         string `json:"filename"`
	BytesTransferred int64  `json:"bytes_transferred"`
	TotalBytes       int64  `json:"total_bytes"`
	Error            string `json:"error"`
}

func (h *FileTransferAuditHandler) Create(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req fileTransferAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.Direction = strings.TrimSpace(req.Direction)
	req.Status = strings.TrimSpace(req.Status)
	req.Filename = strings.TrimSpace(req.Filename)
	req.Error = strings.TrimSpace(req.Error)
	if err := validateFileTransferAudit(req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if h.audit == nil {
		ProblemInternal(c, "audit store is not configured")
		return
	}

	device, err := h.devices.GetByDeviceID(c.Request.Context(), req.DeviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if device.UserID == nil || *device.UserID != uid {
		ProblemForbidden(c, ProblemCodeForbidden, "Device is not owned by this user")
		return
	}

	details, _ := json.Marshal(map[string]any{
		"device_id":         req.DeviceID,
		"status":            req.Status,
		"filename":          req.Filename,
		"bytes_transferred": req.BytesTransferred,
		"total_bytes":       req.TotalBytes,
		"error":             req.Error,
	})
	if _, err := h.audit.LogActorOnce(
		c.Request.Context(),
		uid,
		"file.transfer."+req.Direction,
		"file_transfer",
		req.TransferID,
		string(details),
		c.ClientIP(),
	); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func validateFileTransferAudit(req fileTransferAuditRequest) error {
	if req.DeviceID == "" || utf8.RuneCountInString(req.DeviceID) > 20 {
		return fmt.Errorf("device_id must contain 1 to 20 characters")
	}
	if req.Direction != "upload" && req.Direction != "download" {
		return fmt.Errorf("direction must be upload or download")
	}
	if req.Status != "completed" && req.Status != "failed" {
		return fmt.Errorf("status must be completed or failed")
	}
	if req.TransferID == "" || utf8.RuneCountInString(req.TransferID) > 100 {
		return fmt.Errorf("transfer_id must contain 1 to 100 characters")
	}
	if utf8.RuneCountInString(req.Filename) > 255 {
		return fmt.Errorf("filename must not exceed 255 characters")
	}
	if utf8.RuneCountInString(req.Error) > 1000 {
		return fmt.Errorf("error must not exceed 1000 characters")
	}
	if req.BytesTransferred < 0 || req.TotalBytes < 0 {
		return fmt.Errorf("byte counts must not be negative")
	}
	if req.TotalBytes > 0 && req.BytesTransferred > req.TotalBytes {
		return fmt.Errorf("bytes_transferred must not exceed total_bytes")
	}
	if req.Status == "completed" && req.TotalBytes > 0 && req.BytesTransferred != req.TotalBytes {
		return fmt.Errorf("a completed transfer must include every declared byte")
	}
	return nil
}
