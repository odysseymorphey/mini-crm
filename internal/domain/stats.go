package domain

import "context"

type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type TopAssignee struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	ClosedCount int64  `json:"closed_count"`
}

type TeamStats struct {
	TeamID                int64         `json:"team_id"`
	TasksByStatus         []StatusCount `json:"tasks_by_status"`
	TopAssignees          []TopAssignee `json:"top_3_assignees"`
	AvgClosingTimeSeconds float64       `json:"avg_closing_time_seconds"`
	TotalComments         int64         `json:"total_comments"`
}

type StatsRepository interface {
	GetTeamStats(ctx context.Context, teamID int64) (*TeamStats, error)
}
