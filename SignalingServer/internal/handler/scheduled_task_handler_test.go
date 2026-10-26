package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newScheduledTaskHandlerTestDB(t *testing.T) (*gorm.DB, *service.ScheduledTaskService) {
	t.Helper()
	db := openTestPostgres(t)
	migrateTestSchema(t, db, &models.ScheduledTask{})
	if err := db.Exec("TRUNCATE scheduled_tasks").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	svc := service.NewScheduledTaskService(db).RegisterAction(
		models.ScheduledTaskSignOutAllSessions, &scheduledTaskNoopAction{})
	return db, svc
}

// scheduledTaskNoopAction is a registered action that does nothing: the handler
// tests are about the HTTP surface, not about what a kind does.
type scheduledTaskNoopAction struct{}

func (*scheduledTaskNoopAction) Run(_ context.Context, _ models.ScheduledTask) error { return nil }

func scheduledTaskRouter(h *ScheduledTaskHandler) *gin.Engine {
	router := gin.New()
	router.PATCH("/scheduled-tasks/:id", h.Update)
	return router
}

// The edit is the half of "the page can fix a mistake" the server owes. A task
// whose retry budget or timezone is wrong could only be deleted and recreated,
// which discarded the outcome the owner needed in order to see it was failing.
func TestScheduledTaskHandlerUpdateEditsARow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, svc := newScheduledTaskHandlerTestDB(t)
	const userID = uint(991501)
	seed, err := svc.Create(t.Context(), userID, service.ScheduledTaskSpec{
		Kind: models.ScheduledTaskSignOutAllSessions,
		RunAt: time.Now().Add(time.Hour), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	h := NewScheduledTaskHandler(svc)

	newRunAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	body, _ := json.Marshal(map[string]any{
		"kind":          models.ScheduledTaskSignOutAllSessions,
		"run_at":        newRunAt,
		"repeat_seconds": 120,
		"max_attempts":  4,
	})
	req := httptest.NewRequest(http.MethodPatch, "/scheduled-tasks/"+strconv.FormatUint(uint64(seed.ID), 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(seed.ID), 10)}}
	c.Set("auth.user_id", userID)
	h.Update(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var view map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view["repeat_seconds"].(float64) != 120 || view["max_attempts"].(float64) != 4 {
		t.Fatalf("patched view = %v, want repeat 120 and budget 4", view)
	}

	var stored models.ScheduledTask
	if err := db.First(&stored, seed.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.RepeatSeconds != 120 || stored.MaxAttempts != 4 || !stored.Enabled {
		t.Fatalf("stored row = %+v, want the patched schedule, still enabled", stored)
	}
}

// Another user's id is a 404, and the row must be untouched: an edit is a write,
// so a leaked id must not be enough to reach somebody else's schedule.
func TestScheduledTaskHandlerUpdateRejectsAForeignRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, svc := newScheduledTaskHandlerTestDB(t)
	const ownerID = uint(991502)
	const attackerID = uint(991503)
	seed, err := svc.Create(t.Context(), ownerID, service.ScheduledTaskSpec{
		Kind: models.ScheduledTaskSignOutAllSessions,
		RunAt: time.Now().Add(time.Hour), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	originalRunAt := seed.NextRunAt
	h := NewScheduledTaskHandler(svc)

	body, _ := json.Marshal(map[string]any{
		"kind":   models.ScheduledTaskSignOutAllSessions,
		"run_at": time.Now().Add(5 * time.Hour).UTC().Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPatch, "/scheduled-tasks/"+strconv.FormatUint(uint64(seed.ID), 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(seed.ID), 10)}}
	c.Set("auth.user_id", attackerID)
	h.Update(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("patch status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	var stored models.ScheduledTask
	if err := db.First(&stored, seed.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !stored.NextRunAt.Equal(originalRunAt) {
		t.Fatalf("foreign row moved to %s, want %s untouched", stored.NextRunAt, originalRunAt)
	}
}

// A schedule the worker cannot use is a 400, not a stored row that parks on the
// first tick. Same rules as create, because an edit is a create.
func TestScheduledTaskHandlerUpdateRejectsAnUnusableSchedule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, svc := newScheduledTaskHandlerTestDB(t)
	const userID = uint(991504)
	seed, err := svc.Create(t.Context(), userID, service.ScheduledTaskSpec{
		Kind: models.ScheduledTaskSignOutAllSessions,
		RunAt: time.Now().Add(time.Hour), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}
	h := NewScheduledTaskHandler(svc)

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"unknown kind", map[string]any{"kind": "nope", "run_at": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)}},
		{"past run_at", map[string]any{"kind": models.ScheduledTaskSignOutAllSessions, "run_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}},
		{"budget too high", map[string]any{"kind": models.ScheduledTaskSignOutAllSessions, "run_at": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339), "max_attempts": 11}},
		{"broken schedule", map[string]any{"kind": models.ScheduledTaskSignOutAllSessions, "schedule": "whenever"}},
		{"broken run_at format", map[string]any{"kind": models.ScheduledTaskSignOutAllSessions, "run_at": "tomorrow"}},
	} {
		body, _ := json.Marshal(tc.body)
		req := httptest.NewRequest(http.MethodPatch, "/scheduled-tasks/"+strconv.FormatUint(uint64(seed.ID), 10), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(seed.ID), 10)}}
		c.Set("auth.user_id", userID)
		h.Update(c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400; body = %s", tc.name, rec.Code, rec.Body.String())
		}
	}
}

// An id that is not a number is a 400 rather than a lookup with a garbage key.
func TestScheduledTaskHandlerUpdateRejectsANonNumericID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, svc := newScheduledTaskHandlerTestDB(t)
	h := NewScheduledTaskHandler(svc)

	body, _ := json.Marshal(map[string]any{
		"kind":   models.ScheduledTaskSignOutAllSessions,
		"run_at": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPatch, "/scheduled-tasks/not-a-number", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "not-a-number"}}
	c.Set("auth.user_id", uint(991505))
	h.Update(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}
