package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
	"mini-crm/internal/config"
)

type Client struct {
	rdb *redis.Client
	cb  *gobreaker.CircuitBreaker
}

func NewClient(cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("could not connect to redis, running with degraded cache fallback", "err", err)
	} else {
		slog.Info("connected to redis successfully", "addr", cfg.Addr())
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "RedisCircuitBreaker",
		MaxRequests: 5,
		Interval:    10 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Warn("redis circuit breaker state transition", "from", from.String(), "to", to.String())
		},
	})

	return &Client{
		rdb: rdb,
		cb:  cb,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	res, err := c.cb.Execute(func() (any, error) {
		return c.rdb.Get(ctx, key).Result()
	})
	if err != nil {
		return "", err
	}
	return res.(string), nil
}

func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, c.rdb.Set(ctx, key, value, expiration).Err()
	})
	return err
}

func (c *Client) InvalidateTeamTasks(ctx context.Context, teamID int64) error {
	pattern := fmt.Sprintf("team:%d:tasks:*", teamID)

	_, err := c.cb.Execute(func() (any, error) {
		var cursor uint64
		var keysToDelete []string

		for {
			keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return nil, err
			}
			keysToDelete = append(keysToDelete, keys...)
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		if len(keysToDelete) > 0 {
			if err := c.rdb.Del(ctx, keysToDelete...).Err(); err != nil {
				return nil, err
			}
			slog.Debug("invalidated team tasks cache keys", "team_id", teamID, "count", len(keysToDelete))
		}
		return nil, nil
	})

	return err
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
