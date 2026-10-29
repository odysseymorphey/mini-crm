package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"mini-crm/internal/domain"
)

type TaskRepository struct {
	db *DB
}

func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) CreateWithHistory(ctx context.Context, task *domain.Task, history *domain.TaskHistory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	taskQuery := `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW(), ?, 1)
	`
	res, err := tx.ExecContext(ctx, taskQuery,
		task.TeamID,
		task.Title,
		task.Description,
		task.Status,
		task.CreatedBy,
		task.AssigneeID,
		task.ClosedAt,
	)
	if err != nil {
		return err
	}

	taskID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = taskID
	task.Version = 1

	if history != nil {
		history.TaskID = taskID
		changesJSON, err := json.Marshal(history.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal history changes: %w", err)
		}

		histQuery := `
			INSERT INTO task_history (task_id, changed_by, changes, created_at)
			VALUES (?, ?, ?, NOW())
		`
		histRes, err := tx.ExecContext(ctx, histQuery, taskID, history.ChangedBy, string(changesJSON))
		if err != nil {
			return err
		}
		histID, err := histRes.LastInsertId()
		if err == nil {
			history.ID = histID
		}
	}

	return tx.Commit()
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	query := `
		SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		FROM tasks
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var t domain.Task
	var assigneeID sql.NullInt64
	var closedAt sql.NullTime

	err := row.Scan(
		&t.ID,
		&t.TeamID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.CreatedBy,
		&assigneeID,
		&t.CreatedAt,
		&t.UpdatedAt,
		&closedAt,
		&t.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}

	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.Int64
	}
	if closedAt.Valid {
		t.ClosedAt = &closedAt.Time
	}

	return &t, nil
}

func (r *TaskRepository) GetByIDForUpdate(ctx context.Context, id int64) (*domain.Task, error) {
	query := `
		SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		FROM tasks
		WHERE id = ?
		FOR UPDATE
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var t domain.Task
	var assigneeID sql.NullInt64
	var closedAt sql.NullTime

	err := row.Scan(
		&t.ID,
		&t.TeamID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.CreatedBy,
		&assigneeID,
		&t.CreatedAt,
		&t.UpdatedAt,
		&closedAt,
		&t.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}

	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.Int64
	}
	if closedAt.Valid {
		t.ClosedAt = &closedAt.Time
	}

	return &t, nil
}

func (r *TaskRepository) UpdateWithHistory(ctx context.Context, task *domain.Task, history *domain.TaskHistory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	currentVersion := task.Version
	newVersion := currentVersion + 1

	updateQuery := `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, version = ?, updated_at = NOW()
		WHERE id = ? AND version = ?
	`
	res, err := tx.ExecContext(ctx, updateQuery,
		task.Title,
		task.Description,
		task.Status,
		task.AssigneeID,
		task.ClosedAt,
		newVersion,
		task.ID,
		currentVersion,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrVersionConflict
	}

	task.Version = newVersion

	if history != nil && len(history.Changes) > 0 {
		changesJSON, err := json.Marshal(history.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal history changes: %w", err)
		}

		histQuery := `
			INSERT INTO task_history (task_id, changed_by, changes, created_at)
			VALUES (?, ?, ?, NOW())
		`
		histRes, err := tx.ExecContext(ctx, histQuery, task.ID, history.ChangedBy, string(changesJSON))
		if err != nil {
			return err
		}
		histID, err := histRes.LastInsertId()
		if err == nil {
			history.ID = histID
		}
	}

	return tx.Commit()
}

func (r *TaskRepository) List(ctx context.Context, filter domain.TaskFilter) (*domain.TaskListResponse, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "team_id = ?")
	args = append(args, filter.TeamID)

	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}

	if filter.AssigneeID != nil {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, *filter.AssigneeID)
	}

	if filter.Cursor != nil && *filter.Cursor > 0 {
		conditions = append(conditions, "id > ?")
		args = append(args, *filter.Cursor)
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", whereClause)
	var totalCount int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		FROM tasks
		WHERE %s
		ORDER BY id ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	queryArgs := append(args, limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.Task
	var lastID int64
	for rows.Next() {
		var t domain.Task
		var assigneeID sql.NullInt64
		var closedAt sql.NullTime

		if err := rows.Scan(
			&t.ID,
			&t.TeamID,
			&t.Title,
			&t.Description,
			&t.Status,
			&t.CreatedBy,
			&assigneeID,
			&t.CreatedAt,
			&t.UpdatedAt,
			&closedAt,
			&t.Version,
		); err != nil {
			return nil, err
		}

		if assigneeID.Valid {
			t.AssigneeID = &assigneeID.Int64
		}
		if closedAt.Valid {
			t.ClosedAt = &closedAt.Time
		}

		lastID = t.ID
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	resp := &domain.TaskListResponse{
		Tasks:      tasks,
		TotalCount: totalCount,
	}

	if len(tasks) == limit {
		resp.NextCursor = &lastID
	}

	return resp, nil
}
