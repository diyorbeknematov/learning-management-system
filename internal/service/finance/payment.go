package finance

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Payment struct {
	repo *repo.Repository
}

func NewPayment(deps core.Dependencies) *Payment {
	return &Payment{
		repo: deps.Repo,
	}
}

// GetList lists the payments for the SuperAdmin, with the period and the course
// as filters.
func (s *Payment) GetList(ctx context.Context, actor models.Actor, filter models.PaymentFilter) (*models.ListResponse[models.Payment], error) {
	if err := core.RequireSuperAdmin(actor, "GetPaymentList"); err != nil {
		return nil, err
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, apperror.InvalidInput("service", "GetPaymentList", "from must not be after to", apperror.ErrInvalidInput)
	}

	payments, total, err := s.repo.Payment.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}

	response := core.NewListResponse(payments, total, filter.Page, filter.Limit)

	return &response, nil
}

// canSee allows the SuperAdmin and the student who paid. The instructor of the
// course has no access to the money.
func (s *Payment) canSee(ctx context.Context, actor models.Actor, enrollmentID uuid.UUID, op string) error {
	if actor.IsSuperAdmin() {
		return nil
	}

	enrollment, err := s.repo.Enrollment.GetByID(ctx, enrollmentID)
	if err != nil {
		return err
	}

	if enrollment.StudentID != actor.UserID {
		return apperror.Forbidden("service", op, "you do not have access to this payment", apperror.ErrForbidden)
	}

	return nil
}

func (s *Payment) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.Payment, error) {
	payment, err := s.repo.Payment.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.canSee(ctx, actor, payment.EnrollmentID, "GetPayment"); err != nil {
		return nil, err
	}

	return payment, nil
}

// GetByEnrollment returns the payment of an enrollment. A free course has no
// payment, so that is "not found".
func (s *Payment) GetByEnrollment(ctx context.Context, actor models.Actor, enrollmentID uuid.UUID) (*models.Payment, error) {
	if err := s.canSee(ctx, actor, enrollmentID, "GetEnrollmentPayment"); err != nil {
		return nil, err
	}

	return s.repo.Payment.GetByEnrollmentID(ctx, enrollmentID)
}
