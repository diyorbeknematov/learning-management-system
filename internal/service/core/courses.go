package core

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

// VisibleCourse returns the course when the actor may see it: a published
// course is open to everybody, a draft only to its owner and the SuperAdmin.
// For everybody else a draft is "not found".
func VisibleCourse(ctx context.Context, r *repo.Repository, actor models.Actor, courseID uuid.UUID, op string) (*models.Course, error) {
	course, err := r.Course.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	if course.Status != models.CourseStatusPublished && CanManage(actor, course.InstructorID, op) != nil {
		return nil, apperror.NotFound("service", op, "not found", apperror.ErrNotFound)
	}

	return course, nil
}

// ManageableCourse returns the course when the actor may change it: its owner
// or the SuperAdmin.
func ManageableCourse(ctx context.Context, r *repo.Repository, actor models.Actor, courseID uuid.UUID, op string) (*models.Course, error) {
	course, err := r.Course.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	if err := CanManage(actor, course.InstructorID, op); err != nil {
		return nil, err
	}

	return course, nil
}

// ActiveEnrollment returns the enrollment of the student when they study the
// course now (active or completed). A student who never enrolled or who
// dropped it gets "forbidden".
func ActiveEnrollment(ctx context.Context, r *repo.Repository, studentID, courseID uuid.UUID, op string) (*models.Enrollment, error) {
	enrollment, err := r.Enrollment.GetByStudentAndCourse(ctx, studentID, courseID)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperror.Forbidden("service", op, "enroll in the course first", apperror.ErrForbidden)
		}

		return nil, err
	}

	if enrollment.Status == models.EnrollmentStatusDropped {
		return nil, apperror.Forbidden("service", op, "you dropped this course", apperror.ErrForbidden)
	}

	return enrollment, nil
}
