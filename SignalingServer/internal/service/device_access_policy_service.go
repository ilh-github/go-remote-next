package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidAllowIP means one of the allow-list entries is neither an IP nor
// a CIDR block.
var ErrInvalidAllowIP = errors.New("allow list entries must be IP addresses or CIDR blocks")

// ErrInvalidSessionLimit means max_sessions was negative.
var ErrInvalidSessionLimit = errors.New("max_sessions must be zero (unlimited) or a positive number")

// DeviceAccessPolicyService stores and evaluates per-device access policy.
type DeviceAccessPolicyService struct {
	db *gorm.DB
}

func NewDeviceAccessPolicyService(db *gorm.DB) *DeviceAccessPolicyService {
	return &DeviceAccessPolicyService{db: db}
}

// UpsertDeviceAccessPolicyInput is the validated body of the policy endpoint.
type UpsertDeviceAccessPolicyInput struct {
	RemoteDisabled   bool
	AllowIPs         []string
	MaxSessions      int
	AllowedWindows   []AccessWindow
	Timezone         string
	ExceptionDates   []AccessExceptionDate
	ApprovalRequired bool
}

// Get returns the device policy, or (nil, nil) when the device has none: a
// missing row means "no restrictions", which is what every device had before
// this feature existed.
func (s *DeviceAccessPolicyService) Get(ctx context.Context, deviceID string) (*models.DeviceAccessPolicy, error) {
	var row models.DeviceAccessPolicy
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// Upsert writes the policy for a device, creating the row on first use.
func (s *DeviceAccessPolicyService) Upsert(ctx context.Context, deviceID string, ownerID uint, in UpsertDeviceAccessPolicyInput) (*models.DeviceAccessPolicy, error) {
	if err := ValidateAllowIPs(in.AllowIPs); err != nil {
		return nil, err
	}
	if in.MaxSessions < 0 {
		return nil, ErrInvalidSessionLimit
	}
	if err := ValidateAccessWindows(in.AllowedWindows); err != nil {
		return nil, err
	}
	if err := ValidateTimezone(in.Timezone); err != nil {
		return nil, err
	}
	if err := ValidateAccessExceptions(in.ExceptionDates); err != nil {
		return nil, err
	}
	normalized := normalizeAllowIPs(in.AllowIPs)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	windowsEncoded, err := json.Marshal(in.AllowedWindows)
	if err != nil {
		return nil, err
	}
	exceptionsEncoded, err := json.Marshal(in.ExceptionDates)
	if err != nil {
		return nil, err
	}

	row, err := s.Get(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &models.DeviceAccessPolicy{DeviceID: deviceID}
	}
	row.OwnerID = ownerID
	row.RemoteDisabled = in.RemoteDisabled
	row.AllowIPs = string(encoded)
	row.MaxSessions = in.MaxSessions
	row.AllowedWindows = string(windowsEncoded)
	row.Timezone = strings.TrimSpace(in.Timezone)
	row.ExceptionDates = string(exceptionsEncoded)
	row.ApprovalRequired = in.ApprovalRequired
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// RequiresApproval reports whether a device policy turns a matched access code
// into a pending owner decision. A nil policy never does.
func RequiresApproval(policy *models.DeviceAccessPolicy) bool {
	return policy != nil && policy.ApprovalRequired
}

// Evaluate decides whether a client at clientIP may proceed with session
// setup. A nil policy allows everything; the returned reason is meant to be
// shown to the device owner in the access log.
func EvaluateDeviceAccessPolicy(policy *models.DeviceAccessPolicy, clientIP string) (bool, string) {
	return EvaluateDeviceAccessPolicyAt(policy, clientIP, time.Now())
}

// EvaluateDeviceAccessPolicyAt is EvaluateDeviceAccessPolicy with an
// injectable clock, so the time-window rules can be tested deterministically.
func EvaluateDeviceAccessPolicyAt(policy *models.DeviceAccessPolicy, clientIP string, now time.Time) (bool, string) {
	if policy == nil {
		return true, ""
	}
	if policy.RemoteDisabled {
		return false, "remote access is disabled for this device"
	}
	entries := decodeAllowIPs(policy.AllowIPs)
	if len(entries) > 0 && !IPAllowed(entries, clientIP) {
		return false, "client IP is not in the device allow list"
	}
	windows := decodeAccessWindows(policy.AllowedWindows)
	exceptions := decodeAccessExceptions(policy.ExceptionDates)
	if len(windows) > 0 || len(exceptions) > 0 {
		loc := LocationForPolicy(policy.Timezone)
		if match, allowed, reason := AccessExceptionMatch(exceptions, loc, now); match {
			// A one-off exception replaces the weekly schedule for that date:
			// "deny" blocks the whole day, "allow" admits it all day.
			if !allowed {
				return false, reason
			}
		} else if len(windows) > 0 {
			if allowed, reason := EvaluateAccessWindowsAt(windows, loc, now); !allowed {
				return false, reason
			}
		}
	}
	return true, ""
}

// ValidateAllowIPs rejects entries that are neither a bare IP nor a CIDR
// block. An empty list is valid and means "any IP".
func ValidateAllowIPs(entries []string) error {
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			return fmt.Errorf("%w: empty entry", ErrInvalidAllowIP)
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("%w: %q is not a valid CIDR block", ErrInvalidAllowIP, entry)
			}
			continue
		}
		if net.ParseIP(entry) == nil {
			return fmt.Errorf("%w: %q is not a valid IP address", ErrInvalidAllowIP, entry)
		}
	}
	return nil
}

// IPAllowed reports whether clientIP matches any exact-IP or CIDR entry.
// An empty entry list allows everything; a client address the server cannot
// parse is rejected (fail closed) once a list exists.
func IPAllowed(entries []string, clientIP string) bool {
	if len(entries) == 0 {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		if parsed := net.ParseIP(entry); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}
	return false
}

// SessionLimitReached reports whether one more controller would exceed the
// device's concurrency cap. max <= 0 means unlimited.
func SessionLimitReached(max, active int) bool {
	if max <= 0 {
		return false
	}
	return active >= max
}

func decodeAllowIPs(encoded string) []string {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var entries []string
	if err := json.Unmarshal([]byte(encoded), &entries); err != nil {
		return nil
	}
	return entries
}

func decodeAccessWindows(encoded string) []AccessWindow {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var windows []AccessWindow
	if err := json.Unmarshal([]byte(encoded), &windows); err != nil {
		return nil
	}
	return windows
}

func decodeAccessExceptions(encoded string) []AccessExceptionDate {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var exceptions []AccessExceptionDate
	if err := json.Unmarshal([]byte(encoded), &exceptions); err != nil {
		return nil
	}
	return exceptions
}

func normalizeAllowIPs(entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, raw := range entries {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
