package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestRefreshToken(t *testing.T, tc *TestContext) (uuid.UUID, string) {
	t.Helper()

	ctx := context.Background()

	userID := createTestUser(t, tc)
	hash := "hash_" + uuid.NewString()

	_, err := tc.Repo.RefreshToken.Create(ctx, models.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
		deleteTestUser(t, tc, userID)
	})

	return userID, hash
}

func TestRefreshTokenRepo_Create(t *testing.T) {
	tc := setupTest(t)

	userID, hash := createTestRefreshToken(t, tc)

	var count int
	err := tc.DB.Pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2`,
		userID,
		hash,
	).Scan(&count)

	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestRefreshTokenRepo_GetByTokenHash(t *testing.T) {
	tc := setupTest(t)

	userID, hash := createTestRefreshToken(t, tc)

	token, err := tc.Repo.RefreshToken.GetByTokenHash(context.Background(), hash)

	require.NoError(t, err)
	require.Equal(t, userID, token.UserID)
	require.Equal(t, hash, token.TokenHash)
}

func TestRefreshTokenRepo_GetByTokenHash_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.RefreshToken.GetByTokenHash(context.Background(), "missing_"+uuid.NewString())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestRefreshTokenRepo_DeleteByTokenHash(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	_, hash := createTestRefreshToken(t, tc)

	require.NoError(t, tc.Repo.RefreshToken.DeleteByTokenHash(ctx, hash))

	_, err := tc.Repo.RefreshToken.GetByTokenHash(ctx, hash)
	require.Error(t, err)
}

func TestRefreshTokenRepo_DeleteByUserID(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	userID, hash := createTestRefreshToken(t, tc)

	require.NoError(t, tc.Repo.RefreshToken.DeleteByUserID(ctx, userID))

	_, err := tc.Repo.RefreshToken.GetByTokenHash(ctx, hash)
	require.Error(t, err)
}
