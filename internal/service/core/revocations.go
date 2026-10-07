package core

import (
	"context"
	"errors"
	"time"

	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

const defaultRevocationTTL = time.Hour

// Revocations lets the API reject the access tokens of a user at once, for
// example after the user is blocked.
//
// An access token is not checked against the database (that is what makes it
// fast), so it cannot be taken back by itself. Instead, the time of the
// change is written to Redis, and the API rejects every token of that user that
// was issued before it. The note lives as long as an access token lives: after
// that every old token is dead anyway.
type Revocations struct {
	redis Redis
	ttl   time.Duration
}

func NewRevocations(redis Redis, accessTokenTTL time.Duration) *Revocations {
	if accessTokenTTL <= 0 {
		accessTokenTTL = defaultRevocationTTL
	}

	return &Revocations{
		redis: redis,
		ttl:   accessTokenTTL,
	}
}

func revocationKey(userID uuid.UUID) string {
	return "revoked_user:" + userID.String()
}

// Revoke makes every access token the user has now useless. Tokens the user
// gets later work.
func (r *Revocations) Revoke(ctx context.Context, userID uuid.UUID) error {
	err := r.redis.Set(ctx, revocationKey(userID), time.Now().UnixMilli(), r.ttl)
	if err != nil {
		return apperror.Internal("service", "RevokeTokens", "failed to revoke the access tokens", err)
	}

	return nil
}

// Reinstate removes the note, for example when a blocked user is unblocked.
func (r *Revocations) Reinstate(ctx context.Context, userID uuid.UUID) error {
	if err := r.redis.Del(ctx, revocationKey(userID)); err != nil {
		return apperror.Internal("service", "ReinstateTokens", "failed to remove the revocation", err)
	}

	return nil
}

// IsRevoked tells whether a token, issued at issuedAt, was revoked.
//
// A token carries its time in whole seconds, so a token of the second 10:00:05
// may really have been made at any moment of that second. It counts as revoked
// only when the whole second was over before the revocation; a token made in
// the same second as the revocation is let through (that is a user who logged
// in a moment before being blocked, and the refresh token is gone anyway).
func (r *Revocations) IsRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	var revokedAt int64

	err := r.redis.Get(ctx, revocationKey(userID), &revokedAt)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return !issuedAt.Add(time.Second).After(time.UnixMilli(revokedAt)), nil
}
