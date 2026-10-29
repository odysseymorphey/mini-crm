package domain

import (
	"context"
	"time"
)

type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusClosed     TaskStatus = "closed"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone, StatusClosed:
		return true
	default:
		return false
	}
}

func (s TaskStatus) IsClosed() bool {
	return s == StatusClosed || s == StatusDone
}

type Task struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	AssigneeID  *int64     `json:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	Version     int        `json:"version"`
}

type CreateTaskRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  *int64 `json:"assignee_id,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *TaskStatus `json:"status,omitempty"`
	AssigneeID  *int64      `json:"assignee_id,omitempty"`
	Version     int         `json:"version"`
}

type TaskFilter struct {
	TeamID     int64       `json:"team_id"`
	Status     *TaskStatus `json:"status,omitempty"`
	AssigneeID *int64      `json:"assignee_id,omitempty"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	Cursor     *int64      `json:"cursor,omitempty"`
}

type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	TotalCount int64  `json:"total_count"`
	NextCursor *int64 `json:"next_cursor,omitempty"`
}

type TaskRepository interface {
	CreateWithHistory(ctx context.Context, task *Task, history *TaskHistory) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*Task, error)
	UpdateWithHistory(ctx context.Context, task *Task, history *TaskHistory) error
	List(ctx context.Context, filter TaskFilter) (*TaskListResponse, error)
}
