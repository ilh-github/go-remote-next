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

// ErrInvalidAccessException means an exception date is malformed (bad date or
// range, unknown mode/repeat) or two exceptions with different modes overlap.
var ErrInvalidAccessException = errors.New("access exception dates need YYYY-MM-DD date/end_date, mode \"allow\" or \"deny\", optional repeat \"yearly\", and no overlapping allow/deny entries")

// Exception-date modes.
const (
	AccessExceptionModeAllow = "allow"
	AccessExceptionModeDeny  = "deny"
)

// AccessExceptionRepeatYearly makes an exception recur every year on the same
// month-day(s), so "Spring Festival" or "New Year" is one entry instead of one
// per year.
const AccessExceptionRepeatYearly = "yearly"

// AccessExceptionDate is a one-off (or annually recurring) override, evaluated
// in the policy's timezone. "deny" blocks access for the covered day(s)
// (holidays, maintenance); "allow" permits them and overrides the weekly
// windows (a one-off work day). EndDate is an inclusive range end and defaults
// to Date; Repeat:"yearly" matches the same month-day(s) every year and may
// wrap the new year (e.g. 12-30 through 01-02). Either way only the *time* rule
// is decided here - remote_disabled and the IP allow list are still enforced.
type AccessExceptionDate struct {
	Date    string `json:"date"`
	EndDate string `json:"end_date,omitempty"`
	Mode    string `json:"mode"`
	Repeat  string `json:"repeat,omitempty"`
}

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

// ValidateAccessExceptions rejects malformed exception entries and refuses to
// accept an allow and a deny entry that cover the same day, so a policy can
// never be ambiguous about whether a given day is allowed or denied. Entries
// with the same mode may overlap (harmless duplication).
func ValidateAccessExceptions(exceptions []AccessExceptionDate) error {
	for i, e := range exceptions {
		start, err := parseExceptionDate(e.Date)
		if err != nil {
			return fmt.Errorf("%w: exception %d date %q", ErrInvalidAccessException, i, e.Date)
		}
		end := start
		if strings.TrimSpace(e.EndDate) != "" {
			end, err = parseExceptionDate(e.EndDate)
			if err != nil {
				return fmt.Errorf("%w: exception %d end_date %q", ErrInvalidAccessException, i, e.EndDate)
			}
		}
		if mode := strings.ToLower(strings.TrimSpace(e.Mode)); mode != AccessExceptionModeAllow && mode != AccessExceptionModeDeny {
			return fmt.Errorf("%w: exception %d mode %q", ErrInvalidAccessException, i, e.Mode)
		}
		repeat := strings.ToLower(strings.TrimSpace(e.Repeat))
		if repeat != "" && repeat != AccessExceptionRepeatYearly {
			return fmt.Errorf("%w: exception %d repeat %q", ErrInvalidAccessException, i, e.Repeat)
		}
		// A yearly exception may wrap the new year, so only a non-recurring
		// range has to be ordered.
		if repeat == "" && end.Before(start) {
			return fmt.Errorf("%w: exception %d end_date %s is before date %s",
				ErrInvalidAccessException, i, e.EndDate, e.Date)
		}
	}

	// Pairwise conflict check: an allow and a deny must not cover any shared day.
	for i := 0; i < len(exceptions); i++ {
		for j := i + 1; j < len(exceptions); j++ {
			if sameMode(exceptions[i].Mode, exceptions[j].Mode) {
				continue
			}
			if exceptionsOverlap(exceptions[i], exceptions[j]) {
				return fmt.Errorf("%w: exceptions %d and %d both cover a day with different modes",
					ErrInvalidAccessException, i, j)
			}
		}
	}
	return nil
}

