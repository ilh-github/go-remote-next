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

// ClipboardAuditHandler records control-client reports about clipboard syncs
// that happen on the Remoting data channel.
//
// The report carries metadata only - direction, MIME type, byte count. It
// deliberately has no field for the clipboard text, and that is the whole point
// of this endpoint existing separately from a generic log: a clipboard is by
// definition whatever the user last copied, which is routinely a password, a
// recovery code or a private message. An audit row that could say "the operator
// copied 1.2 KB of text at 14:03" answers the investigative question; one that
// stored the text would turn the audit trail into a credential store that every
// future reader of the database inherits.
type ClipboardAuditHandler struct {
	devices *service.DeviceService
	audit   *service.AuditService
}

func NewClipboardAuditHandler(
	devices *service.DeviceService,
	audit *service.AuditService,
) *ClipboardAuditHandler {
	return &ClipboardAuditHandler{devices: devices, audit: audit}
}

type clipboardAuditRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	TransferID string `json:"transfer_id" binding:"required"`
	Direction  string `json:"direction" binding:"required"`
	Status     string `json:"status" binding:"required"`
	MimeType   string `json:"mime_type" binding:"required"`
	Bytes      int64  `json:"bytes"`
}

func (h *ClipboardAuditHandler) Create(c *gin.Context) {
	uid := middleware.MustUserID(c)
	var req clipboardAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.TransferID = strings.TrimSpace(req.TransferID)
	req.Direction = strings.TrimSpace(req.Direction)
	req.Status = strings.TrimSpace(req.Status)
	req.MimeType = strings.TrimSpace(req.MimeType)
	if err := validateClipboardAudit(req); err != nil {
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
		"device_id": req.DeviceID,
		"status":    req.Status,
		"mime_type": req.MimeType,
		"bytes":     req.Bytes,
	})
	if _, err := h.audit.LogActorOnce(
		c.Request.Context(),
		uid,
		"clipboard.sync."+req.Direction,
		"clipboard_sync",
		req.TransferID,
		string(details),
		c.ClientIP(),
	); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func validateClipboardAudit(req clipboardAuditRequest) error {
	if req.DeviceID == "" || utf8.RuneCountInString(req.DeviceID) > 20 {
		return fmt.Errorf("device_id must contain 1 to 20 characters")
	}
	if req.TransferID == "" || utf8.RuneCountInString(req.TransferID) > 100 {
		return fmt.Errorf("transfer_id must contain 1 to 100 characters")
	}
	// Which way the bytes went is the fact an investigator needs, so an
	// unrecognised direction is refused rather than coerced.
	if req.Direction != "upload" && req.Direction != "download" {
		return fmt.Errorf("direction must be upload or download")
	}
	if req.Status != "completed" && req.Status != "failed" {
		return fmt.Errorf("status must be completed or failed")
	}
	// The MIME type is metadata, so it is bounded and never the payload. An
	// empty one is refused: a report that cannot say what was copied is not
	// worth an audit row.
	if req.MimeType == "" || utf8.RuneCountInString(req.MimeType) > 100 {
		return fmt.Errorf("mime_type must contain 1 to 100 characters")
	}
	if req.Bytes < 0 {
		return fmt.Errorf("bytes must not be negative")
	}
	return nil
}
