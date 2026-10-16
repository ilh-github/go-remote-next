package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"quickdesk/signaling/internal/middleware"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ScheduledTaskHandler serves the owner-scheduled control-plane actions:
//
//	GET    /v1/me/scheduled-tasks
//	POST   /v1/me/scheduled-tasks
//	DELETE /v1/me/scheduled-tasks/:id
//
// It is a schedule-management surface, not an execution one: the worker runs
// due tasks in the background, and `last_status`/`last_error` are how a caller
// finds out what happened.
type ScheduledTaskHandler struct {
	tasks *service.ScheduledTaskService
}

func NewScheduledTaskHandler(tasks *service.ScheduledTaskService) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{tasks: tasks}
}

type scheduledTaskReq struct {
	Kind string `json:"kind" binding:"required"`
	// RunAt is RFC3339 and must be in the future.
	RunAt string `json:"run_at" binding:"required"`
	// RepeatSeconds is 0 for a one-shot task, or 60..31536000 to repeat.
	RepeatSeconds int `json:"repeat_seconds"`
}

// scheduledTaskView is the wire shape. The outcome fields are exposed so an
// owner can see that a task fired and what it did without reading server logs.
type scheduledTaskView struct {
	ID            uint       `json:"id"`
	Kind          string     `json:"kind"`
	NextRunAt     time.Time  `json:"next_run_at"`
	RepeatSeconds int        `json:"repeat_seconds"`
	Enabled       bool       `json:"enabled"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	LastStatus    string     `json:"last_status,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func scheduledTaskViewOf(row models.ScheduledTask) scheduledTaskView {
	return scheduledTaskView{
		ID:            row.ID,
		Kind:          row.Kind,
		NextRunAt:     row.NextRunAt,
		RepeatSeconds: row.RepeatSeconds,
		Enabled:       row.Enabled,
		LastRunAt:     row.LastRunAt,
		LastStatus:    row.LastStatus,
		LastError:     row.LastError,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func (h *ScheduledTaskHandler) List(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	rows, err := h.tasks.List(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	items := make([]scheduledTaskView, 0, len(rows))
	for _, row := range rows {
		items = append(items, scheduledTaskViewOf(row))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":           items,
		"supported_kinds": h.tasks.SupportedKinds(),
	})
}

func (h *ScheduledTaskHandler) Create(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	var req scheduledTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
		return
	}
	runAt, err := time.Parse(time.RFC3339, req.RunAt)
	if err != nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "run_at must be RFC3339, e.g. 2026-10-17T03:00:00Z")
		return
	}
	row, err := h.tasks.Create(c.Request.Context(), middleware.MustUserID(c), req.Kind, runAt, req.RepeatSeconds)
	if err != nil {
		if errors.Is(err, service.ErrInvalidScheduledTaskKind) ||
			errors.Is(err, service.ErrInvalidScheduledTaskSchedule) {
			ProblemBadRequest(c, ProblemCodeInvalidRequest, err.Error())
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, scheduledTaskViewOf(*row))
}

func (h *ScheduledTaskHandler) Delete(c *gin.Context) {
	if h.tasks == nil {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "scheduled tasks are not configured")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ProblemBadRequest(c, ProblemCodeInvalidRequest, "invalid scheduled task id")
		return
	}
	uid := middleware.MustUserID(c)
	// Resolve first so deleting someone else's id is a 404 rather than a
	// silent success, and deleting an id that never existed is honest too.
	if _, err := h.tasks.Get(c.Request.Context(), uid, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ProblemNotFound(c, ProblemCodeNotFound, "Scheduled task not found")
			return
		}
		ProblemInternal(c, err.Error())
		return
	}
	if err := h.tasks.Delete(c.Request.Context(), uid, uint(id)); err != nil {
		ProblemInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
