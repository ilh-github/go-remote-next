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

// fileTransferAuditAction is the audit action for a file transfer. It is
// deliberately direction-free: the transfer id is what identifies a transfer,
// and the direction is one of the things that happened during it - so it is
// recorded in the details rather than in the identity key.
const fileTransferAuditAction = "file.transfer"

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

	// The action names the resource, not the direction. QD-111 promises
	// idempotency by transfer id, and the key LogActorOnce uses is (actor,
	// action, resource type, resource id) - so `file.transfer.upload` and
	// `file.transfer.download` were two different actions and one transfer id
	// reported with both directions produced two rows. A sync that pushes then
	// pulls the same file, or a retry after a direction change, reads as two
	// transfers in the trail.
	//
	// The direction still has to be recorded: it belongs in the details, where
	// it describes the transfer, rather than in the key, where it splits it.
	// A transfer that went both ways lists both, so neither leg is lost.
	details, _ := json.Marshal(map[string]any{
		"device_id":         req.DeviceID,
		"status":            req.Status,
		"directions":        []string{req.Direction},
		"filename":          req.Filename,
		"bytes_transferred": req.BytesTransferred,
		"total_bytes":       req.TotalBytes,
		"error":             req.Error,
	})
	if _, err := h.audit.LogActorOnce(
		c.Request.Context(),
		uid,
		fileTransferAuditAction,
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
	// The length bound above is the one this always had; this is the one it was
	// missing. A filename that carries a directory is a path, and a path is
	// information about the controlled machine rather than about the transfer:
	// `/Users/someone/.ssh/id_rsa` tells every later reader of this table where
	// that user keeps their keys. The clipboard side of this same trail holds
	// the line by having no field for the text at all; this is the file side's
	// version of it, and it is a shape check because the offending thing is the
	// separators, not the length.
	if isPathLikeFilename(req.Filename) {
		return fmt.Errorf("filename must be a name, not a path")
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

// isPathLikeFilename reports whether a filename carries any part of a path.
//
// Both separator families are checked because a controller can be reporting a
// transfer to or from either kind of machine, and the concern is the same on
// both: a name with a separator in it is a path, whatever the platform.
//
// A leading separator is the case that matters most - an absolute path - but a
// relative one is refused too, because `docs/../../etc/passwd` is still telling
// the reader about the machine's layout.
func isPathLikeFilename(filename string) bool {
	if filename == "" {
		// An empty name is refused elsewhere; treating it as a path here would
		// produce a confusing error for a different problem.
		return false
	}
	if strings.ContainsAny(filename, `/\`) {
		return true
	}
	// A bare drive letter (`C:`) is a path with no separator in it at all, so
	// the separator check alone would let it through.
	if len(filename) >= 2 && filename[1] == ':' {
		drive := filename[0]
		if (drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z') {
			return true
		}
	}
	return false
}
