package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// memory is a Redis in memory, only what Revocations needs.
type memory struct {
	values map[string]int64
	ttls   map[string]time.Duration
}

func newMemory() *memory {
	return &memory{values: map[string]int64{}, ttls: map[string]time.Duration{}}
}

func (m *memory) Get(_ context.Context, key string, dest any) error {
	value, ok := m.values[key]
	if !ok {
		return apperror.ErrNotFound
	}

	*(dest.(*int64)) = value

	return nil
}

func (m *memory) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	m.values[key] = value.(int64)
	m.ttls[key] = ttl

	return nil
}

func (m *memory) Del(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(m.values, key)
	}

	return nil
}

func (m *memory) GetDel(context.Context, string, any) error { return nil }

func (m *memory) Incr(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 0, 0, nil
}

func TestRevocations(t *testing.T) {
	ctx := context.Background()
	store := newMemory()
	revocations := core.NewRevocations(store, 15*time.Minute)

	user, other := uuid.New(), uuid.New()
	longAgo := time.Now().Add(-time.Hour)

	revoked, err := revocations.IsRevoked(ctx, user, longAgo)
	require.NoError(t, err)
	require.False(t, revoked, "nothing was revoked")

	require.NoError(t, revocations.Revoke(ctx, user))

	revoked, err = revocations.IsRevoked(ctx, user, longAgo)
	require.NoError(t, err)
	require.True(t, revoked, "a token from long ago is dead")

	revoked, err = revocations.IsRevoked(ctx, other, longAgo)
	require.NoError(t, err)
	require.False(t, revoked, "another user is not touched")

	revoked, err = revocations.IsRevoked(ctx, user, time.Now().Add(2*time.Second))
	require.NoError(t, err)
	require.False(t, revoked, "a token made after the revocation works")

	for _, ttl := range store.ttls {
		require.Equal(t, 15*time.Minute, ttl, "the note is kept as long as an access token lives")
	}
}

func TestRevocations_ATokenOfTheSameSecondIsLetThrough(t *testing.T) {
	ctx := context.Background()
	revocations := core.NewRevocations(newMemory(), time.Minute)

	user := uuid.New()

	// a token carries only whole seconds: this one was made in the second of the revocation
	issued := time.Now().Truncate(time.Second)

	require.NoError(t, revocations.Revoke(ctx, user))

	revoked, err := revocations.IsRevoked(ctx, user, issued)
	require.NoError(t, err)
	require.False(t, revoked)

	// but one that is a full second older is dead
	revoked, err = revocations.IsRevoked(ctx, user, issued.Add(-time.Second))
	require.NoError(t, err)
	require.True(t, revoked)
}

func TestRevocations_Reinstate(t *testing.T) {
	ctx := context.Background()
	revocations := core.NewRevocations(newMemory(), time.Minute)

	user := uuid.New()
	longAgo := time.Now().Add(-time.Hour)

	require.NoError(t, revocations.Revoke(ctx, user))
	require.NoError(t, revocations.Reinstate(ctx, user))

	revoked, err := revocations.IsRevoked(ctx, user, longAgo)
	require.NoError(t, err)
	require.False(t, revoked)
}

func TestRevocations_NoTTLGivenStillWorks(t *testing.T) {
	store := newMemory()
	revocations := core.NewRevocations(store, 0)

	require.NoError(t, revocations.Revoke(context.Background(), uuid.New()))

	for _, ttl := range store.ttls {
		require.Greater(t, ttl, time.Duration(0), "a note must always expire")
	}
}
