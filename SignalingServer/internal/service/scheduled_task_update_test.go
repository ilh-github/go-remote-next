package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// A task that is already scheduled can only be corrected by deleting and
// recreating it, which loses nothing today but does mean the owner cannot fix a
// mistake: a wrong retry budget or a wrong timezone means starting over. Update
// exists so the row keeps its id and its outcome history.
func TestScheduledTaskUpdateChangesScheduleAndBudget(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour).UTC()

	row, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: future, MaxAttempts: 2})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	newRunAt := future.Add(3 * time.Hour)
	updated, err := svc.Update(ctx, 1, row.ID, ScheduledTaskSpec{
		Kind:          "probe",
		RunAt:         newRunAt,
		RepeatSeconds: 120,
		MaxAttempts:   5,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ID != row.ID {
		t.Fatalf("update returned id %d, want %d", updated.ID, row.ID)
	}
	// Truncated to the microsecond Postgres stores, because the column has no
	// nanosecond field: a time.Time built in Go carries them, and comparing the
	// two with Equal fails on the digits the database never had.
	if !updated.NextRunAt.Equal(newRunAt.Truncate(time.Microsecond)) {
		t.Fatalf("next_run_at = %s, want %s", updated.NextRunAt, newRunAt)
	}
	if updated.RepeatSeconds != 120 || updated.MaxAttempts != 5 {
		t.Fatalf("updated row = %+v, want repeat 120 and budget 5", updated)
	}

	stored := loadScheduledTask(t, db, row.ID)
	if !stored.NextRunAt.Equal(newRunAt.Truncate(time.Microsecond)) {
		t.Fatalf("stored row = %+v, want the updated schedule and budget", stored)
	}
	if stored.RepeatSeconds != 120 || stored.MaxAttempts != 5 {
		t.Fatalf("stored row = %+v, want the updated schedule and budget", stored)
	}
}

// An edit must not silently clear the outcome the row already recorded: the
// owner needs to see that the task failed before they fixed it.
func TestScheduledTaskUpdateKeepsOutcomeAndReenablesARunTask(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()

	row, err := svc.Create(ctx, 1, ScheduledTaskSpec{
		Kind: "probe", RunAt: time.Now().Add(time.Hour), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	// The state a one-shot is left in after it ran: disabled, with a recorded
	// outcome. An owner editing such a row is fixing a mistake, not cancelling.
	if err := db.Model(&models.ScheduledTask{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"enabled":     false,
		"last_status": ScheduledTaskStatusFailed,
		"last_error":  "boom",
		"attempts":    1,
	}).Error; err != nil {
		t.Fatalf("seed outcome: %v", err)
	}

	if _, err := svc.Update(ctx, 1, row.ID, ScheduledTaskSpec{
		Kind:       "probe",
		RunAt:      time.Now().Add(2 * time.Hour),
		MaxAttempts: 3,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	stored := loadScheduledTask(t, db, row.ID)
	if !stored.Enabled {
		t.Fatal("updating a task that already ran left it disabled, so the new schedule never fires")
	}
	if stored.LastStatus != ScheduledTaskStatusFailed || stored.LastError != "boom" {
		t.Fatalf("stored outcome = %+v, want the recorded failure preserved", stored)
	}
}

// Another user's id must be a 404, exactly like get and delete: the edit is a
// write, so a leaked id must not be enough to reach somebody else's schedule.
//
// The assertion is on the stored row, not only on the error. The first version
// of this test checked the returned error alone and stayed green when the
// write's owner scope was removed: the edit landed on the other user's row and
// only the owner-scoped read that followed reported not-found. An error the
// caller can see is not the same as a write that did not happen.
func TestScheduledTaskUpdateIsOwnerScoped(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	originalRunAt := time.Now().Add(time.Hour).UTC()

	theirs := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 2, Kind: "probe", NextRunAt: originalRunAt, Enabled: true, MaxAttempts: 1,
	})
	_, err := svc.Update(ctx, 1, theirs.ID, ScheduledTaskSpec{
		Kind:       "probe",
		RunAt:      time.Now().Add(2 * time.Hour),
		MaxAttempts: 1,
	})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user update err = %v, want ErrRecordNotFound", err)
	}
	stored := loadScheduledTask(t, db, theirs.ID)
	if !stored.NextRunAt.Equal(originalRunAt.Truncate(time.Microsecond)) {
		t.Fatalf("another user's next_run_at moved to %s, want %s untouched",
			stored.NextRunAt, originalRunAt)
	}
	if stored.MaxAttempts != 1 {
		t.Fatalf("another user's max_attempts = %d, want 1 untouched", stored.MaxAttempts)
	}
}

// The same rules as create, because an edit is a create: a task saved with an
// unusable schedule is one the worker will park on its first tick.
func TestScheduledTaskUpdateValidatesTheSchedule(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour)

	row, err := svc.Create(ctx, 1, ScheduledTaskSpec{Kind: "probe", RunAt: future})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	for _, tc := range []struct {
		name string
		spec ScheduledTaskSpec
		want error
	}{
		{"unknown kind", ScheduledTaskSpec{Kind: "nope", RunAt: future}, ErrInvalidScheduledTaskKind},
		{"past run_at", ScheduledTaskSpec{Kind: "probe", RunAt: time.Now().Add(-time.Minute)}, ErrInvalidScheduledTaskSchedule},
		{"budget too high", ScheduledTaskSpec{Kind: "probe", RunAt: future, MaxAttempts: 11}, ErrInvalidScheduledTaskMaxAttempts},
		{"run_at with schedule", ScheduledTaskSpec{Kind: "probe", RunAt: future, Schedule: "daily@09:00"}, ErrInvalidScheduledTaskCalendar},
		{"skip_holidays without schedule", ScheduledTaskSpec{Kind: "probe", RunAt: future, SkipHolidays: true}, ErrInvalidScheduledTaskCalendar},
	} {
		if _, err := svc.Update(ctx, 1, row.ID, tc.spec); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

// A calendar edit recomputes the next occurrence from the new schedule rather
// than carrying the old instant over, so the timezone and the cadence the owner
// just chose are the ones that fire.
func TestScheduledTaskUpdateRecomputesACalendarOccurrence(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()

	row, err := svc.Create(ctx, 1, ScheduledTaskSpec{
		Kind: "probe", RunAt: time.Now().Add(time.Hour), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	updated, err := svc.Update(ctx, 1, row.ID, ScheduledTaskSpec{
		Kind:          "probe",
		Schedule:      "daily@09:05",
		Timezone:      "Asia/Shanghai",
		SkipHolidays:  true,
		MaxAttempts:   2,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Schedule != "daily@09:05" || updated.Timezone != "Asia/Shanghai" || !updated.SkipHolidays {
		t.Fatalf("updated row = %+v, want the calendar fields stored", updated)
	}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	// 09:05 Asia/Shanghai is 01:05 UTC, and the stored instant has to be the UTC
	// one: the worker compares against UTC, so a row that kept the wall clock
	// would fire eight hours late.
	want := time.Date(updated.NextRunAt.Year(), updated.NextRunAt.Month(), updated.NextRunAt.Day(),
		9, 5, 0, 0, shanghai).UTC()
	if !updated.NextRunAt.Equal(want.Truncate(time.Microsecond)) {
		t.Fatalf("next_run_at = %s, want %s (09:05 Asia/Shanghai in UTC)",
			updated.NextRunAt, want)
	}
}
