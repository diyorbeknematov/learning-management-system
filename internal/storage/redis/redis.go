package redis

import (
	"context"
	"fmt"

	redisClient "github.com/redis/go-redis/v9"

	"github.com/diyorbeknematov/lms/internal/config"
)

type Redis struct {
	Client *redisClient.Client
}

func New(cfg config.RedisConfig) (*Redis, error) {
	client := redisClient.NewClient(&redisClient.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Redis{
		Client: client,
	}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
