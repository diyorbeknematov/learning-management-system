package account

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/google/uuid"
)

// deleteSessions removes every refresh token of the user. A user without any
// token is not an error.
func deleteSessions(ctx context.Context, r *repo.Repository, userID uuid.UUID) error {
	err := r.RefreshToken.DeleteByUserID(ctx, userID)
	if err != nil && !core.IsNotFound(err) {
		return err
	}

	return nil
}
