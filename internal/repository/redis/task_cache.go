package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mini-crm/internal/domain"
)

type TaskCacheRepository struct {
	client *Client
	ttl    time.Duration
}

func NewTaskCacheRepository(client *Client, ttl time.Duration) *TaskCacheRepository {
	return &TaskCacheRepository{
		client: client,
		ttl:    ttl,
	}
}

func (c *TaskCacheRepository) buildKey(filter domain.TaskFilter) string {
	status := "all"
	if filter.Status != nil && *filter.Status != "" {
		status = string(*filter.Status)
	}

	assignee := "all"
	if filter.AssigneeID != nil {
		assignee = fmt.Sprintf("%d", *filter.AssigneeID)
	}

	cursor := "0"
	if filter.Cursor != nil {
		cursor = fmt.Sprintf("%d", *filter.Cursor)
	}

	return fmt.Sprintf("team:%d:tasks:s=%s:a=%s:l=%d:o=%d:c=%s",
		filter.TeamID, status, assignee, filter.Limit, filter.Offset, cursor)
}

func (c *TaskCacheRepository) Get(ctx context.Context, filter domain.TaskFilter) (*domain.TaskListResponse, error) {
	key := c.buildKey(filter)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var resp domain.TaskListResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *TaskCacheRepository) Set(ctx context.Context, filter domain.TaskFilter, resp *domain.TaskListResponse) error {
	key := c.buildKey(filter)
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, string(data), c.ttl)
}

func (c *TaskCacheRepository) Invalidate(ctx context.Context, teamID int64) error {
	return c.client.InvalidateTeamTasks(ctx, teamID)
}
