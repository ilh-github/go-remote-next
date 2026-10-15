package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// ErrInvalidAllowIP means one of the allow-list entries is neither an IP nor
// a CIDR block.
var ErrInvalidAllowIP = errors.New("allow list entries must be IP addresses or CIDR blocks")

// DeviceAccessPolicyService stores and evaluates per-device access policy.
type DeviceAccessPolicyService struct {
	db *gorm.DB
}

func NewDeviceAccessPolicyService(db *gorm.DB) *DeviceAccessPolicyService {
	return &DeviceAccessPolicyService{db: db}
}

// UpsertDeviceAccessPolicyInput is the validated body of the policy endpoint.
type UpsertDeviceAccessPolicyInput struct {
	RemoteDisabled bool
	AllowIPs       []string
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
	normalized := normalizeAllowIPs(in.AllowIPs)
	encoded, err := json.Marshal(normalized)
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
	if err := s.db.WithContext(ctx).Save(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Evaluate decides whether a client at clientIP may proceed with session
// setup. A nil policy allows everything; the returned reason is meant to be
// shown to the device owner in the access log.
func EvaluateDeviceAccessPolicy(policy *models.DeviceAccessPolicy, clientIP string) (bool, string) {
	if policy == nil {
		return true, ""
	}
	if policy.RemoteDisabled {
		return false, "remote access is disabled for this device"
	}
	entries := decodeAllowIPs(policy.AllowIPs)
	if len(entries) == 0 {
		return true, ""
	}
	if !IPAllowed(entries, clientIP) {
		return false, "client IP is not in the device allow list"
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

func normalizeAllowIPs(entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, raw := range entries {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
