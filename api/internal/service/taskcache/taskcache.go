package taskcache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TaskCache struct {
	client      *redis.Client
	taskTTL     time.Duration
	taskListTTL time.Duration
}

func New(client *redis.Client, taskTTL, taskListTTL time.Duration) *TaskCache {
	return &TaskCache{
		client:      client,
		taskTTL:     taskTTL,
		taskListTTL: taskListTTL,
	}
}

func BuildTaskKey(taskID int64) string {
	return fmt.Sprintf("task:%d", taskID)
}

func BuildTaskListKey(userID int64, limit int) string {
	return fmt.Sprintf("task_list:user:%d:limit:%d", userID, limit)
}

func (c *TaskCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	raw, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis GET failed for key=%s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return false, fmt.Errorf("unmarshal cached value for key=%s: %w", key, err)
	}

	return true, nil
}

func (c *TaskCache) SetTask(ctx context.Context, taskID int64, value any) error {
	return c.set(ctx, BuildTaskKey(taskID), value, c.taskTTL)
}

func (c *TaskCache) SetTaskList(ctx context.Context, userID int64, limit int, value any) error {
	return c.set(ctx, BuildTaskListKey(userID, limit), value, c.taskListTTL)
}

func (c *TaskCache) DeleteTask(ctx context.Context, taskID int64) error {
	return c.client.Del(ctx, BuildTaskKey(taskID)).Err()
}

func (c *TaskCache) DeleteTaskList(ctx context.Context, userID int64, limit int) error {
	return c.client.Del(ctx, BuildTaskListKey(userID, limit)).Err()
}

func (c *TaskCache) DeleteTaskListsForUser(ctx context.Context, userID int64) error {
	pattern := fmt.Sprintf("task_list:user:%d:limit:*", userID)

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis SCAN failed for pattern=%s: %w", pattern, err)
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("redis DEL failed for task list keys: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (c *TaskCache) set(ctx context.Context, key string, value any, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value for key=%s: %w", key, err)
	}

	if err := c.client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return fmt.Errorf("redis SET failed for key=%s: %w", key, err)
	}

	return nil
}
