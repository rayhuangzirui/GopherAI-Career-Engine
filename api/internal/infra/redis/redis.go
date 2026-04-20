package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr string
	Password string
	DB int
}

func New(cfg Config) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis addr is empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:  		cfg.Addr,
		Password: 	cfg.Password,
		DB: 		cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis failed: %w", err)
	}

	return client, nil
}
