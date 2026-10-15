package service

import (
	"errors"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func TestValidateAccessWindows(t *testing.T) {
	tests := []struct {
		name    string
		windows []AccessWindow
		wantErr bool
	}{
		{"empty means any time", nil, false},
		{"weekday office hours", []AccessWindow{{Days: []int{1, 2, 3, 4, 5}, Start: "09:00", End: "18:00"}}, false},
		{"every day when days omitted", []AccessWindow{{Start: "00:00", End: "23:59"}}, false},
		{"wrapping window", []AccessWindow{{Days: []int{5}, Start: "22:00", End: "06:00"}}, false},
		{"day out of range", []AccessWindow{{Days: []int{7}, Start: "09:00", End: "18:00"}}, true},
		{"negative day", []AccessWindow{{Days: []int{-1}, Start: "09:00", End: "18:00"}}, true},
		{"bad start", []AccessWindow{{Start: "9am", End: "18:00"}}, true},
		{"hour out of range", []AccessWindow{{Start: "25:00", End: "18:00"}}, true},
		{"missing end", []AccessWindow{{Start: "09:00"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAccessWindows(tt.windows)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAccessWindows(%v) err = %v, wantErr %v", tt.windows, err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidAccessWindow) {
				t.Fatalf("error should wrap ErrInvalidAccessWindow, got %v", err)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	for _, ok := range []string{"", "  ", "Asia/Shanghai", "UTC", "America/New_York"} {
		if err := ValidateTimezone(ok); err != nil {
			t.Errorf("ValidateTimezone(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"Mars/Phobos", "GMT+8", "Shanghai"} {
		if err := ValidateTimezone(bad); !errors.Is(err, ErrInvalidTimezone) {
			t.Errorf("ValidateTimezone(%q) = %v, want ErrInvalidTimezone", bad, err)
		}
	}
}

func TestEvaluateAccessWindowsAt(t *testing.T) {
	shanghai := mustLoad(t, "Asia/Shanghai")
	officeHours := []AccessWindow{{Days: []int{1, 2, 3, 4, 5}, Start: "09:00", End: "18:00"}}

	at := func(loc *time.Location, year int, month time.Month, day, hour, min int) time.Time {
		return time.Date(year, month, day, hour, min, 0, 0, loc)
	}

	tests := []struct {
		name    string
		windows []AccessWindow
		now     time.Time
		want    bool
	}{
		{"no windows allows anything", nil, at(shanghai, 2026, 10, 19, 3, 0), true},
		{"inside weekday window", officeHours, at(shanghai, 2026, 10, 19, 10, 0), true},
		{"before window", officeHours, at(shanghai, 2026, 10, 19, 8, 59), false},
		{"end is exclusive", officeHours, at(shanghai, 2026, 10, 19, 18, 0), false},
		{"weekend is excluded", officeHours, at(shanghai, 2026, 10, 17, 10, 0), false},
		{"same instant in another zone", officeHours, at(time.UTC, 2026, 10, 19, 3, 0), true}, // 11:00 上海
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := EvaluateAccessWindowsAt(tt.windows, shanghai, tt.now)
			if allowed != tt.want {
				t.Fatalf("allowed = %v (reason %q), want %v", allowed, reason, tt.want)
			}
			if !allowed && reason == "" {
				t.Fatal("denied window should explain itself")
			}
		})
	}
}

// A window that crosses midnight is anchored on the day it starts: Friday
// 22:00-06:00 covers Friday night *and* Saturday morning.
func TestEvaluateAccessWindowsAtWrapsMidnight(t *testing.T) {
	loc := mustLoad(t, "Asia/Shanghai")
	fridayNight := []AccessWindow{{Days: []int{5}, Start: "22:00", End: "06:00"}}

	cases := []struct {
		when time.Time
		want bool
	}{
		{time.Date(2026, 10, 23, 22, 30, 0, 0, loc), true},  // Friday 22:30
		{time.Date(2026, 10, 24, 5, 30, 0, 0, loc), true},   // Saturday 05:30
		{time.Date(2026, 10, 24, 7, 0, 0, 0, loc), false},   // Saturday 07:00
		{time.Date(2026, 10, 24, 22, 30, 0, 0, loc), false}, // Saturday night is a different start day
		{time.Date(2026, 10, 23, 21, 59, 0, 0, loc), false},
	}
	for _, c := range cases {
		allowed, _ := EvaluateAccessWindowsAt(fridayNight, loc, c.when)
		if allowed != c.want {
			t.Errorf("window at %s = %v, want %v", c.when.Format(time.RFC3339), allowed, c.want)
		}
	}
}

func TestEvaluateDeviceAccessPolicyAtHonoursWindows(t *testing.T) {
	loc := mustLoad(t, "Asia/Shanghai")
	policy := &models.DeviceAccessPolicy{
		AllowIPs:       `["127.0.0.0/8"]`,
		AllowedWindows: `[{"days":[1],"start":"09:00","end":"18:00"}]`,
		Timezone:       "Asia/Shanghai",
	}

	mondayInside := time.Date(2026, 10, 19, 10, 0, 0, 0, loc)
	if allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", mondayInside); !allowed {
		t.Fatalf("inside window should be allowed, got %q", reason)
	}

	mondayOutside := time.Date(2026, 10, 19, 20, 0, 0, 0, loc)
	allowed, reason := EvaluateDeviceAccessPolicyAt(policy, "127.0.0.1", mondayOutside)
	if allowed {
		t.Fatal("outside window must be denied")
	}
	if reason != "outside the allowed access window" {
		t.Fatalf("unexpected reason %q", reason)
	}

	// The IP rule still wins when both would deny: it is checked first.
	if _, reason := EvaluateDeviceAccessPolicyAt(policy, "10.0.0.1", mondayInside); reason != "client IP is not in the device allow list" {
		t.Fatalf("expected the IP reason first, got %q", reason)
	}
}
