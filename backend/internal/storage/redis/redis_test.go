package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/internal/storage/redis"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// setupRedis connects to the Redis from the environment. The test is skipped
// when Redis is not reachable. It returns a unique key prefix so tests do not
// touch each other's keys.
func setupRedis(t *testing.T) (*redis.Client, string) {
	t.Helper()

	client, err := redis.New(config.Load().Redis)
	if err != nil {
		t.Skipf("redis is not available: %v", err)
	}

	prefix := "test:" + uuid.NewString() + ":"

	t.Cleanup(func() {
		_ = client.DelWildcard(context.Background(), prefix+"*")
		_ = client.Close()
	})

	return client, prefix
}

func TestSetAndGet(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "course"
	want := payload{Name: "Go", Count: 3}

	require.NoError(t, client.Set(ctx, key, want, time.Minute))

	var got payload
	require.NoError(t, client.Get(ctx, key, &got))
	require.Equal(t, want, got)
}

func TestGet_Missing(t *testing.T) {
	client, prefix := setupRedis(t)

	var got payload
	err := client.Get(context.Background(), prefix+"missing", &got)

	require.True(t, errors.Is(err, apperror.ErrNotFound))
}

func TestGet_InvalidJSON(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "text"

	require.NoError(t, client.Set(ctx, key, "just a string", time.Minute))

	var got payload
	err := client.Get(ctx, key, &got)

	require.Error(t, err)
	require.False(t, errors.Is(err, apperror.ErrNotFound))
}

func TestSet_InvalidTTL(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()

	require.Error(t, client.Set(ctx, prefix+"zero", "x", 0))
	require.Error(t, client.Set(ctx, prefix+"negative", "x", -time.Second))
}

func TestSet_Expires(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "short"

	require.NoError(t, client.Set(ctx, key, "x", time.Second))
	time.Sleep(1200 * time.Millisecond)

	var got string
	err := client.Get(ctx, key, &got)

	require.True(t, errors.Is(err, apperror.ErrNotFound))
}

func TestSet_Overwrites(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "value"

	require.NoError(t, client.Set(ctx, key, "first", time.Minute))
	require.NoError(t, client.Set(ctx, key, "second", time.Minute))

	var got string
	require.NoError(t, client.Get(ctx, key, &got))
	require.Equal(t, "second", got)
}

func TestGetDel(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "reset"
	userID := uuid.New()

	require.NoError(t, client.Set(ctx, key, userID, time.Minute))

	var got uuid.UUID
	require.NoError(t, client.GetDel(ctx, key, &got))
	require.Equal(t, userID, got)

	err := client.GetDel(ctx, key, &got)
	require.True(t, errors.Is(err, apperror.ErrNotFound))
}

func TestGetDel_OnlyOneWins(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "once"

	require.NoError(t, client.Set(ctx, key, "token", time.Minute))

	const workers = 20

	results := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func() {
			var got string
			results <- client.GetDel(ctx, key, &got)
		}()
	}

	succeeded := 0
	for i := 0; i < workers; i++ {
		if err := <-results; err == nil {
			succeeded++
		}
	}

	require.Equal(t, 1, succeeded)
}

func TestDel(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	first, second, kept := prefix+"a", prefix+"b", prefix+"c"

	for _, key := range []string{first, second, kept} {
		require.NoError(t, client.Set(ctx, key, "x", time.Minute))
	}

	require.NoError(t, client.Del(ctx, first, second))

	var got string
	require.True(t, errors.Is(client.Get(ctx, first, &got), apperror.ErrNotFound))
	require.True(t, errors.Is(client.Get(ctx, second, &got), apperror.ErrNotFound))
	require.NoError(t, client.Get(ctx, kept, &got))
}

func TestDel_NoKeysAndMissingKey(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()

	require.NoError(t, client.Del(ctx))
	require.NoError(t, client.Del(ctx, prefix+"missing"))
}

