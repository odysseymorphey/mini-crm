package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"mini-crm/internal/domain"
)

type StatsRepository struct {
	db *DB
}

func NewStatsRepository(db *DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) GetTeamStats(ctx context.Context, teamID int64) (*domain.TeamStats, error) {
	query := `
		WITH
		  team_tasks AS (
		    SELECT id, status, created_at, closed_at, assignee_id
		    FROM tasks
		    WHERE team_id = ?
		  ),
		  status_counts AS (
		    SELECT status, COUNT(*) AS cnt
		    FROM team_tasks
		    GROUP BY status
		  ),
		  status_stats AS (
		    SELECT
		      COALESCE(
		        JSON_ARRAYAGG(
		          JSON_OBJECT('status', status, 'count', cnt)
		        ),
		        JSON_ARRAY()
		      ) AS tasks_by_status
		    FROM status_counts
		  ),
		  top_assignees_ranked AS (
		    SELECT
		      u.id AS user_id,
		      u.name,
		      COUNT(t.id) AS closed_count
		    FROM team_tasks t
		    JOIN users u ON t.assignee_id = u.id
		    WHERE t.closed_at IS NOT NULL
		      AND t.closed_at >= NOW() - INTERVAL 30 DAY
		    GROUP BY u.id, u.name
		    ORDER BY closed_count DESC
		    LIMIT 3
		  ),
		  top_assignees AS (
		    SELECT
		      COALESCE(
		        JSON_ARRAYAGG(
		          JSON_OBJECT('user_id', user_id, 'name', name, 'closed_count', closed_count)
		        ),
		        JSON_ARRAY()
		      ) AS top_3_assignees
		    FROM top_assignees_ranked
		  ),
		  closing_stats AS (
		    SELECT
		      COALESCE(AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)), 0) AS avg_closing_time_seconds
		    FROM team_tasks
		    WHERE closed_at IS NOT NULL
		  ),
		  comment_stats AS (
		    SELECT
		      COUNT(c.id) AS total_comments
		    FROM task_comments c
		    JOIN team_tasks tt ON c.task_id = tt.id
		  )
		SELECT
		  ss.tasks_by_status,
		  ta.top_3_assignees,
		  cs.avg_closing_time_seconds,
		  cm.total_comments
		FROM status_stats ss
		CROSS JOIN top_assignees ta
		CROSS JOIN closing_stats cs
		CROSS JOIN comment_stats cm;
	`

	row := r.db.QueryRowContext(ctx, query, teamID)

	var tasksByStatusRaw sql.NullString
	var topAssigneesRaw sql.NullString
	var avgClosingTime float64
	var totalComments int64

	if err := row.Scan(&tasksByStatusRaw, &topAssigneesRaw, &avgClosingTime, &totalComments); err != nil {
		return nil, fmt.Errorf("error executing stats query: %w", err)
	}

	stats := &domain.TeamStats{
		TeamID:                teamID,
		AvgClosingTimeSeconds: avgClosingTime,
		TotalComments:         totalComments,
		TasksByStatus:         []domain.StatusCount{},
		TopAssignees:          []domain.TopAssignee{},
	}

	if tasksByStatusRaw.Valid && tasksByStatusRaw.String != "" {
		_ = json.Unmarshal([]byte(tasksByStatusRaw.String), &stats.TasksByStatus)
	}

	if topAssigneesRaw.Valid && topAssigneesRaw.String != "" {
		_ = json.Unmarshal([]byte(topAssigneesRaw.String), &stats.TopAssignees)
	}

	return stats, nil
}
