package service

import (
	"errors"
	"testing"

	"quickdesk/signaling/internal/models"
)

func TestValidateAllowIPs(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		wantErr bool
	}{
		{"empty list allows anything", nil, false},
		{"exact ipv4", []string{"10.211.55.4"}, false},
		{"ipv4 cidr", []string{"10.211.55.0/24"}, false},
		{"ipv6 and cidr", []string{"::1", "2001:db8::/32"}, false},
		{"trimmed entry", []string{"  10.0.0.1  "}, false},
		{"hostname is not enough", []string{"example.com"}, true},
		{"bad cidr", []string{"10.0.0.0/99"}, true},
		{"empty entry", []string{"  "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAllowIPs(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAllowIPs(%v) err = %v, wantErr %v", tt.entries, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidAllowIP) {
				t.Fatalf("error should wrap ErrInvalidAllowIP, got %v", err)
			}
		})
	}
}

func TestIPAllowed(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		ip      string
		want    bool
	}{
		{"empty list allows", nil, "127.0.0.1", true},
		{"exact match", []string{"127.0.0.1"}, "127.0.0.1", true},
		{"exact mismatch", []string{"127.0.0.1"}, "10.0.0.1", false},
		{"cidr match", []string{"10.211.55.0/24"}, "10.211.55.4", true},
		{"cidr mismatch", []string{"10.211.55.0/24"}, "10.211.56.4", false},
		{"one of many matches", []string{"192.168.0.0/16", "10.211.55.4"}, "10.211.55.4", true},
		{"unparsable client ip fails closed", []string{"10.0.0.0/8"}, "not-an-ip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPAllowed(tt.entries, tt.ip); got != tt.want {
				t.Fatalf("IPAllowed(%v, %q) = %v, want %v", tt.entries, tt.ip, got, tt.want)
			}
		})
	}
}

func TestEvaluateDeviceAccessPolicy(t *testing.T) {
	// No policy row at all: every existing device keeps working.
	if allowed, reason := EvaluateDeviceAccessPolicy(nil, "127.0.0.1"); !allowed || reason != "" {
		t.Fatalf("nil policy should allow, got allowed=%v reason=%q", allowed, reason)
	}

	disabled := &models.DeviceAccessPolicy{RemoteDisabled: true, AllowIPs: `["127.0.0.1"]`}
	if allowed, reason := EvaluateDeviceAccessPolicy(disabled, "127.0.0.1"); allowed {
		t.Fatal("remote_disabled must block even an allow-listed IP")
	} else if reason == "" {
		t.Fatal("expected a reason for the blocked attempt")
	}

	allowListed := &models.DeviceAccessPolicy{AllowIPs: `["10.211.55.0/24"]`}
	if allowed, _ := EvaluateDeviceAccessPolicy(allowListed, "10.211.55.4"); !allowed {
		t.Fatal("client inside the allow list should pass")
	}
	if allowed, reason := EvaluateDeviceAccessPolicy(allowListed, "127.0.0.1"); allowed {
		t.Fatal("client outside the allow list must be blocked")
	} else if reason == "" {
		t.Fatal("expected a reason for the blocked attempt")
	}

	emptyList := &models.DeviceAccessPolicy{AllowIPs: `[]`}
	if allowed, _ := EvaluateDeviceAccessPolicy(emptyList, "8.8.8.8"); !allowed {
		t.Fatal("an empty allow list means any IP")
	}
}
