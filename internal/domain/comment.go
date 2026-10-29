package domain

import (
	"context"
	"time"
)

type TaskComment struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

type CommentRepository interface {
	Create(ctx context.Context, comment *TaskComment) error
	GetByTaskID(ctx context.Context, taskID int64) ([]TaskComment, error)
}
