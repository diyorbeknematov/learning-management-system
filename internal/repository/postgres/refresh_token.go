package postgres

import (
	"context"
	"errors"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type refreshTokenRepo struct {
	db DBTX
}

func NewRefreshTokenRepository(db DBTX) *refreshTokenRepo {
	return &refreshTokenRepo{
		db: db,
	}
}

func (r *refreshTokenRepo) Create(
	ctx context.Context,
	token models.RefreshToken,
) (uuid.UUID, error) {
	query := `
		INSERT INTO refresh_tokens (
			id,
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3, $4);
	`

	id := uuid.New()

	_, err := r.db.Exec(
		ctx,
		query,
		id,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	)

	if err != nil {
		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateRefreshToken",
			"failed to create refresh token",
			err,
		)
	}

	return id, nil
}

func (r *refreshTokenRepo) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*models.RefreshToken, error) {
	query := `
		SELECT
			t.id,
			t.user_id,
			t.token_hash,
			t.expires_at,
			t.created_at
		FROM refresh_tokens t
		INNER JOIN users u
			ON t.user_id = u.id
		WHERE t.token_hash = $1
			AND u.deleted_at IS NULL;
	`

	var token models.RefreshToken

	err := r.db.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.Unauthorized(
				"repository",
				"GetRefreshTokenByTokenHash",
				"refresh token not found",
				apperror.ErrUnauthorized,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetRefreshTokenByTokenHash",
			"failed to get refresh token by token hash",
			err,
		)
	}

	return &token, nil
}

func (r *refreshTokenRepo) DeleteByTokenHash(
	ctx context.Context,
	tokenHash string,
) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE token_hash = $1;
	`

	res, err := r.db.Exec(
		ctx,
		query,
		tokenHash,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteRefreshTokenByTokenHash",
			"failed to delete refresh token by token hash",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteRefreshTokenByTokenHash",
			"token not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *refreshTokenRepo) DeleteByUserID(
	ctx context.Context,
	userID uuid.UUID,
) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1;
	`

	res, err := r.db.Exec(
		ctx,
		query,
		userID,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteRefreshTokenByUserID",
			"failed to delete refresh tokens by user id",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteRefreshTokenByUserID",
			"tokens not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
