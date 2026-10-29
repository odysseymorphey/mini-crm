package mysql

import (
	"context"

	"mini-crm/internal/domain"
)

type CommentRepository struct {
	db *DB
}

func NewCommentRepository(db *DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *domain.TaskComment) error {
	query := `INSERT INTO task_comments (task_id, user_id, content, created_at) VALUES (?, ?, ?, NOW())`
	res, err := r.db.ExecContext(ctx, query, comment.TaskID, comment.UserID, comment.Content)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	comment.ID = id
	return nil
}

func (r *CommentRepository) GetByTaskID(ctx context.Context, taskID int64) ([]domain.TaskComment, error) {
	query := `
		SELECT tc.id, tc.task_id, tc.user_id, tc.content, tc.created_at, u.name
		FROM task_comments tc
		JOIN users u ON tc.user_id = u.id
		WHERE tc.task_id = ?
		ORDER BY tc.created_at ASC, tc.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.TaskComment
	for rows.Next() {
		var c domain.TaskComment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt, &c.UserName); err != nil {
			return nil, err
		}
		list = append(list, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []domain.TaskComment{}
	}

	return list, nil
}
