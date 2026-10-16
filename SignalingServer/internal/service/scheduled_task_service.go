package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// Scheduled-task outcome values written to ScheduledTask.LastStatus.
const (
	ScheduledTaskStatusOK     = "ok"
	ScheduledTaskStatusFailed = "failed"
)

// Schedule bounds. A repeat interval below a minute would let one task occupy
// the worker, and a year is already longer than any maintenance cadence worth
// configuring.
const (
	MinScheduledTaskRepeatSeconds = 60
	MaxScheduledTaskRepeatSeconds = 365 * 24 * 60 * 60
	// scheduledTaskBatch bounds one worker pass.
	scheduledTaskBatch = 100
)

var (
	// ErrInvalidScheduledTaskKind means the kind has no registered action, so
	// scheduling it would create a row nothing can ever run.
	ErrInvalidScheduledTaskKind = errors.New("unknown scheduled task kind")
	// ErrInvalidScheduledTaskSchedule covers a run_at that is absent, in the
	// past, or a repeat interval outside the accepted range.
	ErrInvalidScheduledTaskSchedule = errors.New("scheduled task needs a future run_at and repeat_seconds of 0 or 60..31536000")
)

// ScheduledTaskAction is the server-side work one kind performs. Actions must
// be safe to run unattended and without a request context: the worker calls
// them with the task's owner as the only identity.
type ScheduledTaskAction interface {
	Run(ctx context.Context, task models.ScheduledTask) error
}

// ScheduledTaskService owns owner-scheduled control-plane actions: the row,
// the due-claim, and the execution. It deliberately starts with kinds whose
// whole effect lives in the control plane, so a task cannot silently depend on
// a client that is not there.
type ScheduledTaskService struct {
	db      *gorm.DB
	actions map[string]ScheduledTaskAction
	audit   *AuditService
}

func NewScheduledTaskService(db *gorm.DB) *ScheduledTaskService {
	return &ScheduledTaskService{db: db, actions: map[string]ScheduledTaskAction{}}
}

// WithAudit records one audit row per run. Optional: a deployment without an
// audit service still gets the task's own last_status/last_error.
func (s *ScheduledTaskService) WithAudit(a *AuditService) *ScheduledTaskService {
	s.audit = a
	return s
}

// RegisterAction wires a kind to its work. Called at startup so the API's
// supported-kind list and the worker's dispatch table cannot disagree.
func (s *ScheduledTaskService) RegisterAction(kind string, action ScheduledTaskAction) *ScheduledTaskService {
	s.actions[strings.TrimSpace(kind)] = action
	return s
}

