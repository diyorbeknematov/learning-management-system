package postgres

import (
	"errors"

	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// handleError turns a database error into an apperror. op names the
// repository method and message describes what failed; message is used for
// errors that are not the caller's fault. The error is not logged here: the
// handler logs it once, with the layer and op taken from the apperror.
func handleError(err error, op, message string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NotFound("repository", op, "not found", apperror.ErrNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperror.Conflict("repository", op, "already exists", apperror.ErrAlreadyExists)
		case "23503": // foreign_key_violation: the related row does not exist
			return apperror.NotFound("repository", op, "related record not found", apperror.ErrNotFound)
		case "23514", "23502": // check_violation, not_null_violation
			return apperror.InvalidInput("repository", op, "invalid value for one of the fields", apperror.ErrInvalidInput)
		}
	}

	return apperror.Internal("repository", op, message, err)
}
