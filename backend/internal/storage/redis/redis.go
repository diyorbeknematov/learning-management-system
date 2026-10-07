package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	redisClient "github.com/redis/go-redis/v9"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/pkg/apperror"
)

type Client struct {
	rdb *redisClient.Client
}

func New(cfg config.RedisConfig) (*Client, error) {
	client := redisClient.NewClient(&redisClient.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &Client{
		rdb: client,
	}, nil
}

func (r *Client) Get(ctx context.Context, key string, dest any) error {
	raw, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redisClient.Nil) {
			return apperror.ErrNotFound
		}
		return fmt.Errorf("getting key %s from redis: %w", key, err)
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("unmarshaling value for key %s: %w", key, err)
	}

	return nil
}

// GetDel reads the key and deletes it in one atomic step, so a value (for
// example a password reset token) can be used only once. It uses MULTI/EXEC
// instead of the GETDEL command, which exists only in Redis 6.2 and newer.
func (r *Client) GetDel(ctx context.Context, key string, dest any) error {
	var get *redisClient.StringCmd

	_, err := r.rdb.TxPipelined(ctx, func(pipe redisClient.Pipeliner) error {
		get = pipe.Get(ctx, key)
		pipe.Del(ctx, key)

		return nil
	})

	raw, err := get.Bytes()
	if err != nil {
		if errors.Is(err, redisClient.Nil) {
			return apperror.ErrNotFound
		}
		return fmt.Errorf("getting and deleting key %s from redis: %w", key, err)
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("unmarshaling value for key %s: %w", key, err)
	}

	return nil
}

func (r *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("invalid ttl: %d", ttl)
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling value for key %s: %w", key, err)
	}

	if err := r.rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("setting key %s in redis: %w", key, err)
	}

	return nil
}

// incrScript counts a hit in a window. The first hit sets the lifetime of the
// key, in the same atomic step, so a counter can never be left without one.
// It returns the count and the milliseconds the window has left.
var incrScript = redisClient.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return {count, redis.call('PTTL', KEYS[1])}
`)

// Incr counts one more hit on key within a window that starts at the first hit
// and lasts for window. It returns how many hits there are in the window and
// how long the window still lasts. Use it for rate limits.
func (r *Client) Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if window <= 0 {
		return 0, 0, fmt.Errorf("invalid window: %d", window)
	}

	result, err := incrScript.Run(ctx, r.rdb, []string{key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return 0, 0, fmt.Errorf("counting key %s in redis: %w", key, err)
	}

	left := time.Duration(result[1]) * time.Millisecond
	if left < 0 {
		left = window
	}

	return result[0], left, nil
}

func (r *Client) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("deleting key %s from redis: %w", keys, err)
	}
	return nil
}

func (r *Client) DelWildcard(ctx context.Context, pattern string) error {
	var cursor uint64

	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scanning keys with pattern %s: %w", pattern, err)
		}

		if len(keys) > 0 {
			if err := r.Del(ctx, keys...); err != nil {
				return fmt.Errorf("deleting keys with pattern %s: %w", pattern, err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (r *Client) Close() error {
	return r.rdb.Close()
}