// SupportedKinds lists the registered kinds in a stable order.
func (s *ScheduledTaskService) SupportedKinds() []string {
	out := make([]string, 0, len(s.actions))
	for kind := range s.actions {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}

// Create schedules one action for the user. The kind has to be registered and
// run_at has to be in the future: a task that is already due is almost always a
// timezone or clock mistake, and the worker would otherwise run it before the
// caller saw the response.
func (s *ScheduledTaskService) Create(
	ctx context.Context,
	userID uint,
	kind string,
	runAt time.Time,
	repeatSeconds int,
) (*models.ScheduledTask, error) {
	kind = strings.TrimSpace(kind)
	if _, ok := s.actions[kind]; !ok {
		return nil, fmt.Errorf("%w: %q (supported: %s)", ErrInvalidScheduledTaskKind, kind,
			strings.Join(s.SupportedKinds(), ", "))
	}
	if runAt.IsZero() || !runAt.After(time.Now()) {
		return nil, fmt.Errorf("%w: run_at must be in the future", ErrInvalidScheduledTaskSchedule)
	}
	if repeatSeconds != 0 &&
		(repeatSeconds < MinScheduledTaskRepeatSeconds || repeatSeconds > MaxScheduledTaskRepeatSeconds) {
		return nil, fmt.Errorf("%w: repeat_seconds is %d", ErrInvalidScheduledTaskSchedule, repeatSeconds)
	}
	row := &models.ScheduledTask{
		UserID:        userID,
		Kind:          kind,
		NextRunAt:     runAt.UTC(),
		RepeatSeconds: repeatSeconds,
		Enabled:       true,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// List returns the user's tasks, soonest first, so the UI can show what is
// about to happen.
func (s *ScheduledTaskService) List(ctx context.Context, userID uint) ([]models.ScheduledTask, error) {
	var rows []models.ScheduledTask
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("next_run_at, id").
		Find(&rows).Error
	return rows, err
}

// Get is the ownership-scoped read used by delete.
func (s *ScheduledTaskService) Get(ctx context.Context, userID, id uint) (*models.ScheduledTask, error) {
	var row models.ScheduledTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Delete removes one of the user's tasks. Filtering on user_id is what makes
// another user's id a 404 rather than a deletion.
func (s *ScheduledTaskService) Delete(ctx context.Context, userID, id uint) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.ScheduledTask{}).Error
}

// RunDue claims and runs every task that is due at `now`, returning how many it
// ran. Run is exported so maintenance commands and tests can drive the
// scheduler without waiting for the ticker.
func (s *ScheduledTaskService) RunDue(ctx context.Context, now time.Time) (int, error) {
	claimed, err := s.claimDue(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, task := range claimed {
		s.runOne(ctx, task, now)
	}
	return len(claimed), nil
}

// claimDue atomically takes the due tasks and advances their schedule.
//
// The advance happens inside the claiming transaction, before the action runs,
// so a second worker (or a second instance) can never fire the same occurrence
// twice. The trade-off is at-most-once: a crash in the window between the claim
// and the action skips that occurrence. That is the right side to err on for a
// scheduler - a duplicate "sign out everywhere" is surprising, a skipped one is
// visible in last_status - and recurring tasks simply run again next period.
//
// A recurring task whose worker was down for several periods fires once and is
// rescheduled from `now`, rather than replaying every missed occurrence.
func (s *ScheduledTaskService) claimDue(ctx context.Context, now time.Time) ([]models.ScheduledTask, error) {
	var claimed []models.ScheduledTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`
			SELECT * FROM scheduled_tasks
			WHERE enabled = TRUE AND next_run_at <= ?
			ORDER BY next_run_at, id
			LIMIT ?
			FOR UPDATE SKIP LOCKED`, now, scheduledTaskBatch).Scan(&claimed).Error; err != nil {
			return err
		}
		for i := range claimed {
			updates := map[string]interface{}{"updated_at": now}
			if claimed[i].RepeatSeconds > 0 {
				updates["next_run_at"] = now.Add(time.Duration(claimed[i].RepeatSeconds) * time.Second)
			} else {
				updates["enabled"] = false
			}
			if err := tx.Model(&models.ScheduledTask{}).
				Where("id = ?", claimed[i].ID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return claimed, err
}

// runOne executes one claimed task and records the outcome. Failures are
// recorded rather than retried in place: the schedule has already moved on, and
// the next occurrence (or the operator reading last_error) is the recovery.
func (s *ScheduledTaskService) runOne(ctx context.Context, task models.ScheduledTask, now time.Time) {
	action, ok := s.actions[task.Kind]
	status := ScheduledTaskStatusOK
	message := ""
	if !ok {
		status = ScheduledTaskStatusFailed
		message = "no action registered for kind " + task.Kind
	} else if err := action.Run(ctx, task); err != nil {
		status = ScheduledTaskStatusFailed
		message = err.Error()
		if len(message) > 255 {
			message = message[:255]
		}
	}
	if err := s.db.WithContext(ctx).Model(&models.ScheduledTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"last_run_at": now,
			"last_status": status,
			"last_error":  message,
			"updated_at":  now,
		}).Error; err != nil {
		log.Printf("[scheduled-task] record outcome for task %d failed: %v", task.ID, err)
	}
	if s.audit != nil {
		s.audit.LogActor(ctx, task.UserID, "scheduled_task.run", "scheduled_task",
			fmt.Sprintf("%d", task.ID),
			fmt.Sprintf("kind=%s status=%s error=%q", task.Kind, status, message),
			"")
	}
}

// Run drives RunDue on a ticker until ctx is cancelled. interval <= 0 disables
// the worker entirely, for a deployment that never schedules anything; a tick
// with nothing due is one indexed query.
func (s *ScheduledTaskService) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RunDue(ctx, time.Now().UTC()); err != nil {
				log.Printf("[scheduled-task] run due tasks failed: %v", err)
			}
		}
	}
}

// SignOutAllSessionsAction revokes every active session family of the task's
// owner. It is the first registered kind because the whole effect lives in the
// control plane: no controlled machine and no controller client has to
// cooperate, so it can be scheduled and verified today.
type SignOutAllSessionsAction struct {
	tokens *TokenService
	bus    *EventBus
}

func NewSignOutAllSessionsAction(tokens *TokenService, bus *EventBus) *SignOutAllSessionsAction {
	return &SignOutAllSessionsAction{tokens: tokens, bus: bus}
}

func (a *SignOutAllSessionsAction) Run(ctx context.Context, task models.ScheduledTask) error {
	if a.tokens == nil {
		return errors.New("token service is not configured")
	}
	a.tokens.RevokeAllForSubject(ctx, ScopeUser, task.UserID)
	if a.bus != nil {
		a.bus.Publish(ctx, Event{
			Type:   EventSessionRevoked,
			UserID: task.UserID,
			Data: map[string]interface{}{
				"reason":  "scheduled_task",
				"task_id": task.ID,
			},
		})
	}
	return nil
}
