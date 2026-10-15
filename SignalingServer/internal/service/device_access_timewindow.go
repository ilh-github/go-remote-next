package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidAccessWindow means an allowed-access window is malformed (bad day,
// bad "HH:MM" time, or no start/end).
var ErrInvalidAccessWindow = errors.New("access windows need days 0-6 and HH:MM start/end times")

// ErrInvalidTimezone means the IANA timezone could not be loaded.
var ErrInvalidTimezone = errors.New("timezone must be an IANA name like Asia/Shanghai")

// AccessWindow is one recurring window in which remote access is allowed.
//
// Days use Go's weekday order, 0 = Sunday … 6 = Saturday; an empty list means
// every day. Start and End are wall-clock "HH:MM" in the policy's timezone.
// Start > End means the window wraps past midnight, and the day filter then
// applies to the day the window *starts* (so `{days:[5],start:"22:00",end:"06:00"}`
// covers Friday 22:00 through Saturday 06:00).
type AccessWindow struct {
	Days  []int  `json:"days,omitempty"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// ValidateAccessWindows rejects malformed windows. An empty list is valid and
// means "no time restriction".
func ValidateAccessWindows(windows []AccessWindow) error {
	for i, w := range windows {
		for _, d := range w.Days {
			if d < 0 || d > 6 {
				return fmt.Errorf("%w: window %d has day %d (want 0-6)", ErrInvalidAccessWindow, i, d)
			}
		}
		if _, err := parseClock(w.Start); err != nil {
			return fmt.Errorf("%w: window %d start %q", ErrInvalidAccessWindow, i, w.Start)
		}
		if _, err := parseClock(w.End); err != nil {
			return fmt.Errorf("%w: window %d end %q", ErrInvalidAccessWindow, i, w.End)
		}
	}
	return nil
}

// ValidateTimezone accepts an empty string (meaning the server's local zone)
// or an IANA location name.
func ValidateTimezone(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidTimezone, name)
	}
	return nil
}

// EvaluateAccessWindowsAt reports whether now falls inside any window. An
// empty list allows everything; a nil location falls back to UTC.
func EvaluateAccessWindowsAt(windows []AccessWindow, loc *time.Location, now time.Time) (bool, string) {
	if len(windows) == 0 {
		return true, ""
	}
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	for _, w := range windows {
		if windowAllows(w, local) {
			return true, ""
		}
	}
	return false, "outside the allowed access window"
}

// LocationForPolicy resolves the policy timezone, falling back to the
// server's local zone when unset and to UTC when an unknown zone somehow
// slipped past validation.
func LocationForPolicy(name string) *time.Location {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return time.UTC
	}
	return loc
}

func windowAllows(w AccessWindow, local time.Time) bool {
	start, err := parseClock(w.Start)
	if err != nil {
		return false
	}
	end, err := parseClock(w.End)
	if err != nil {
		return false
	}
	today := int(local.Weekday())
	yesterday := (today + 6) % 7
	cur := local.Hour()*60 + local.Minute()

	switch {
	case start == end:
		// Same start and end reads as "all day" for the selected days.
		return dayMatches(w.Days, today)
	case start < end:
		return dayMatches(w.Days, today) && cur >= start && cur < end
	default:
		// Wraps past midnight.
		if dayMatches(w.Days, today) && cur >= start {
			return true
		}
		return dayMatches(w.Days, yesterday) && cur < end
	}
}

func dayMatches(days []int, day int) bool {
	if len(days) == 0 {
		return true
	}
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

// parseClock turns "HH:MM" into minutes since midnight.
func parseClock(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	t, err := time.Parse("15:04", trimmed)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}
