package service

import (
	"context"
	"strings"

	"mini-crm/internal/domain"
)

type CommentService struct {
	commentRepo domain.CommentRepository
	taskRepo    domain.TaskRepository
	teamRepo    domain.TeamRepository
}

func NewCommentService(
	commentRepo domain.CommentRepository,
	taskRepo domain.TaskRepository,
	teamRepo domain.TeamRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		taskRepo:    taskRepo,
		teamRepo:    teamRepo,
	}
}

func (s *CommentService) AddComment(ctx context.Context, taskID, userID int64, req domain.CreateCommentRequest) (*domain.TaskComment, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, domain.ErrInvalidInput
	}

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

	comment := &domain.TaskComment{
		TaskID:  taskID,
		UserID:  userID,
		Content: content,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) ListComments(ctx context.Context, taskID, userID int64) ([]domain.TaskComment, error) {
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

	return s.commentRepo.GetByTaskID(ctx, taskID)
}