func TestDelWildcard(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()

	for _, key := range []string{"courses:1", "courses:2", "courses:list:page1", "users:1"} {
		require.NoError(t, client.Set(ctx, prefix+key, "x", time.Minute))
	}

	require.NoError(t, client.DelWildcard(ctx, prefix+"courses:*"))

	var got string
	for _, key := range []string{"courses:1", "courses:2", "courses:list:page1"} {
		require.True(t, errors.Is(client.Get(ctx, prefix+key, &got), apperror.ErrNotFound), key)
	}

	require.NoError(t, client.Get(ctx, prefix+"users:1", &got))
}

func TestDelWildcard_ManyKeys(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()

	// more keys than one SCAN page (100) so the cursor loop is used
	keys := make([]string, 250)
	for i := range keys {
		keys[i] = prefix + "bulk:" + uuid.NewString()
		require.NoError(t, client.Set(ctx, keys[i], "x", time.Minute))
	}

	require.NoError(t, client.DelWildcard(ctx, prefix+"bulk:*"))

	var got string
	for _, key := range keys {
		require.True(t, errors.Is(client.Get(ctx, key, &got), apperror.ErrNotFound), key)
	}
}

func TestNew_Unreachable(t *testing.T) {
	cfg := config.Load().Redis
	cfg.Host = "127.0.0.1"
	cfg.Port = 1

	_, err := redis.New(cfg)

	require.Error(t, err)
}

func TestIncr_CountsHitsInAWindow(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "hits"

	for want := int64(1); want <= 5; want++ {
		count, left, err := client.Incr(ctx, key, time.Minute)
		require.NoError(t, err)
		require.Equal(t, want, count)
		require.Greater(t, left, time.Duration(0))
		require.LessOrEqual(t, left, time.Minute)
	}

	var stored int

	require.NoError(t, client.Get(ctx, key, &stored), "a counter is read like any value")
	require.Equal(t, 5, stored)
}

func TestIncr_KeysCountSeparately(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _, err := client.Incr(ctx, prefix+"a", time.Minute)
		require.NoError(t, err)
	}

	count, _, err := client.Incr(ctx, prefix+"b", time.Minute)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestIncr_TheWindowEnds(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "short"

	for i := 0; i < 3; i++ {
		_, _, err := client.Incr(ctx, key, 300*time.Millisecond)
		require.NoError(t, err)
	}

	time.Sleep(400 * time.Millisecond)

	count, _, err := client.Incr(ctx, key, 300*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, int64(1), count, "a new window starts after the old one ends")
}

func TestIncr_TheWindowDoesNotMoveWithEveryHit(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "fixed"

	_, first, err := client.Incr(ctx, key, time.Second)
	require.NoError(t, err)

	time.Sleep(300 * time.Millisecond)

	_, second, err := client.Incr(ctx, key, time.Second)
	require.NoError(t, err)
	require.Less(t, second, first, "the window runs out even when hits keep coming")
}

func TestIncr_ManyAtOnceCountExactly(t *testing.T) {
	client, prefix := setupRedis(t)

	ctx := context.Background()
	key := prefix + "parallel"

	const workers = 100

	results := make(chan int64, workers)

	for i := 0; i < workers; i++ {
		go func() {
			count, _, err := client.Incr(ctx, key, time.Minute)
			if err != nil {
				count = -1
			}

			results <- count
		}()
	}

	seen := map[int64]bool{}

	for i := 0; i < workers; i++ {
		seen[<-results] = true
	}

	require.Len(t, seen, workers, "every hit gets its own number")
	require.True(t, seen[1] && seen[workers])
}

func TestIncr_InvalidWindow(t *testing.T) {
	client, prefix := setupRedis(t)

	_, _, err := client.Incr(context.Background(), prefix+"x", 0)
	require.Error(t, err)

	_, _, err = client.Incr(context.Background(), prefix+"x", -time.Second)
	require.Error(t, err)
}
