package mysql

import (
	"context"
	"encoding/json"
	"fmt"

	"mini-crm/internal/domain"
)

type HistoryRepository struct {
	db *DB
}

func NewHistoryRepository(db *DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Create(ctx context.Context, history *domain.TaskHistory) error {
	changesJSON, err := json.Marshal(history.Changes)
	if err != nil {
		return fmt.Errorf("failed to marshal history changes: %w", err)
	}

	query := `INSERT INTO task_history (task_id, changed_by, changes, created_at) VALUES (?, ?, ?, NOW())`
	res, err := r.db.ExecContext(ctx, query, history.TaskID, history.ChangedBy, string(changesJSON))
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	history.ID = id
	return nil
}

func (r *HistoryRepository) GetByTaskID(ctx context.Context, taskID int64) ([]domain.TaskHistory, error) {
	query := `
		SELECT th.id, th.task_id, th.changed_by, th.changes, th.created_at, u.name
		FROM task_history th
		JOIN users u ON th.changed_by = u.id
		WHERE th.task_id = ?
		ORDER BY th.created_at ASC, th.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.TaskHistory
	for rows.Next() {
		var h domain.TaskHistory
		var changesRaw []byte

		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &changesRaw, &h.CreatedAt, &h.ChangedByName); err != nil {
			return nil, err
		}

		if len(changesRaw) > 0 {
			if err := json.Unmarshal(changesRaw, &h.Changes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal history changes: %w", err)
			}
		}
		list = append(list, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []domain.TaskHistory{}
	}

	return list, nil
}
