package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"mini-crm/internal/domain"
	"mini-crm/internal/repository/redis"
)

type TaskService struct {
	taskRepo    domain.TaskRepository
	teamRepo    domain.TeamRepository
	historyRepo domain.TaskHistoryRepository
	cacheRepo   *redis.TaskCacheRepository
}

func NewTaskService(
	taskRepo domain.TaskRepository,
	teamRepo domain.TeamRepository,
	historyRepo domain.TaskHistoryRepository,
	cacheRepo *redis.TaskCacheRepository,
) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		teamRepo:    teamRepo,
		historyRepo: historyRepo,
		cacheRepo:   cacheRepo,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, userID int64, req domain.CreateTaskRequest) (*domain.Task, error) {
	if req.TeamID <= 0 {
		return nil, domain.ErrMissingTeamID
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return nil, domain.ErrInvalidInput
	}

	isMember, err := s.teamRepo.IsMember(ctx, req.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	if req.AssigneeID != nil && *req.AssigneeID > 0 {
		assigneeIsMember, err := s.teamRepo.IsMember(ctx, req.TeamID, *req.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !assigneeIsMember {
			return nil, domain.ErrAssigneeNotMember
		}
	}

	task := &domain.Task{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		Status:      domain.StatusTodo,
		CreatedBy:   userID,
		AssigneeID:  req.AssigneeID,
	}

	history := &domain.TaskHistory{
		ChangedBy: userID,
		Changes: map[string]domain.FieldChange{
			"status": {Old: nil, New: string(domain.StatusTodo)},
			"title":  {Old: nil, New: req.Title},
		},
	}
	if req.AssigneeID != nil {
		history.Changes["assignee_id"] = domain.FieldChange{Old: nil, New: *req.AssigneeID}
	}

	if err := s.taskRepo.CreateWithHistory(ctx, task, history); err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		if err := s.cacheRepo.Invalidate(ctx, req.TeamID); err != nil {
			slog.Warn("failed to invalidate tasks cache", "team_id", req.TeamID, "err", err)
		}
	}

	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, userID int64, filter domain.TaskFilter) (*domain.TaskListResponse, error) {
	if filter.TeamID <= 0 {
		return nil, domain.ErrMissingTeamID
	}

	isMember, err := s.teamRepo.IsMember(ctx, filter.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	if s.cacheRepo != nil {
		if cached, err := s.cacheRepo.Get(ctx, filter); err == nil && cached != nil {
			return cached, nil
		}
	}

	resp, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		if err := s.cacheRepo.Set(ctx, filter, resp); err != nil {
			slog.Warn("failed to set tasks cache", "team_id", filter.TeamID, "err", err)
		}
	}

	return resp, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, taskID, userID int64, req domain.UpdateTaskRequest) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	requesterMember, err := s.teamRepo.GetMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, domain.ErrNotTeamMember
	}

	if req.Version > 0 && req.Version != task.Version {
		return nil, domain.ErrVersionConflict
	}

	isOwnerOrAdmin := requesterMember.Role == domain.RoleOwner || requesterMember.Role == domain.RoleAdmin
	isCreator := task.CreatedBy == userID
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == userID

	if !isOwnerOrAdmin && !isCreator && !isAssignee {
		return nil, domain.ErrForbidden
	}

	if isAssignee && !isOwnerOrAdmin && !isCreator {
		if req.Title != nil && *req.Title != task.Title {
			return nil, fmt.Errorf("%w: assignee cannot modify task title", domain.ErrForbidden)
		}
		if req.Description != nil && *req.Description != task.Description {
			return nil, fmt.Errorf("%w: assignee cannot modify task description", domain.ErrForbidden)
		}
		if req.AssigneeID != nil {
			if task.AssigneeID == nil || *req.AssigneeID != *task.AssigneeID {
				return nil, fmt.Errorf("%w: assignee cannot reassign task", domain.ErrForbidden)
			}
		}
	}

	if req.AssigneeID != nil {
		isNewAssigneeMember, err := s.teamRepo.IsMember(ctx, task.TeamID, *req.AssigneeID)
		if err != nil {
			return nil, err
		}
		if !isNewAssigneeMember {
			return nil, domain.ErrAssigneeNotMember
		}
	}

	changes := make(map[string]domain.FieldChange)

	if req.Title != nil && *req.Title != task.Title {
		newTitle := strings.TrimSpace(*req.Title)
		if newTitle == "" {
			return nil, domain.ErrInvalidInput
		}
		changes["title"] = domain.FieldChange{Old: task.Title, New: newTitle}
		task.Title = newTitle
	}

	if req.Description != nil && *req.Description != task.Description {
		changes["description"] = domain.FieldChange{Old: task.Description, New: *req.Description}
		task.Description = *req.Description
	}

	if req.Status != nil && *req.Status != task.Status {
		if !req.Status.IsValid() {
			return nil, fmt.Errorf("%w: invalid task status", domain.ErrInvalidInput)
		}
		changes["status"] = domain.FieldChange{Old: string(task.Status), New: string(*req.Status)}
		task.Status = *req.Status

		if req.Status.IsClosed() {
			if task.ClosedAt == nil {
				now := time.Now()
				task.ClosedAt = &now
			}
		} else {
			task.ClosedAt = nil
		}
	}

	if req.AssigneeID != nil {
		if task.AssigneeID == nil || *req.AssigneeID != *task.AssigneeID {
			var oldAssignee any = nil
			if task.AssigneeID != nil {
				oldAssignee = *task.AssigneeID
			}
			changes["assignee_id"] = domain.FieldChange{Old: oldAssignee, New: *req.AssigneeID}
			task.AssigneeID = req.AssigneeID
		}
	}

	if len(changes) == 0 {
		return task, nil
	}

	history := &domain.TaskHistory{
		TaskID:    task.ID,
		ChangedBy: userID,
		Changes:   changes,
	}

	if err := s.taskRepo.UpdateWithHistory(ctx, task, history); err != nil {
		return nil, err
	}

	if s.cacheRepo != nil {
		if err := s.cacheRepo.Invalidate(ctx, task.TeamID); err != nil {
			slog.Warn("failed to invalidate tasks cache", "team_id", task.TeamID, "err", err)
		}
	}

	return task, nil
}

func (s *TaskService) GetTaskHistory(ctx context.Context, taskID, userID int64) ([]domain.TaskHistory, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.teamRepo.IsMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	return s.historyRepo.GetByTaskID(ctx, taskID)
}

func (s *TaskService) GetTask(ctx context.Context, taskID, userID int64) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.teamRepo.IsMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotTeamMember
	}

	return task, nil
}
