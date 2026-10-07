package learning

import (
	"context"
	"math"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Enrollment struct {
	repo    *repo.Repository
	storage core.Storage
}

func NewEnrollment(deps core.Dependencies) *Enrollment {
	return &Enrollment{
		repo:    deps.Repo,
		storage: deps.Storage,
	}
}

// Enroll signs the student up for a published course. A paid course is paid
// at once (the payment is emulated): the enrollment, the payment and the
// instructor's share are saved together. A student who dropped the course
// comes back to it without paying again.
func (s *Enrollment) Enroll(ctx context.Context, actor models.Actor, courseID uuid.UUID) (*models.Enrollment, error) {
	course, err := core.VisibleCourse(ctx, s.repo, models.Actor{}, courseID, "Enroll")
	if err != nil {
		return nil, err
	}

	if actor.UserID == course.InstructorID {
		return nil, apperror.Forbidden("service", "Enroll", "you cannot enroll in your own course", apperror.ErrForbidden)
	}

	existing, err := s.repo.Enrollment.GetByStudentAndCourse(ctx, actor.UserID, courseID)
	if err != nil && !core.IsNotFound(err) {
		return nil, err
	}

	if err == nil {
		if existing.Status != models.EnrollmentStatusDropped {
			return nil, apperror.Conflict("service", "Enroll", "you are already enrolled in this course", apperror.ErrAlreadyExists)
		}

		err := s.repo.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{
			ID:     existing.ID,
			Status: models.EnrollmentStatusActive,
		})
		if err != nil {
			return nil, err
		}

		return s.repo.Enrollment.GetByID(ctx, existing.ID)
	}

	var id uuid.UUID

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		id, err = tx.Enrollment.Create(ctx, models.CreateEnrollment{CourseID: courseID, StudentID: actor.UserID})
		if err != nil {
			return err
		}

		if course.Price <= 0 {
			return nil
		}

		_, err = tx.Payment.Create(ctx, models.CreatePayment{
			EnrollmentID: id,
			Amount:       course.Price,
			PaidAt:       time.Now(),
		})
		if err != nil {
			return err
		}

		return createPayout(ctx, tx, course, id)
	})
	if err != nil {
		return nil, err
	}

	return s.repo.Enrollment.GetByID(ctx, id)
}

// createPayout records the instructor's share of an enrollment. Without a
// payout set on the course (the SuperAdmin sets it) there is no share.
func createPayout(ctx context.Context, tx *repo.Repository, course *models.Course, enrollmentID uuid.UUID) error {
	if course.PayoutType == nil || course.PayoutValue == nil {
		return nil
	}

	amount := payoutAmount(*course.PayoutType, *course.PayoutValue, course.Price)

	_, err := tx.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     course.ID,
		EnrollmentID: enrollmentID,
		Type:         *course.PayoutType,
		Value:        *course.PayoutValue,
		Amount:       amount,
	})

	return err
}

// payoutAmount is a percentage of the price or a fixed sum, never more than the
// price, rounded to cents.
func payoutAmount(payoutType models.PayoutType, value, price float64) float64 {
	amount := value

	if payoutType == models.PayoutTypePercentage {
		amount = price * value / 100
	}

	return math.Round(math.Min(amount, price)*100) / 100
}

// GetByID shows an enrollment to the student, the owner of the course and the
// SuperAdmin.
func (s *Enrollment) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.Enrollment, error) {
	enrollment, err := s.repo.Enrollment.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.canSee(ctx, actor, enrollment, "GetEnrollment"); err != nil {
		return nil, err
	}

	return enrollment, nil
}

// canSee allows the student of the enrollment, the owner of the course and the
// SuperAdmin.
func (s *Enrollment) canSee(ctx context.Context, actor models.Actor, enrollment *models.Enrollment, op string) error {
	if actor.UserID == enrollment.StudentID {
		return nil
	}

	course, err := s.repo.Course.GetByID(ctx, enrollment.CourseID)
	if err != nil {
		return err
	}

	return core.CanManage(actor, course.InstructorID, op)
}

// GetListByCourse is the roster of a course, for its owner and the SuperAdmin.
func (s *Enrollment) GetListByCourse(
	ctx context.Context,
	actor models.Actor,
	courseID uuid.UUID,
	filter models.EnrollmentFilter,
) (*models.ListResponse[models.EnrollmentRosterItem], error) {
	if _, err := core.ManageableCourse(ctx, s.repo, actor, courseID, "GetEnrollmentList"); err != nil {
		return nil, err
	}

	items, total, err := s.repo.Enrollment.GetListByCourseID(ctx, courseID, filter)
	if err != nil {
		return nil, err
	}

	response := core.NewListResponse(items, total, filter.Page, filter.Limit)

	return &response, nil
}

// GetMyList is the dashboard of the student: the courses with the progress.
func (s *Enrollment) GetMyList(
	ctx context.Context,
	actor models.Actor,
	filter models.EnrollmentFilter,
) (*models.ListResponse[models.MyEnrollment], error) {
	items, total, err := s.repo.Enrollment.GetMyList(ctx, actor.UserID, filter)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].CourseCoverURL, err = core.DownloadURL(ctx, s.storage, items[i].CourseCover)
		if err != nil {
			return nil, err
		}
	}

	response := core.NewListResponse(items, total, filter.Page, filter.Limit)

	return &response, nil
}

// UpdateStatus lets a student leave a course, and the owner of the course or
// the SuperAdmin remove a student. The only status that can be set by hand is
// dropped: completed comes from the progress.
func (s *Enrollment) UpdateStatus(ctx context.Context, actor models.Actor, req models.UpdateEnrollmentStatus) error {
	if req.Status != models.EnrollmentStatusDropped {
		return apperror.InvalidInput(
			"service",
			"UpdateEnrollmentStatus",
			"only dropped can be set; completed is set when all lessons are done",
			apperror.ErrInvalidInput,
		)
	}

	enrollment, err := s.repo.Enrollment.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}

	if err := s.canSee(ctx, actor, enrollment, "UpdateEnrollmentStatus"); err != nil {
		return err
	}

	return s.repo.Enrollment.UpdateStatus(ctx, req)
}
