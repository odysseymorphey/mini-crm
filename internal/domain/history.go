package domain

import (
	"context"
	"time"
)

type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type TaskHistory struct {
	ID            int64                  `json:"id"`
	TaskID        int64                  `json:"task_id"`
	ChangedBy     int64                  `json:"changed_by"`
	Changes       map[string]FieldChange `json:"changes"`
	CreatedAt     time.Time              `json:"created_at"`
	ChangedByName string                 `json:"changed_by_name,omitempty"`
}

type TaskHistoryRepository interface {
	Create(ctx context.Context, history *TaskHistory) error
	GetByTaskID(ctx context.Context, taskID int64) ([]TaskHistory, error)
}
