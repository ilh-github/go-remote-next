package service

import (
	"errors"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
)

// TestParseCalendarScheduleAcceptsTheDocumentedForms pins the DSL's grammar:
// three cadences, HH:MM on a 24-hour clock, a weekday as a name or 0..6.
func TestParseCalendarScheduleAcceptsTheDocumentedForms(t *testing.T) {
	cases := []struct {
		spec   string
		hour   int
		minute int
		weekly bool
		dow    time.Weekday
		dom    int
	}{
		{spec: "daily@05:30", hour: 5, minute: 30},
		{spec: "daily@00:00", hour: 0, minute: 0},
		{spec: "daily@23:59", hour: 23, minute: 59},
		{spec: "weekly@mon@09:00", hour: 9, weekly: true, dow: time.Monday},
		{spec: "weekly@SUN@00:15", hour: 0, minute: 15, weekly: true, dow: time.Sunday},
		{spec: "weekly@0@12:00", hour: 12, weekly: true, dow: time.Sunday},
		{spec: "weekly@6@12:00", hour: 12, weekly: true, dow: time.Saturday},
		{spec: "weekly@saturday@12:00", hour: 12, weekly: true, dow: time.Saturday},
		{spec: "monthly@1@02:00", hour: 2, dom: 1},
		{spec: "monthly@31@23:45", hour: 23, minute: 45, dom: 31},
		// Surrounding whitespace is tolerated: it comes from JSON and a form.
		{spec: " daily@06:00 ", hour: 6},
	}
	for _, tc := range cases {
		got, err := parseCalendarSchedule(tc.spec)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.spec, err)
		}
		if got.hour != tc.hour || got.minute != tc.minute || got.dom != tc.dom {
			t.Fatalf("parse %q = %+v, want hour=%d minute=%d dom=%d", tc.spec, got, tc.hour, tc.minute, tc.dom)
		}
		if tc.weekly && got.dow != tc.dow {
			t.Fatalf("parse %q dow = %v, want %v", tc.spec, got.dow, tc.dow)
		}
	}
}

// TestParseCalendarScheduleRejectsMalformed pins that every rejection is the
// sentinel the API maps to a 400, and that the message names the problem.
func TestParseCalendarScheduleRejectsMalformed(t *testing.T) {
	specs := []string{
		"",
		"daily",
		"hourly@10:00",
		"daily@24:00",
		"daily@10:60",
		"daily@1000",
		"daily@10:00@extra",
		"weekly@10:00",
		"weekly@funday@10:00",
		"weekly@7@10:00",
		"monthly@0@10:00",
		"monthly@32@10:00",
		"monthly@jan@10:00",
	}
	for _, spec := range specs {
		if _, err := parseCalendarSchedule(spec); !errors.Is(err, ErrInvalidScheduledTaskCalendar) {
			t.Fatalf("parse %q err = %v, want ErrInvalidScheduledTaskCalendar", spec, err)
		}
	}
}

