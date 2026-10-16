package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/observability"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HostHandler implements the device-side surface under /v1/devices/*:
//
//	POST /v1/devices:provision                    (X-API-Key only)
//	POST /v1/devices/:device_id/heartbeat         (device_secret)
//	POST /v1/devices/:device_id/signal-tokens     (device_secret)
//	PUT  /v1/devices/:device_id/access-code       (device_secret)
//	GET  /v1/ice-config                           (X-API-Key OR device_secret)
//	POST /v1/devices/:device_id/access-code:verify (X-API-Key OR Origin-allowed)
//
// See docs §2.5, §2.6, §2.10, §2.19, §2.23.
type HostHandler struct {
	devices         *service.DeviceService
	tokens          *service.TokenService
	presence        *service.PresenceService
	settings        *service.SettingsService
	rateLimit       *service.RateLimitService
	access          *service.DeviceAccessService
	accessPolicy    *service.DeviceAccessPolicyService
	accessGrants    *service.DeviceAccessGrantService
	accessApprovals *service.DeviceAccessApprovalService
	bus             *service.EventBus
	serverSince     time.Time
	appConfig       *config.Config
}

func NewHostHandler(
	devices *service.DeviceService,
	tokens *service.TokenService,
	presence *service.PresenceService,
	settings *service.SettingsService,
	rateLimit *service.RateLimitService,
	access *service.DeviceAccessService,
	accessPolicy *service.DeviceAccessPolicyService,
	accessGrants *service.DeviceAccessGrantService,
	accessApprovals *service.DeviceAccessApprovalService,
	bus *service.EventBus,
	appConfig *config.Config,
) *HostHandler {
	return &HostHandler{
		devices:         devices,
		tokens:          tokens,
		presence:        presence,
		settings:        settings,
		rateLimit:       rateLimit,
		access:          access,
		accessPolicy:    accessPolicy,
		accessGrants:    accessGrants,
		accessApprovals: accessApprovals,
		bus:             bus,
		serverSince:     time.Now().UTC(),
		appConfig:       appConfig,
	}
}

// -----------------------------------------------------------------------
// POST /v1/devices:provision
// -----------------------------------------------------------------------

type provisionReq struct {
	DeviceUUID         string `json:"device_uuid" binding:"required"`
	MachineFingerprint string `json:"machine_fingerprint"`
	OS                 string `json:"os"`
	OSVersion          string `json:"os_version"`
	AppVersion         string `json:"app_version"`
}

type provisionResp struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"` // returned exactly once
	IsNew        bool   `json:"is_new"`
}