// AccessExceptionMatch reports whether the policy's local calendar date for now
// is covered by an exception, and if so whether it allows access. matched=false
// means "no exception today", so the caller should fall back to the weekly
// windows.
func AccessExceptionMatch(exceptions []AccessExceptionDate, loc *time.Location, now time.Time) (matched bool, allowed bool, reason string) {
	if len(exceptions) == 0 {
		return false, false, ""
	}
	if loc == nil {
		loc = time.UTC
	}
	today := now.In(loc)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	// Deny is scanned first as a defensive tie-break; validation already
	// rejects allow/deny overlaps, but a malformed stored policy should not
	// silently become an access grant.
	for _, e := range exceptions {
		if sameMode(e.Mode, AccessExceptionModeDeny) && exceptionCovers(e, day) {
			return true, false, "today is blocked by an access exception date"
		}
	}
	for _, e := range exceptions {
		if sameMode(e.Mode, AccessExceptionModeAllow) && exceptionCovers(e, day) {
			return true, true, ""
		}
	}
	return false, false, ""
}

func sameMode(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// exceptionCovers reports whether the (midnight, UTC-normalised) day falls in
// the exception, honouring yearly recurrence and new-year wrapping.
func exceptionCovers(e AccessExceptionDate, day time.Time) bool {
	start, err := parseExceptionDate(e.Date)
	if err != nil {
		return false
	}
	end := start
	if strings.TrimSpace(e.EndDate) != "" {
		if end, err = parseExceptionDate(e.EndDate); err != nil {
			return false
		}
	}
	if strings.EqualFold(strings.TrimSpace(e.Repeat), AccessExceptionRepeatYearly) {
		return inMonthDayRange(monthDayKey(start), monthDayKey(end), monthDayKey(day))
	}
	return !day.Before(start) && !day.After(end)
}

// exceptionsOverlap reports whether two entries cover at least one shared day.
func exceptionsOverlap(a, b AccessExceptionDate) bool {
	aStart, errA1 := parseExceptionDate(a.Date)
	bStart, errB1 := parseExceptionDate(b.Date)
	if errA1 != nil || errB1 != nil {
		return false
	}
	aEnd := aStart
	if strings.TrimSpace(a.EndDate) != "" {
		if parsed, err := parseExceptionDate(a.EndDate); err == nil {
			aEnd = parsed
		}
	}
	bEnd := bStart
	if strings.TrimSpace(b.EndDate) != "" {
		if parsed, err := parseExceptionDate(b.EndDate); err == nil {
			bEnd = parsed
		}
	}
	aYearly := strings.EqualFold(strings.TrimSpace(a.Repeat), AccessExceptionRepeatYearly)
	bYearly := strings.EqualFold(strings.TrimSpace(b.Repeat), AccessExceptionRepeatYearly)

	switch {
	case aYearly && bYearly:
		return monthDayRangesOverlap(monthDayKey(aStart), monthDayKey(aEnd), monthDayKey(bStart), monthDayKey(bEnd))
	case aYearly:
		return inMonthDayRange(monthDayKey(aStart), monthDayKey(aEnd), monthDayKey(bStart)) ||
			inMonthDayRange(monthDayKey(aStart), monthDayKey(aEnd), monthDayKey(bEnd))
	case bYearly:
		return inMonthDayRange(monthDayKey(bStart), monthDayKey(bEnd), monthDayKey(aStart)) ||
			inMonthDayRange(monthDayKey(bStart), monthDayKey(bEnd), monthDayKey(aEnd))
	default:
		return !aStart.After(bEnd) && !bStart.After(aEnd)
	}
}

// parseExceptionDate parses YYYY-MM-DD into a date-only UTC time so range
// comparisons never depend on a clock or a location.
func parseExceptionDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

// monthDayKey returns month*100+day, a sortable stand-in for "day of year".
func monthDayKey(t time.Time) int {
	return int(t.Month())*100 + t.Day()
}

// inMonthDayRange reports membership in a month-day range that may wrap the new
// year (start > end).
func inMonthDayRange(start, end, md int) bool {
	if start <= end {
		return md >= start && md <= end
	}
	return md >= start || md <= end
}

// monthDayRangesOverlap reports whether two month-day ranges (each possibly
// wrapping the new year) share at least one day.
//
// Two arcs on a circle intersect exactly when one arc's start lies in the
// other arc, which avoids any cyclic-interval case analysis.
func monthDayRangesOverlap(aStart, aEnd, bStart, bEnd int) bool {
	return inMonthDayRange(bStart, bEnd, aStart) || inMonthDayRange(aStart, aEnd, bStart)
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
