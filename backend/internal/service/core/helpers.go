package core

import (
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/google/uuid"
)

// CanManage allows the owner of a resource and the SuperAdmin. Casbin has
// already checked that the role may perform the action at all; this checks
// that the user may do it on this particular resource.
func CanManage(actor models.Actor, ownerID uuid.UUID, op string) error {
	if actor.IsSuperAdmin() || actor.UserID == ownerID {
		return nil
	}

	return apperror.Forbidden(
		"service",
		op,
		"you do not have access to this resource",
		apperror.ErrForbidden,
	)
}

// NewListResponse wraps a page of items with the paging values that were
// really applied (defaults and the maximum limit included).
func NewListResponse[T any](items []T, total, page, limit int) models.ListResponse[T] {
	limit, _ = helpers.Pagination(page, limit)

	if page <= 0 {
		page = 1
	}

	return models.ListResponse[T]{
		Items: items,
		Total: total,
		Page:  page,
		Limit: limit,
	}
}

func IsNotFound(err error) bool {
	appErr, ok := apperror.As(err)

	return ok && appErr.Code == apperror.CodeNotFound
}

// RequireSuperAdmin lets only the SuperAdmin through. Casbin already keeps
// other roles away from the finance and report endpoints; this is the second
// lock, so a mistake in the policy cannot open the money to everybody.
func RequireSuperAdmin(actor models.Actor, op string) error {
	if actor.IsSuperAdmin() {
		return nil
	}

	return apperror.Forbidden("service", op, "only the SuperAdmin can do this", apperror.ErrForbidden)
}
