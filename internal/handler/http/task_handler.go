package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"mini-crm/internal/domain"
	"mini-crm/internal/middleware"
	"mini-crm/internal/service"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// CreateTask godoc
// @Summary Create task in a team
// @Tags Tasks
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body domain.CreateTaskRequest true "Task creation payload"
// @Success 201 {object} domain.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), userID, req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, task)
}

// ListTasks godoc
// @Summary List tasks for a team with filters and pagination
// @Tags Tasks
// @Security BearerAuth
// @Produce json
// @Param team_id query int true "Team ID (mandatory)"
// @Param status query string false "Task status (todo, in_progress, done, closed)"
// @Param assignee_id query int false "Assignee User ID"
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Param cursor query int false "Cursor (Task ID after which to fetch)"
// @Success 200 {object} domain.TaskListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/tasks [get]
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	q := r.URL.Query()
	teamIDStr := q.Get("team_id")
	if teamIDStr == "" {
		RespondError(w, http.StatusBadRequest, "team_id query parameter is required")
		return
	}

	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid team_id query parameter")
		return
	}

	filter := domain.TaskFilter{
		TeamID: teamID,
		Limit:  20,
		Offset: 0,
	}

	if status := q.Get("status"); status != "" {
		taskStatus := domain.TaskStatus(status)
		filter.Status = &taskStatus
	}

	if assigneeStr := q.Get("assignee_id"); assigneeStr != "" {
		if assigneeID, err := strconv.ParseInt(assigneeStr, 10, 64); err == nil && assigneeID > 0 {
			filter.AssigneeID = &assigneeID
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if cursorStr := q.Get("cursor"); cursorStr != "" {
		if cursor, err := strconv.ParseInt(cursorStr, 10, 64); err == nil && cursor > 0 {
			filter.Cursor = &cursor
		}
	}

	tasks, err := h.taskService.ListTasks(r.Context(), userID, filter)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, tasks)
}

// UpdateTask godoc
// @Summary Update a task (role-based permissions and optimistic locking)
// @Tags Tasks
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body domain.UpdateTaskRequest true "Task update payload"
// @Success 200 {object} domain.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	idParam := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || taskID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskService.UpdateTask(r.Context(), taskID, userID, req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, task)
}

// GetTaskHistory godoc
// @Summary Get history of changes for a task
// @Tags Tasks
// @Security BearerAuth
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {array} domain.TaskHistory
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/tasks/{id}/history [get]
func (h *TaskHandler) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	idParam := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || taskID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	history, err := h.taskService.GetTaskHistory(r.Context(), taskID, userID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, history)
}
