package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceAccessGrantHandler serves owner-managed time-boxed access grants:
//
//	GET    /v1/me/devices/:device_id/access-grants
//	POST   /v1/me/devices/:device_id/access-grants
//	DELETE /v1/me/devices/:device_id/access-grants/:grant_id
//
// A grant lets the owner admit a guest for a while (optionally once) without
// telling them the device's own rotating access code.
type DeviceAccessGrantHandler struct {
	devices *service.DeviceService
	grants  *service.DeviceAccessGrantService
}

func NewDeviceAccessGrantHandler(devices *service.DeviceService, grants *service.DeviceAccessGrantService) *DeviceAccessGrantHandler {
	return &DeviceAccessGrantHandler{devices: devices, grants: grants}
}

type accessGrantReq struct {
	Label            string `json:"label"`
	ExpiresInMinutes int    `json:"expires_in_minutes" binding:"required"`
	MaxUses          int    `json:"max_uses"`
	Code             string `json:"code"`
	// Optional binding: who may redeem the code. Empty means "anyone who has
	// the code", which is what a grant meant before these fields existed.
	BoundClientID string `json:"bound_client_id"`
	// Optional proof of possession for that client id: base64 of a raw 32-byte
	// Ed25519 public key. When set, redeeming the code requires a signature
	// from the matching private key over the server's challenge.
	BoundClientPublicKey string   `json:"bound_client_public_key"`
	BoundIPs             []string `json:"bound_ips"`
}

// accessGrantView is the wire shape: it adds the derived `active` flag so a
// client never has to re-implement the expiry/quota/revocation rules.
type accessGrantView struct {
	ID         uint       `json:"id"`
	DeviceID   string     `json:"device_id"`
	Code       string     `json:"code"`
	Label      string     `json:"label"`
	ExpiresAt  time.Time  `json:"expires_at"`
	MaxUses    int        `json:"max_uses"`
	Uses       int        `json:"uses"`
	Active     bool       `json:"active"`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	// BoundClientID / BoundIPs echo the binding, and LastUsed* show how the
	// code was actually redeemed.
	BoundClientID string `json:"bound_client_id"`
	// BoundClientPublicKey is echoed so the owner can tell an assertion-only
	// binding from a provable one without keeping track of which code got which
	// key. The key is public, so echoing it leaks nothing.
	BoundClientPublicKey string    `json:"bound_client_public_key"`
	BoundIPs             []string  `json:"bound_ips"`
	LastUsedClientID     string    `json:"last_used_client_id"`
	LastUsedIP           string    `json:"last_used_ip"`
	CreatedAt            time.Time `json:"created_at"`
}

func (h *DeviceAccessGrantHandler) List(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	rows, err := h.grants.List(c.Request.Context(), deviceID)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	now := time.Now().UTC()
	views := make([]accessGrantView, 0, len(rows))
	for _, row := range rows {
		views = append(views, grantView(row, now))
	}
	c.JSON(http.StatusOK, gin.H{"items": views})
}

func (h *DeviceAccessGrantHandler) Create(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	var req accessGrantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	actorIP := c.ClientIP()
	row, err := h.grants.CreateWithEvent(c.Request.Context(), deviceID, middleware.MustUserID(c),
		service.CreateDeviceAccessGrantInput{
			Label:                req.Label,
			ExpiresInMinutes:     req.ExpiresInMinutes,
			MaxUses:              req.MaxUses,
			Code:                 req.Code,
			BoundClientID:        req.BoundClientID,
			BoundClientPublicKey: req.BoundClientPublicKey,
			BoundIPs:             req.BoundIPs,
		}, func(row *models.DeviceAccessGrant) service.Event {
			return service.Event{
				Type:     service.EventDeviceAccessGrantChanged,
				UserID:   row.OwnerID,
				DeviceID: row.DeviceID,
				Data: map[string]interface{}{
					"action":     "created",
					"grant_id":   row.ID,
					"label":      row.Label,
					"expires_at": row.ExpiresAt.UTC().Format(time.RFC3339),
					"max_uses":   row.MaxUses,
					"uses":       row.Uses,
					"ip":         actorIP,
				},
			}
		})
	if err != nil {
		if errors.Is(err, service.ErrInvalidAccessGrant) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, grantView(*row, time.Now().UTC()))
}

func (h *DeviceAccessGrantHandler) Delete(c *gin.Context) {
	deviceID, ok := h.ownedDeviceID(c)
	if !ok {
		return
	}
	grantID, err := strconv.ParseUint(c.Param("grant_id"), 10, 32)
	if err != nil || grantID == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid grant id")
		return
	}
	row, err := h.grants.Get(c.Request.Context(), deviceID, uint(grantID))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if row == nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Access grant not found")
		return
	}
	actorIP := c.ClientIP()
	revoked, _, err := h.grants.RevokeWithEvent(
		c.Request.Context(), deviceID, uint(grantID),
		func(row *models.DeviceAccessGrant) service.Event {
			return service.Event{
				Type:     service.EventDeviceAccessGrantChanged,
				UserID:   row.OwnerID,
				DeviceID: row.DeviceID,
				Data: map[string]interface{}{
					"action":     "revoked",
					"grant_id":   row.ID,
					"label":      row.Label,
					"expires_at": row.ExpiresAt.UTC().Format(time.RFC3339),
					"max_uses":   row.MaxUses,
					"uses":       row.Uses,
					"ip":         actorIP,
				},
			}
		})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if revoked == nil {
		ProblemNotFound(c, ProblemCodeNotFound, "Access grant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ownedDeviceID resolves :device_id and enforces that the caller owns it.
func (h *DeviceAccessGrantHandler) ownedDeviceID(c *gin.Context) (string, bool) {
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

func grantView(row models.DeviceAccessGrant, now time.Time) accessGrantView {
	return accessGrantView{
		ID:                   row.ID,
		DeviceID:             row.DeviceID,
		Code:                 row.Code,
		Label:                row.Label,
		ExpiresAt:            row.ExpiresAt,
		MaxUses:              row.MaxUses,
		Uses:                 row.Uses,
		Active:               row.UsableAt(now),
		RevokedAt:            row.RevokedAt,
		LastUsedAt:           row.LastUsedAt,
		BoundClientID:        row.BoundClientID,
		BoundClientPublicKey: row.BoundClientPublicKey,
		BoundIPs:             decodeAllowIPsForEvent(row.BoundIPs),
		LastUsedClientID:     row.LastUsedClientID,
		LastUsedIP:           row.LastUsedIP,
		CreatedAt:            row.CreatedAt,
	}
}
