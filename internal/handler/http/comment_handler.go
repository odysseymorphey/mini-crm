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

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// AddComment godoc
// @Summary Add a comment to a task
// @Tags Comments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body domain.CreateCommentRequest true "Comment body"
// @Success 201 {object} domain.TaskComment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/tasks/{id}/comments [post]
func (h *CommentHandler) AddComment(w http.ResponseWriter, r *http.Request) {
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

	var req domain.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	comment, err := h.commentService.AddComment(r.Context(), taskID, userID, req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, comment)
}

// ListComments godoc
// @Summary List comments for a task
// @Tags Comments
// @Security BearerAuth
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {array} domain.TaskComment
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/tasks/{id}/comments [get]
func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
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

	comments, err := h.commentService.ListComments(r.Context(), taskID, userID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, comments)
}
