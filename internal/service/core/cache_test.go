package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/stretchr/testify/require"
)

// jsonMemory is a Redis in memory that keeps JSON like the real client and can
// be made to fail.
type jsonMemory struct {
	values map[string][]byte
	counts map[string]int64
	fail   error
}

func newJSONMemory() *jsonMemory {
	return &jsonMemory{values: map[string][]byte{}, counts: map[string]int64{}}
}

func (m *jsonMemory) Get(_ context.Context, key string, dest any) error {
	if m.fail != nil {
		return m.fail
	}

	if count, ok := m.counts[key]; ok {
		return json.Unmarshal([]byte(strconv.FormatInt(count, 10)), dest)
	}

	raw, ok := m.values[key]
	if !ok {
		return apperror.ErrNotFound
	}

	return json.Unmarshal(raw, dest)
}

func (m *jsonMemory) Set(_ context.Context, key string, value any, _ time.Duration) error {
	if m.fail != nil {
		return m.fail
	}

	raw, err := json.Marshal(value)
	m.values[key] = raw

	return err
}

func (m *jsonMemory) Del(context.Context, ...string) error { return nil }

func (m *jsonMemory) GetDel(context.Context, string, any) error { return nil }

func (m *jsonMemory) Incr(_ context.Context, key string, _ time.Duration) (int64, time.Duration, error) {
	if m.fail != nil {
		return 0, 0, m.fail
	}

	m.counts[key]++

	return m.counts[key], 0, nil
}

type page struct {
	Title string `json:"title"`
}

// loader counts how often the database would be asked.
func loader(calls *int, title string, keep bool) func() (*page, bool, error) {
	return func() (*page, bool, error) {
		*calls++

		return &page{Title: title}, keep, nil
	}
}

func TestCached_SecondCallComesFromTheCache(t *testing.T) {
	ctx := context.Background()
	cache := core.NewCache(newJSONMemory())
	calls := 0

	first, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "one", true))
	require.NoError(t, err)

	second, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "two", true))
	require.NoError(t, err)

	require.Equal(t, 1, calls)
	require.Equal(t, "one", first.Title)
	require.Equal(t, "one", second.Title, "the loader was not called again")

	_, err = core.Cached(ctx, cache, core.CacheCourses, "b", loader(&calls, "other", true))
	require.NoError(t, err)
	require.Equal(t, 2, calls, "another key is another entry")
}

func TestCached_DoesNotKeepWhatMustNotBeShared(t *testing.T) {
	ctx := context.Background()
	cache := core.NewCache(newJSONMemory())
	calls := 0

	for range 2 {
		_, err := core.Cached(ctx, cache, core.CacheCourses, "draft", loader(&calls, "draft", false))
		require.NoError(t, err)
	}

	require.Equal(t, 2, calls)
}

func TestCached_InvalidateOnlyTouchesItsScope(t *testing.T) {
	ctx := context.Background()
	cache := core.NewCache(newJSONMemory())
	calls := 0

	for _, scope := range []string{core.CacheCourses, core.CacheCategories} {
		_, err := core.Cached(ctx, cache, scope, "a", loader(&calls, scope, true))
		require.NoError(t, err)
	}

	cache.Invalidate(ctx, core.CacheCourses)

	courses, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "new", true))
	require.NoError(t, err)
	require.Equal(t, "new", courses.Title, "the courses were read again")

	categories, err := core.Cached(ctx, cache, core.CacheCategories, "a", loader(&calls, "new", true))
	require.NoError(t, err)
	require.Equal(t, core.CacheCategories, categories.Title, "the categories stayed")
}

func TestCached_ALoaderErrorIsReturnedAndNotKept(t *testing.T) {
	ctx := context.Background()
	cache := core.NewCache(newJSONMemory())

	boom := errors.New("boom")

	_, err := core.Cached(ctx, cache, core.CacheCourses, "a", func() (*page, bool, error) { return nil, false, boom })
	require.ErrorIs(t, err, boom)

	calls := 0
	value, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "ok", true))
	require.NoError(t, err)
	require.Equal(t, "ok", value.Title)
}

func TestCached_WorksWhenRedisFails(t *testing.T) {
	ctx := context.Background()
	store := newJSONMemory()
	store.fail = errors.New("redis is down")

	cache := core.NewCache(store)
	calls := 0

	for range 2 {
		value, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "db", true))
		require.NoError(t, err)
		require.Equal(t, "db", value.Title)
	}

	require.Equal(t, 2, calls, "every request reads the database")

	require.NotPanics(t, func() { cache.Invalidate(ctx, core.CacheCourses) })
}

func TestCached_NoCacheAtAll(t *testing.T) {
	ctx := context.Background()
	calls := 0

	for _, cache := range []*core.Cache{nil, core.NewCache(nil)} {
		value, err := core.Cached(ctx, cache, core.CacheCourses, "a", loader(&calls, "db", true))
		require.NoError(t, err)
		require.Equal(t, "db", value.Title)

		require.NotPanics(t, func() { cache.Invalidate(ctx, core.CacheCourses) })
	}

	require.Equal(t, 2, calls)
}

func TestCacheKey(t *testing.T) {
	type filter struct {
		Search string
		Page   int
	}

	a := core.CacheKey("list", filter{"go", 1})

	require.Equal(t, a, core.CacheKey("list", filter{"go", 1}), "the same parameters, the same key")
	require.NotEqual(t, a, core.CacheKey("list", filter{"go", 2}))
	require.NotEqual(t, a, core.CacheKey("other", filter{"go", 1}))
}
