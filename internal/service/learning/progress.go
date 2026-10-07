package learning

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/google/uuid"
)

type Progress struct {
	repo *repo.Repository
	cfg  core.Config
}

func NewProgress(deps core.Dependencies) *Progress {
	return &Progress{
		repo: deps.Repo,
		cfg:  deps.Config,
	}
}

// SetLessonProgress marks a lesson as done or not done. When the last lesson
// is done (and the final quiz, if there is one, is passed) the course is
// completed and the certificate is issued; when a lesson of a completed course
// is taken back, the enrollment is active again.
func (s *Progress) SetLessonProgress(
	ctx context.Context,
	actor models.Actor,
	lessonID uuid.UUID,
	completed bool,
) (*models.LessonProgress, error) {
	lesson, err := s.repo.Lesson.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	module, err := s.repo.Module.GetByID(ctx, lesson.ModuleID)
	if err != nil {
		return nil, err
	}

	course, err := core.VisibleCourse(ctx, s.repo, models.Actor{}, module.CourseID, "SetLessonProgress")
	if err != nil {
		return nil, err
	}

	_, err = core.ActiveEnrollment(ctx, s.repo, actor.UserID, course.ID, "SetLessonProgress")
	if err != nil {
		return nil, err
	}

	var progress *models.LessonProgress

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		progress, err = tx.Progress.Set(ctx, models.SetLessonProgress{
			StudentID: actor.UserID,
			LessonID:  lessonID,
			Completed: completed,
		})
		if err != nil {
			return err
		}

		return syncCompletion(ctx, tx, s.cfg, actor.UserID, course)
	})
	if err != nil {
		return nil, err
	}

	return progress, nil
}

// GetLessonProgress is the state of a lesson for the student. A lesson that
// was never touched is "not completed".
func (s *Progress) GetLessonProgress(ctx context.Context, actor models.Actor, lessonID uuid.UUID) (*models.LessonProgress, error) {
	if _, err := s.repo.Lesson.GetByID(ctx, lessonID); err != nil {
		return nil, err
	}

	progress, err := s.repo.Progress.GetByLesson(ctx, actor.UserID, lessonID)
	if err != nil {
		if core.IsNotFound(err) {
			return &models.LessonProgress{StudentID: actor.UserID, LessonID: lessonID}, nil
		}

		return nil, err
	}

	return progress, nil
}

// GetCourseProgress is the progress of the student in a course they study.
func (s *Progress) GetCourseProgress(ctx context.Context, actor models.Actor, courseID uuid.UUID) (*models.CourseProgress, error) {
	if _, err := core.ActiveEnrollment(ctx, s.repo, actor.UserID, courseID, "GetCourseProgress"); err != nil {
		return nil, err
	}

	return s.courseProgress(ctx, actor.UserID, courseID)
}

func (s *Progress) courseProgress(ctx context.Context, studentID, courseID uuid.UUID) (*models.CourseProgress, error) {
	progress, err := s.repo.Progress.GetCourseProgress(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}

	progress.CompletedLessonIDs, err = s.repo.Progress.GetCompletedLessonIDs(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}

	return progress, nil
}

// GetStudents lists the students of a course with their progress, for the
// owner of the course and the SuperAdmin.
func (s *Progress) GetStudents(
	ctx context.Context,
	actor models.Actor,
	courseID uuid.UUID,
	filter models.EnrollmentFilter,
) (*models.ListResponse[models.StudentProgress], error) {
	if _, err := core.ManageableCourse(ctx, s.repo, actor, courseID, "GetStudents"); err != nil {
		return nil, err
	}

	students, total, err := s.repo.Progress.GetStudentList(ctx, courseID, filter)
	if err != nil {
		return nil, err
	}

	response := core.NewListResponse(students, total, filter.Page, filter.Limit)

	return &response, nil
}

// GetStudentProgress is the progress of one student, for the owner of the
// course and the SuperAdmin.
func (s *Progress) GetStudentProgress(
	ctx context.Context,
	actor models.Actor,
	courseID, studentID uuid.UUID,
) (*models.CourseProgress, error) {
	if _, err := core.ManageableCourse(ctx, s.repo, actor, courseID, "GetStudentProgress"); err != nil {
		return nil, err
	}

	if _, err := s.repo.Enrollment.GetByStudentAndCourse(ctx, studentID, courseID); err != nil {
		return nil, err
	}

	return s.courseProgress(ctx, studentID, courseID)
}