func (h *HostHandler) Provision(c *gin.Context) {
	var req provisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	result, err := h.devices.Provision(c.Request.Context(), service.ProvisionRequest{
		DeviceUUID:         req.DeviceUUID,
		MachineFingerprint: req.MachineFingerprint,
		OS:                 req.OS,
		OSVersion:          req.OSVersion,
		AppVersion:         req.AppVersion,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	observability.Event("device", "provisioned", map[string]interface{}{
		"device_id":  result.DeviceID,
		"is_new":     result.IsNew,
		"request_id": c.GetString("request_id"),
	})
	c.JSON(http.StatusOK, provisionResp{
		DeviceID:     result.DeviceID,
		DeviceSecret: result.DeviceSecret,
		IsNew:        result.IsNew,
	})
}

// -----------------------------------------------------------------------
// POST /v1/devices/:device_id/heartbeat
// -----------------------------------------------------------------------

type heartbeatReq struct {
	AppVersion string `json:"app_version"`
	OS         string `json:"os"`
	OSVersion  string `json:"os_version"`
}

type heartbeatResp struct {
	ServerTime                    string `json:"server_time"`
	TurnConfigVersion             int64  `json:"turn_config_version"`
	SuggestedHeartbeatIntervalSec int    `json:"suggested_heartbeat_interval_sec,omitempty"`
}

func (h *HostHandler) Heartbeat(c *gin.Context) {
	deviceID := middleware.MustDeviceID(c)
	// Basic 1s-floor rate limit (§2.10 footnote).
	if blocked, _ := h.rateLimit.HeartbeatThrottle(c.Request.Context(), deviceID); blocked {
		ProblemTooManyRequests(c, ProblemCodeRateLimited, "Heartbeat rate exceeded", 1)
		return
	}
	var req heartbeatReq
	_ = c.ShouldBindJSON(&req)

	if err := h.devices.Heartbeat(c.Request.Context(), deviceID, req.OS, req.OSVersion, req.AppVersion); err != nil {
		ProblemInternal(c, err.Error())
		return
	}

	// Refresh presence heartbeat TTL.
	if err := h.presence.Heartbeat(c.Request.Context(), deviceID); err != nil {
		ProblemInternal(c, "failed to refresh heartbeat presence")
		return
	}
	state := h.presence.State(c.Request.Context(), deviceID)
	rememberedNow := false
	if state.Online {
		rememberedNow = h.presence.RememberOnlineCandidate(c.Request.Context(), deviceID)
	}
	if state.Online && rememberedNow {
		log.Printf("[presence] device online via heartbeat device=%s presence={%s}", deviceID, state.String())
		if d, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID); err == nil && d.UserID != nil {
			h.bus.Publish(c.Request.Context(), service.Event{
				Type:     service.EventDeviceOnlineChanged,
				UserID:   *d.UserID,
				DeviceID: deviceID,
				Data: map[string]interface{}{
					"device_id": deviceID,
					"online":    true,
					"logged_in": d.LoggedIn,
				},
			})
		}
	} else if state.Online {
		log.Printf("[presence] heartbeat online_event_skipped device=%s presence={%s} remembered_now=%t",
			deviceID, state.String(), rememberedNow)
	}

	c.JSON(http.StatusOK, heartbeatResp{
		ServerTime:        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		TurnConfigVersion: h.settings.Get().TurnConfigVersion,
	})
}

// -----------------------------------------------------------------------
// POST /v1/devices/:device_id/signal-tokens
// -----------------------------------------------------------------------

type signalTokenResp struct {
	SignalToken string `json:"signal_token"`
	ExpiresAt   string `json:"expires_at"`
}

func (h *HostHandler) IssueHostSignalToken(c *gin.Context) {
	deviceID := middleware.MustDeviceID(c)
	if blocked, _ := h.rateLimit.SignalTokenThrottle(c.Request.Context(), deviceID); blocked {
		observability.Event("signal", "host_token_rate_limited", map[string]interface{}{
			"device_id": deviceID, "request_id": c.GetString("request_id"),
		})
		ProblemTooManyRequests(c, ProblemCodeRateLimited, "signal-tokens rate exceeded", 1)
		return
	}
	tok, exp, err := h.tokens.IssueSignalToken(c.Request.Context(), service.SignalTokenPayload{
		DeviceID: deviceID,
		Role:     service.SignalRoleHost,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, signalTokenResp{
		SignalToken: tok,
		ExpiresAt:   exp.Format("2006-01-02T15:04:05Z"),
	})
}

// -----------------------------------------------------------------------
// PUT /v1/devices/:device_id/access-code  (called by Qt, §2.23)
// -----------------------------------------------------------------------

type setAccessCodeReq struct {
	AccessCode string `json:"access_code" binding:"required"`
}

func (h *HostHandler) SetAccessCode(c *gin.Context) {
	deviceID := middleware.MustDeviceID(c)
	var req setAccessCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	if err := h.devices.SetAccessCode(c.Request.Context(), deviceID, req.AccessCode); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	// Owner gets notified so their UI refreshes. Only publish to the
	// device's current owner (if any); unbound devices produce no user
	// event (webhook/audit subscribers see it anyway).
	d, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID)
	if err == nil && d.UserID != nil {
		h.bus.Publish(c.Request.Context(), service.Event{
			Type:     service.EventDeviceAccessCodeChanged,
			UserID:   *d.UserID,
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"device_id":           deviceID,
				"access_code_changed": true,
			},
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// -----------------------------------------------------------------------
// GET /v1/ice-config
// -----------------------------------------------------------------------

type iceServerJSON struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type iceConfigResp struct {
	IceServers        []iceServerJSON `json:"ice_servers"`
	TurnConfigVersion int64           `json:"turn_config_version"`
	// LifetimeDuration is the Google-TURN-compatible freshness hint
	// (e.g. "86400s"). Required for remoting's protocol::IceConfig::Parse
	// to compute a sensible expiration_time — without it, Chromium logs
	// an error on every fetch and marks the config immediately expired,
	// causing the transport to refetch before every new connection. The
	// Qt / WebClient layers ignore this field today. (R16)
	LifetimeDuration string `json:"lifetime_duration"`
}

// GetICEConfig generates TURN credentials against the coturn shared secret
// and returns the full ICE servers list. The endpoint is callable with
// either X-API-Key (host startup) or a valid device_secret Bearer.
func (h *HostHandler) GetICEConfig(c *gin.Context) {
	s := h.settings.Get()

	servers := make([]iceServerJSON, 0, 4)
	for _, u := range splitLines(s.StunURLs) {
		if u != "" {
			servers = append(servers, iceServerJSON{URLs: []string{u}})
		}
	}
	turnURLs := splitLines(s.TurnURLs)
	if len(turnURLs) > 0 && s.TurnAuthSecret != "" {
		user, cred := buildTurnCredential(s.TurnAuthSecret, s.TurnCredentialTTL)
		servers = append(servers, iceServerJSON{
			URLs:       turnURLs,
			Username:   user,
			Credential: cred,
		})
	}
	// Freshness hint. Mirror coturn's TTL (default 86400 when unset) so
	// IceConfig::is_expired() doesn't flip true until the TURN creds
	// actually rotate out.
	ttl := s.TurnCredentialTTL
	if ttl <= 0 {
		ttl = 86400
	}
	c.JSON(http.StatusOK, iceConfigResp{
		IceServers:        servers,
		TurnConfigVersion: s.TurnConfigVersion,
		LifetimeDuration:  fmt.Sprintf("%ds", ttl),
	})
}

// splitLines splits a newline/comma-delimited blob into trimmed tokens.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' })
	out := parts[:0]
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// buildTurnCredential implements coturn's `use-auth-secret` pattern:
//
//	username = "<unix_ts>:<anything>"
//	credential = base64( HMAC-SHA1(secret, username) )
func buildTurnCredential(secret string, ttlSec int) (user, cred string) {
	if ttlSec <= 0 {
		ttlSec = 86400
	}
	user = fmt.Sprintf("%d:quickdesk", time.Now().Unix()+int64(ttlSec))
	cred = hmacSHA1Base64(secret, user)
	return user, cred
}

// -----------------------------------------------------------------------
// POST /v1/devices/:device_id/access-code:verify
// -----------------------------------------------------------------------

type verifyCodeReq struct {
	Code     string `json:"code" binding:"required"`
	ClientID string `json:"client_id"`
}

type verifyCodeResp struct {
	SignalToken string `json:"signal_token"`
	ExpiresAt   string `json:"expires_at"`
}

// approvalRequiredResp is the 202 the controller gets instead of a token when
// the device policy wants the owner to confirm the session. The client polls
// /v1/devices/:id/access-approvals:claim with approval_id + approval_secret.
type approvalRequiredResp struct {
	ApprovalID     uint   `json:"approval_id"`
	ApprovalSecret string `json:"approval_secret"`
	Status         string `json:"status"`
	ExpiresAt      string `json:"expires_at"`
	PollAfterMS    int    `json:"poll_after_ms"`
}

func approvalSource(grant *models.DeviceAccessGrant) string {
	if grant != nil {
		return models.ApprovalSourceGrant
	}
	return models.ApprovalSourceDeviceCode
}

func (h *HostHandler) VerifyAccessCode(c *gin.Context) {
	deviceID := c.Param("device_id")
	ip := c.ClientIP()

	// Rate-limit precheck (§2.10): per-IP ⇒ 429, per-device or
	// per-(device,ip) ⇒ 403 TOO_MANY_ATTEMPTS.
	if decision, _ := h.rateLimit.CheckVerifyPreflight(c.Request.Context(), deviceID, ip); decision.Blocked {
		h.recordDeviceAccess(c, deviceID, "", service.DeviceAccessDeniedRateLimited, decision.Reason)
		switch decision.Kind {
		case service.VerifyBlockPerIP:
			ProblemTooManyRequests(c, ProblemCodeRateLimited, decision.Reason, decision.RetryAfterSec)
		default:
			// per-device or per-(device,ip): 403 + Retry-After (§2.10).
			if decision.RetryAfterSec > 0 {
				c.Header("Retry-After", fmt.Sprintf("%d", decision.RetryAfterSec))
			}
			ProblemForbidden(c, ProblemCodeTooManyAttempts, decision.Reason)
		}
		return
	}

	// Device-level access policy, owned by the device owner: a device with
	// remote access switched off, or a client outside the allow list, never
	// reaches the access-code comparison at all. Checked after the rate
	// limiter so a blocked caller cannot use this path to spray requests.
	var policy *models.DeviceAccessPolicy
	if h.accessPolicy != nil {
		loaded, err := h.accessPolicy.Get(c.Request.Context(), deviceID)
		if err != nil {
			ProblemInternal(c, err.Error())
			return
		}
		policy = loaded
		if allowed, reason := service.EvaluateDeviceAccessPolicy(policy, ip); !allowed {
			h.recordDeviceAccess(c, deviceID, "", service.DeviceAccessDeniedPolicy, reason)
			observability.Event("access_verify", "policy_denied", map[string]interface{}{
				"device_id": deviceID, "ip": ip, "reason": reason,
				"request_id": c.GetString("request_id"),
			})
			ProblemForbidden(c, ProblemCodeAccessPolicyDenied, reason)
			return
		}
	}

	var req verifyCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}

	exists, matches, err := h.devices.VerifyAccessCode(c.Request.Context(), deviceID, req.Code)
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	if !exists {
		ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not registered")
		return
	}
	// A time-boxed grant is an alternative to the device's own code. It got
	// here only after the policy gate above, so a grant can never open a device
	// the owner has switched off or restricted. It is kept (not consumed yet)
	// so that an offline host does not burn one of the grant's uses.
	var usedGrant *models.DeviceAccessGrant
	if !matches && h.accessGrants != nil {
		grant, gerr := h.accessGrants.Match(c.Request.Context(), deviceID, req.Code)
		if gerr != nil {
			ProblemInternal(c, gerr.Error())
			return
		}
		if grant != nil {
			matches = true
			usedGrant = grant
		}
	}
	// IMPORTANT ordering (R5 fix):
	//   1. wrong code  → count failure + INVALID_CODE / TOO_MANY_ATTEMPTS
	//   2. right code + host offline → HOST_OFFLINE (don't count as failure)
	//   3. right code + host online  → mint signal_token
	// Checking online-ness BEFORE code-match would let an attacker
	// enumerate access_codes against an offline device without ever
	// tripping the per-device failure counter (§2.10). Keep the code
	// comparison first so rate limiting engages consistently regardless
	// of host presence.
	if !matches {
		if failure, _ := h.rateLimit.RecordVerifyFailure(c.Request.Context(), deviceID, ip); failure.TripsLimit {
			h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessDeniedRateLimited,
				"too many failed attempts")
			observability.Event("access_verify", "rate_limited", map[string]interface{}{
				"device_id": deviceID, "ip": ip, "request_id": c.GetString("request_id"),
			})
			// Tripped a per-device error counter (not the per-IP total
			// counter), so §2.10 mandates 403 + Retry-After.
			if failure.RetryAfterSec > 0 {
				c.Header("Retry-After", fmt.Sprintf("%d", failure.RetryAfterSec))
			}
			ProblemForbidden(c, ProblemCodeTooManyAttempts, "Too many failed attempts")
			return
		}
		h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessDeniedInvalidCode,
			"access code is incorrect")
		observability.Event("access_verify", "rejected", map[string]interface{}{
			"device_id": deviceID, "ip": ip, "request_id": c.GetString("request_id"), "reason": "invalid_code",
		})
		ProblemForbidden(c, ProblemCodeInvalidCode, "Access code is incorrect")
		return
	}
	if !h.presence.IsOnline(c.Request.Context(), deviceID) {
		// Conflict per §2.2 / scenario 37. Successful code match resets
		// the per-(device,ip) failure counter — this is a legitimate
		// caller who just happens to have arrived while the host is
		// offline.
		state := h.presence.State(c.Request.Context(), deviceID)
		log.Printf("[access-code:verify] HOST_OFFLINE device=%s ip=%s presence={%s}",
			deviceID, ip, state.String())
		h.rateLimit.ResetVerifyFailures(c.Request.Context(), deviceID, ip)
		h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessHostOffline, "host is offline")
		ProblemConflict(c, ProblemCodeHostOffline, "Host is offline")
		return
	}

	// Code matched → reset counters and mint a single-use signal_token.
	h.rateLimit.ResetVerifyFailures(c.Request.Context(), deviceID, ip)

	// Second factor: the code is right, but this device's owner wants to say
	// yes first. Instead of a token, the controller gets a pending request it
	// polls with the one-time secret until the owner decides.
	if service.RequiresApproval(policy) {
		// Fail closed: a policy that demands approval must never fall through
		// to "mint a token anyway" because the service was not wired.
		if h.accessApprovals == nil {
			ProblemInternal(c, "approval service is not configured")
			return
		}
		var grantID *uint
		if usedGrant != nil {
			grantID = &usedGrant.ID
		}
		ownerID := uint(0)
		if h.access != nil {
			ownerID = h.access.OwnerID(c.Request.Context(), deviceID)
		}
		row, secret, aerr := h.accessApprovals.Create(c.Request.Context(),
			service.CreateDeviceAccessApprovalInput{
				DeviceID: deviceID,
				OwnerID:  ownerID,
				ClientID: req.ClientID,
				ClientIP: ip,
				Source:   approvalSource(usedGrant),
				GrantID:  grantID,
			})
		if aerr != nil {
			ProblemInternal(c, aerr.Error())
			return
		}
		h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessPendingApproval,
			"waiting for owner approval")
		observability.Event("access_verify", "approval_pending", map[string]interface{}{
			"device_id": deviceID, "ip": ip, "approval_id": row.ID,
			"request_id": c.GetString("request_id"),
		})
		if h.bus != nil && ownerID != 0 {
			h.bus.Publish(c.Request.Context(), service.Event{
				Type:     service.EventDeviceAccessApprovalRequested,
				UserID:   ownerID,
				DeviceID: deviceID,
				Data: map[string]interface{}{
					"approval_id": row.ID,
					"client_id":   req.ClientID,
					"client_ip":   ip,
					"source":      row.Source,
					"expires_at":  row.ExpiresAt.UTC().Format(time.RFC3339),
				},
			})
		}
		c.JSON(http.StatusAccepted, approvalRequiredResp{
			ApprovalID:     row.ID,
			ApprovalSecret: secret,
			Status:         models.ApprovalPending,
			ExpiresAt:      row.ExpiresAt.UTC().Format(time.RFC3339),
			PollAfterMS:    2000,
		})
		return
	}

	// A grant only counts a use once a session can actually start. Consume
	// re-checks the window and quota in its own UPDATE, so two controllers
	// racing the last slot cannot both win.
	if usedGrant != nil {
		consumed, cerr := h.accessGrants.Consume(c.Request.Context(), usedGrant.ID)
		if cerr != nil {
			ProblemInternal(c, cerr.Error())
			return
		}
		if !consumed {
			h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessDeniedInvalidCode,
				"access grant is no longer valid")
			observability.Event("access_verify", "rejected", map[string]interface{}{
				"device_id": deviceID, "ip": ip,
				"request_id": c.GetString("request_id"), "reason": "grant_exhausted",
			})
			ProblemForbidden(c, ProblemCodeInvalidCode, "Access code is incorrect")
			return
		}
		if h.bus != nil {
			// Tell the owner their shared code was just used. `UsedGrant.Uses`
			// is the value read before the increment, so +1 is the new count.
			h.bus.Publish(c.Request.Context(), service.Event{
				Type:     service.EventDeviceAccessGrantUsed,
				UserID:   usedGrant.OwnerID,
				DeviceID: deviceID,
				Data: map[string]interface{}{
					"grant_id":  usedGrant.ID,
					"label":     usedGrant.Label,
					"uses":      usedGrant.Uses + 1,
					"max_uses":  usedGrant.MaxUses,
					"client_id": req.ClientID,
				},
			})
		}
	}

	tok, exp, err := h.tokens.IssueSignalToken(c.Request.Context(), service.SignalTokenPayload{
		DeviceID: deviceID,
		Role:     service.SignalRoleClient,
		ClientID: req.ClientID,
	})
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	observability.Event("access_verify", "accepted", map[string]interface{}{
		"client_id": req.ClientID, "device_id": deviceID, "ip": ip, "request_id": c.GetString("request_id"),
	})
	grantNote := ""
	if usedGrant != nil {
		grantNote = fmt.Sprintf("access grant %d", usedGrant.ID)
	}
	h.recordDeviceAccess(c, deviceID, req.ClientID, service.DeviceAccessGranted, grantNote)
	c.JSON(http.StatusOK, verifyCodeResp{
		SignalToken: tok,
		ExpiresAt:   exp.Format("2006-01-02T15:04:05Z"),
	})
}

