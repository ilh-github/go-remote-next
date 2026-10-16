package service

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// countingAction records how many times it ran, so a test can tell "the claim
// handed this task to exactly one worker" from "it ran twice".
type countingAction struct {
	mu    sync.Mutex
	runs  int
	tasks []uint
	err   error
}

func (a *countingAction) Run(_ context.Context, task models.ScheduledTask) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.runs++
	a.tasks = append(a.tasks, task.ID)
	return a.err
}

func (a *countingAction) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runs
}

// newScheduledTaskTestDB gives the scheduler tests a real Postgres, because the
// claim is `FOR UPDATE SKIP LOCKED`.
func newScheduledTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the scheduler tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.ScheduledTask{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE scheduled_tasks").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func seedScheduledTask(t *testing.T, db *gorm.DB, row models.ScheduledTask) models.ScheduledTask {
	t.Helper()
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return row
}

func loadScheduledTask(t *testing.T, db *gorm.DB, id uint) models.ScheduledTask {
	t.Helper()
	var row models.ScheduledTask
	if err := db.First(&row, id).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	return row
}

func TestScheduledTaskCreateValidation(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()
	future := time.Now().Add(time.Hour)

	if _, err := svc.Create(ctx, 1, "nope", future, 0); !errors.Is(err, ErrInvalidScheduledTaskKind) {
		t.Fatalf("unknown kind err = %v, want ErrInvalidScheduledTaskKind", err)
	}
	if _, err := svc.Create(ctx, 1, "probe", time.Now().Add(-time.Minute), 0); !errors.Is(err, ErrInvalidScheduledTaskSchedule) {
		t.Fatalf("past run_at err = %v, want ErrInvalidScheduledTaskSchedule", err)
	}
	if _, err := svc.Create(ctx, 1, "probe", future, 30); !errors.Is(err, ErrInvalidScheduledTaskSchedule) {
		t.Fatalf("repeat below the minimum err = %v, want ErrInvalidScheduledTaskSchedule", err)
	}
	row, err := svc.Create(ctx, 1, "probe", future, 60)
	if err != nil {
		t.Fatalf("valid create: %v", err)
	}
	if !row.Enabled || row.RepeatSeconds != 60 {
		t.Fatalf("created row = %+v, want an enabled 60s repeat", row)
	}
}

// A one-shot task fires once, records what happened, and does not stay armed.
func TestScheduledTaskRunDueFiresOnceAndRecordsOutcome(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	// Seeded already due: Create refuses a past run_at on purpose, so a test
	// that wants a due task writes the row directly.
	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "probe", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
	})

	ran, err := svc.RunDue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("run due: %v", err)
	}
	if ran != 1 || action.count() != 1 {
		t.Fatalf("ran=%d action.count=%d, want 1 and 1", ran, action.count())
	}
	after := loadScheduledTask(t, db, task.ID)
	if after.LastStatus != ScheduledTaskStatusOK || after.LastRunAt == nil {
		t.Fatalf("outcome = %+v, want ok with last_run_at set", after)
	}
	if after.Enabled {
		t.Fatal("a one-shot task must be disabled after it runs")
	}

	// Second pass: nothing due, nothing runs.
	if ran, err = svc.RunDue(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("second run due: %v", err)
	}
	if ran != 0 || action.count() != 1 {
		t.Fatalf("second pass ran %d (action count %d), want 0 and 1", ran, action.count())
	}
}

// The claim is the whole safety story: with several workers racing, each due
// task must be handed to exactly one of them.
func TestScheduledTaskClaimIsExclusiveAcrossWorkers(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	const rows = 60
	for i := 0; i < rows; i++ {
		seedScheduledTask(t, db, models.ScheduledTask{
			UserID: 7, Kind: "probe", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
		})
	}

	var wg sync.WaitGroup
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				n, err := svc.RunDue(ctx, time.Now().UTC())
				if err != nil || n == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	if action.count() != rows {
		t.Fatalf("action ran %d times for %d tasks; the claim handed some out twice", action.count(), rows)
	}
	var stillEnabled int64
	if err := db.Model(&models.ScheduledTask{}).Where("enabled = TRUE").Count(&stillEnabled).Error; err != nil {
		t.Fatalf("count enabled: %v", err)
	}
	if stillEnabled != 0 {
		t.Fatalf("%d one-shot tasks are still enabled", stillEnabled)
	}
}

// A recurring task reschedules from now rather than replaying every period the
// worker was down for, and stays armed.
func TestScheduledTaskRecurringReschedulesFromNow(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	action := &countingAction{}
	svc := NewScheduledTaskService(db).RegisterAction("probe", action)
	ctx := context.Background()

	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "probe",
		// Ten periods in the past: the scheduler must fire once, not ten times.
		NextRunAt:     time.Now().Add(-10 * time.Minute),
		RepeatSeconds: 60,
		Enabled:       true,
	})
	now := time.Now().UTC()
	if ran, err := svc.RunDue(ctx, now); err != nil || ran != 1 {
		t.Fatalf("run due = %d, %v; want 1, nil", ran, err)
	}
	after := loadScheduledTask(t, db, task.ID)
	if !after.Enabled {
		t.Fatal("a recurring task must stay enabled")
	}
	if after.NextRunAt.Before(now.Add(30 * time.Second)) {
		t.Fatalf("next_run_at = %s, want it rescheduled roughly one period from now", after.NextRunAt)
	}
	if action.count() != 1 {
		t.Fatalf("action ran %d times, want exactly 1 (missed periods are skipped, not replayed)", action.count())
	}
}

// A failing action is recorded, not retried in place: the schedule has already
// moved on, so the operator reads last_error instead of watching a hot loop.
func TestScheduledTaskFailureIsRecorded(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("boom", &countingAction{err: errors.New("redis is down")})
	ctx := context.Background()

	task := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 7, Kind: "boom", NextRunAt: time.Now().Add(-time.Minute), Enabled: true,
	})
	if _, err := svc.RunDue(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("run due: %v", err)
	}
	after := loadScheduledTask(t, db, task.ID)
	if after.LastStatus != ScheduledTaskStatusFailed {
		t.Fatalf("last_status = %q, want failed", after.LastStatus)
	}
	if after.LastError != "redis is down" {
		t.Fatalf("last_error = %q, want the action's message", after.LastError)
	}
}

// Tasks are owner-scoped: another user's id is invisible, not deletable.
func TestScheduledTaskListAndDeleteAreOwnerScoped(t *testing.T) {
	db := newScheduledTaskTestDB(t)
	svc := NewScheduledTaskService(db).RegisterAction("probe", &countingAction{})
	ctx := context.Background()

	mine := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 1, Kind: "probe", NextRunAt: time.Now().Add(time.Hour), Enabled: true,
	})
	theirs := seedScheduledTask(t, db, models.ScheduledTask{
		UserID: 2, Kind: "probe", NextRunAt: time.Now().Add(time.Hour), Enabled: true,
	})

	rows, err := svc.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != mine.ID {
		t.Fatalf("list for user 1 returned %+v, want only their own task", rows)
	}
	if _, err := svc.Get(ctx, 1, theirs.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross-user get err = %v, want ErrRecordNotFound", err)
	}
	if err := svc.Delete(ctx, 1, theirs.ID); err != nil {
		t.Fatalf("cross-user delete: %v", err)
	}
	if after := loadScheduledTask(t, db, theirs.ID); after.ID != theirs.ID {
		t.Fatal("another user's task was deleted")
	}
}