// TestCalendarScheduleNextIsStrictlyAfter pins the "fire once per period, not
// once per tick" rule: an occurrence exactly at `after` is already spent, so
// next() must move on rather than return it again.
func TestCalendarScheduleNextIsStrictlyAfter(t *testing.T) {
	daily, err := parseCalendarSchedule("daily@09:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	before := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	if got := daily.next(before, time.UTC); !got.Equal(time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("daily next from 08:00 = %v, want 2026-10-16T09:00Z", got)
	}
	exact := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	if got := daily.next(exact, time.UTC); !got.Equal(time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("daily next at the occurrence = %v, want the next day", got)
	}
	// The month and the year have to roll over without help.
	monthEnd := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	if got := daily.next(monthEnd, time.UTC); !got.Equal(time.Date(2027, 1, 1, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("daily next across new year = %v, want 2027-01-01T09:00Z", got)
	}
}

// TestCalendarScheduleWeeklyAndMonthly covers the calendar arithmetic a cron
// expression would normally hide: weekdays, day-of-month clamping, and the
// leap-year case.
func TestCalendarScheduleWeeklyAndMonthly(t *testing.T) {
	// 2026-10-16 is a Friday.
	monday, err := parseCalendarSchedule("weekly@mon@09:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	friday := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	if got := monday.next(friday, time.UTC); !got.Equal(time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("weekly next from Friday = %v, want Monday 2026-10-19T09:00Z", got)
	}
	// A weekly schedule whose weekday is today, already spent, waits a week.
	fridaySchedule, err := parseCalendarSchedule("weekly@fri@09:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := fridaySchedule.next(friday, time.UTC); !got.Equal(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("weekly next from later the same day = %v, want the following Friday", got)
	}

	monthly31, err := parseCalendarSchedule("monthly@31@02:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// February cannot hold the 31st: clamp to the last day instead of skipping
	// the month, which is what the docs promise.
	jan31 := time.Date(2027, 1, 31, 3, 0, 0, 0, time.UTC)
	if got := monthly31.next(jan31, time.UTC); !got.Equal(time.Date(2027, 2, 28, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthly@31 from Jan 31 = %v, want 2027-02-28T02:00Z", got)
	}
	leapJan31 := time.Date(2028, 1, 31, 3, 0, 0, 0, time.UTC)
	if got := monthly31.next(leapJan31, time.UTC); !got.Equal(time.Date(2028, 2, 29, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthly@31 in a leap year = %v, want 2028-02-29T02:00Z", got)
	}
	// Days in a month that is shorter than 31 by two: April.
	mar31 := time.Date(2027, 3, 31, 3, 0, 0, 0, time.UTC)
	if got := monthly31.next(mar31, time.UTC); !got.Equal(time.Date(2027, 4, 30, 2, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthly@31 from Mar 31 = %v, want 2027-04-30T02:00Z", got)
	}
}

// TestCalendarScheduleHonoursTimezone pins the reason the timezone field
// exists: "05:30" means 05:30 where the operator lives, not 05:30 UTC.
func TestCalendarScheduleHonoursTimezone(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	daily, err := parseCalendarSchedule("daily@05:30")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 2026-10-15T22:00Z is 06:00 on the 16th in Shanghai, so the 16th's 05:30
	// has already gone: the next one is 05:30 on the 17th, i.e. 21:30 the same
	// UTC day.
	after := time.Date(2026, 10, 15, 22, 0, 0, 0, time.UTC)
	got := daily.next(after, shanghai)
	want := time.Date(2026, 10, 16, 21, 30, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("daily@05:30 Asia/Shanghai next = %v, want %v", got, want)
	}
	if hour, minute := got.In(shanghai).Hour(), got.In(shanghai).Minute(); hour != 5 || minute != 30 {
		t.Fatalf("local wall clock = %02d:%02d, want 05:30", hour, minute)
	}
}

// TestCalendarScheduleAcrossSpringForward pins what happens to a wall clock
// that DST deletes. US 2026-03-08 has no 02:30 in New York, and Go's own
// normalisation would put it at 01:30 - an hour early. The schedule must
// instead fire at the first real instant at or after 02:30, which is 03:00 EDT,
// and it must still be strictly after the previous occurrence.
func TestCalendarScheduleAcrossSpringForward(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	daily, err := parseCalendarSchedule("daily@02:30")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// 2026-03-07T08:00Z is 03:00 EST on the 7th: that day's 02:30 is spent.
	after := time.Date(2026, 3, 7, 8, 0, 0, 0, time.UTC)
	got := daily.next(after, newYork)
	if !got.After(after) {
		t.Fatalf("next occurrence %v is not after %v", got, after)
	}
	local := got.In(newYork)
	if local.Day() != 8 || local.Hour() != 3 || local.Minute() != 0 {
		t.Fatalf("spring-forward occurrence = %v, want 2026-03-08 03:00 local", local)
	}

	// The ambiguous hour of a fall-back day resolves to a real instant and does
	// not repeat or move backwards.
	afterFall := time.Date(2026, 11, 1, 0, 0, 0, 0, newYork)
	gotFall := daily.next(afterFall, newYork)
	if !gotFall.After(afterFall) {
		t.Fatalf("fall-back occurrence %v is not after %v", gotFall, afterFall)
	}
}

// TestScheduledTaskNextOccurrenceCoversBothShapes keeps the helper that the
// claim and the retry clamp share honest for interval, calendar and one-shot
// rows.
func TestScheduledTaskNextOccurrenceCoversBothShapes(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	interval := models.ScheduledTask{RepeatSeconds: 90}
	if got, err := scheduledTaskNextOccurrence(interval, now); err != nil || !got.Equal(now.Add(90*time.Second)) {
		t.Fatalf("interval next = %v (%v), want %v", got, err, now.Add(90*time.Second))
	}
	calendar := models.ScheduledTask{Schedule: "daily@09:00", Timezone: "UTC"}
	if got, err := scheduledTaskNextOccurrence(calendar, now); err != nil || !got.Equal(time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("calendar next = %v (%v), want 2026-10-16T09:00Z", got, err)
	}
	oneShot := models.ScheduledTask{}
	if scheduledTaskRecurs(oneShot) {
		t.Fatal("a one-shot row must not report a recurrence")
	}
	if !scheduledTaskRecurs(calendar) || !scheduledTaskRecurs(interval) {
		t.Fatal("interval and calendar rows must both report a recurrence")
	}
	// A row written by hand with a schedule nobody can parse has to fail
	// loudly: the claim parks such a row instead of looping on it.
	broken := models.ScheduledTask{Schedule: "fortnightly@mon@09:00"}
	if _, err := scheduledTaskNextOccurrence(broken, now); !errors.Is(err, ErrInvalidScheduledTaskCalendar) {
		t.Fatalf("broken schedule err = %v, want ErrInvalidScheduledTaskCalendar", err)
	}
}

// TestScheduledTaskLocationDefaultsToUTC pins that the zero value of the new
// column keeps meaning "UTC", so rows written before this feature still work.
func TestScheduledTaskLocationDefaultsToUTC(t *testing.T) {
	for _, value := range []string{"", "   "} {
		loc, err := scheduledTaskLocation(value)
		if err != nil {
			t.Fatalf("location %q: %v", value, err)
		}
		if loc != time.UTC {
			t.Fatalf("location %q = %v, want UTC", value, loc)
		}
	}
	if _, err := scheduledTaskLocation("Mars/Olympus"); !errors.Is(err, ErrInvalidScheduledTaskCalendar) {
		t.Fatalf("unknown zone err = %v, want ErrInvalidScheduledTaskCalendar", err)
	}
}