// recordDeviceAccess persists one access attempt and fans it out to the
// device owner. Audit writes must never alter the verification response, so
// failures are logged and swallowed.
func (h *HostHandler) recordDeviceAccess(c *gin.Context, deviceID, clientID, result, reason string) {
	if h.access == nil {
		return
	}
	row, err := h.access.Record(c.Request.Context(), service.RecordDeviceAccessInput{
		DeviceID: deviceID,
		ClientID: clientID,
		IP:       c.ClientIP(),
		Result:   result,
		Reason:   reason,
	})
	if err != nil {
		log.Printf("[device-access] record %s device=%s failed: %v", result, deviceID, err)
		return
	}
	if row.OwnerID == 0 || h.bus == nil {
		return
	}
	// A pending approval is not a denial: the caller publishes
	// device.access_approval.requested separately. Keep it out of the
	// approved/denied realtime stream so the owner is not told "denied" while
	// they still have a request to answer.
	if result == service.DeviceAccessPendingApproval {
		return
	}
	eventType := service.EventDeviceAccessGranted
	if result != service.DeviceAccessGranted {
		eventType = service.EventDeviceAccessDenied
	}
	h.bus.Publish(c.Request.Context(), service.Event{
		Type:     eventType,
		UserID:   row.OwnerID,
		DeviceID: deviceID,
		Data: map[string]interface{}{
			"device_id": deviceID,
			"client_id": row.ClientID,
			"ip":        row.IP,
			"result":    row.Result,
			"reason":    row.Reason,
			"log_id":    row.ID,
		},
	})
}

// -----------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------

// ErrDeviceRecordGone is returned when a device_secret is valid but the
// underlying row has been deleted — this is how an admin secret:rotate
// tells an offending host to re-provision (§2.17).
var ErrDeviceRecordGone = errors.New("device row removed")

// ensureDeviceExists is a tiny guard for endpoints that want a pre-check
// without pulling the whole row.
func (h *HostHandler) ensureDeviceExists(c *gin.Context, deviceID string) bool {
	if _, err := h.devices.GetByDeviceID(c.Request.Context(), deviceID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeDeviceNotFound, "Device not registered")
		} else {
			ProblemInternal(c, err.Error())
		}
		return false
	}
	return true
}
