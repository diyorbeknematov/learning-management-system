package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/diyorbeknematov/lms/pkg/apperror"
)

const (
	// CacheCourses is the scope of the public catalog: course lists and pages.
	CacheCourses = "courses"
	// CacheCategories is the scope of the categories.
	CacheCategories = "categories"

	// cacheTTL is how long an entry lives. It is shorter than the time a
	// presigned file link is valid, because the entries hold such links.
	cacheTTL = 5 * time.Minute
	// versionLifetime is how long the version of a scope lives; entries live
	// far shorter, so a version that starts again from 1 meets no old entry.
	versionLifetime = 24 * time.Hour
)

// Cache keeps answers that are the same for everybody (the public catalog) in
// Redis for a few minutes. It never breaks a request: when Redis fails the
// answer is read from the database as without a cache.
//
// Entries are not deleted when the data changes. Every scope has a version
// number inside the key of its entries; Invalidate raises the version, so old
// entries are not found any more and expire by themselves.
type Cache struct {
	redis Redis
}

// NewCache returns a cache on top of Redis; with nil Redis it does nothing.
func NewCache(redis Redis) *Cache {
	return &Cache{redis: redis}
}

// missing tells that Redis has no such key (the client answers with the plain
// ErrNotFound).
func missing(err error) bool {
	return errors.Is(err, apperror.ErrNotFound) || IsNotFound(err)
}

func versionKey(scope string) string {
	return "cache_version:" + scope
}

// version is the current version of a scope; a scope that was never changed
// (or whose version expired) is at 0.
func (c *Cache) version(ctx context.Context, scope string) (int64, error) {
	var version int64

	if err := c.redis.Get(ctx, versionKey(scope), &version); err != nil {
		if missing(err) {
			return 0, nil
		}

		return 0, err
	}

	return version, nil
}

// Invalidate makes everything cached in the scope unusable. Call it after the
// data of the scope was changed.
func (c *Cache) Invalidate(ctx context.Context, scope string) {
	if c == nil || c.redis == nil {
		return
	}

	if _, _, err := c.redis.Incr(ctx, versionKey(scope), versionLifetime); err != nil {
		slog.WarnContext(ctx, "the cache could not be invalidated", "scope", scope, "error", err)
	}
}

// CacheKey makes a short key from the parameters of a request (a filter).
func CacheKey(prefix string, params any) string {
	raw, err := json.Marshal(params)
	if err != nil {
		return prefix + ":" + "unkeyed"
	}

	sum := sha256.Sum256(raw)

	return prefix + ":" + hex.EncodeToString(sum[:8])
}

// Cached returns the entry of key in the scope, or calls load and keeps its
// result. load says whether the result may be kept: an answer that depends on
// who asks must not be shared.
func Cached[T any](ctx context.Context, c *Cache, scope, key string, load func() (*T, bool, error)) (*T, error) {
	if c == nil || c.redis == nil {
		value, _, err := load()

		return value, err
	}

	version, err := c.version(ctx, scope)
	if err != nil {
		slog.WarnContext(ctx, "the cache could not be read", "scope", scope, "error", err)

		value, _, loadErr := load()

		return value, loadErr
	}

	entry := "cache:" + scope + ":v" + strconv.FormatInt(version, 10) + ":" + key

	var cached T

	if err := c.redis.Get(ctx, entry, &cached); err == nil {
		return &cached, nil
	} else if !missing(err) {
		slog.WarnContext(ctx, "the cache could not be read", "scope", scope, "error", err)
	}

	value, keep, err := load()
	if err != nil {
		return nil, err
	}

	if keep {
		if err := c.redis.Set(ctx, entry, value, cacheTTL); err != nil {
			slog.WarnContext(ctx, "the cache could not be written", "scope", scope, "error", err)
		}
	}

	return value, nil
}
