package learning

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Review struct {
	repo *repo.Repository
}

func NewReview(deps core.Dependencies) *Review {
	return &Review{
		repo: deps.Repo,
	}
}

// Create adds the review of a student who completed the course; each student
// writes one review per course.
func (s *Review) Create(ctx context.Context, actor models.Actor, courseID uuid.UUID, req models.CreateReview) (*models.Review, error) {
	if _, err := core.VisibleCourse(ctx, s.repo, models.Actor{}, courseID, "CreateReview"); err != nil {
		return nil, err
	}

	enrollment, err := s.repo.Enrollment.GetByStudentAndCourse(ctx, actor.UserID, courseID)
	if err != nil {
		if core.IsNotFound(err) {
			return nil, apperror.Forbidden("service", "CreateReview", "complete the course to review it", apperror.ErrForbidden)
		}

		return nil, err
	}

	if enrollment.Status != models.EnrollmentStatusCompleted {
		return nil, apperror.Forbidden("service", "CreateReview", "complete the course to review it", apperror.ErrForbidden)
	}

	req.StudentID, req.CourseID = actor.UserID, courseID

	id, err := s.repo.Review.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.repo.Review.GetByID(ctx, id)
}

// GetListByCourse returns the reviews of a published course with the average
// rating.
func (s *Review) GetListByCourse(
	ctx context.Context,
	actor models.Actor,
	courseID uuid.UUID,
	filter models.ReviewFilter,
) (*models.ReviewList, error) {
	if _, err := core.VisibleCourse(ctx, s.repo, actor, courseID, "GetReviewList"); err != nil {
		return nil, err
	}

	reviews, total, err := s.repo.Review.GetListByCourseID(ctx, courseID, filter)
	if err != nil {
		return nil, err
	}

	average, err := s.repo.Review.GetAverageRating(ctx, courseID)
	if err != nil {
		return nil, err
	}

	return &models.ReviewList{
		ListResponse: core.NewListResponse(reviews, total, filter.Page, filter.Limit),
		AvgRating:    average,
	}, nil
}

// Update changes a review; only its author can.
func (s *Review) Update(ctx context.Context, actor models.Actor, id uuid.UUID, req models.UpdateReview) (*models.Review, error) {
	review, err := s.repo.Review.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if review.StudentID != actor.UserID {
		return nil, apperror.Forbidden("service", "UpdateReview", "you can change only your own review", apperror.ErrForbidden)
	}

	req.ID = id

	return s.repo.Review.Update(ctx, req)
}

// Delete removes a review: its author can, and the SuperAdmin, who moderates.
func (s *Review) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	review, err := s.repo.Review.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if review.StudentID != actor.UserID && !actor.IsSuperAdmin() {
		return apperror.Forbidden("service", "DeleteReview", "you can delete only your own review", apperror.ErrForbidden)
	}

	return s.repo.Review.Delete(ctx, id)
}
