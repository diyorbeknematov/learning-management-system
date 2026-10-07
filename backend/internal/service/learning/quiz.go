package learning

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type Quiz struct {
	repo *repo.Repository
}

func NewQuiz(deps core.Dependencies) *Quiz {
	return &Quiz{
		repo: deps.Repo,
	}
}

// quizCourse returns the course a quiz belongs to: directly for a final quiz
// of the course, through the module for a module quiz.
func quizCourse(ctx context.Context, r *repo.Repository, quiz *models.Quiz) (*models.Course, error) {
	courseID := uuid.Nil

	switch {
	case quiz.CourseID != nil:
		courseID = *quiz.CourseID
	case quiz.ModuleID != nil:
		module, err := r.Module.GetByID(ctx, *quiz.ModuleID)
		if err != nil {
			return nil, err
		}

		courseID = module.CourseID
	}

	return r.Course.GetByID(ctx, courseID)
}

// canView allows the owner of the course, the SuperAdmin and a student who
// studies the course.
func canView(ctx context.Context, r *repo.Repository, actor models.Actor, course *models.Course, op string) error {
	if core.CanManage(actor, course.InstructorID, op) == nil {
		return nil
	}

	if course.Status != models.CourseStatusPublished {
		return apperror.NotFound("service", op, "not found", apperror.ErrNotFound)
	}

	_, err := core.ActiveEnrollment(ctx, r, actor.UserID, course.ID, op)

	return err
}

// CreateForCourse adds the final quiz of a course; a course has only one.
func (s *Quiz) CreateForCourse(ctx context.Context, actor models.Actor, courseID uuid.UUID, req models.CreateQuiz) (*models.Quiz, error) {
	if _, err := core.ManageableCourse(ctx, s.repo, actor, courseID, "CreateQuiz"); err != nil {
		return nil, err
	}

	req.CourseID, req.ModuleID = &courseID, nil

	return s.create(ctx, req)
}

// CreateForModule adds a quiz to a module.
func (s *Quiz) CreateForModule(ctx context.Context, actor models.Actor, moduleID uuid.UUID, req models.CreateQuiz) (*models.Quiz, error) {
	module, err := s.repo.Module.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}

	if _, err := core.ManageableCourse(ctx, s.repo, actor, module.CourseID, "CreateQuiz"); err != nil {
		return nil, err
	}

	req.CourseID, req.ModuleID = nil, &moduleID

	return s.create(ctx, req)
}

func (s *Quiz) create(ctx context.Context, req models.CreateQuiz) (*models.Quiz, error) {
	id, err := s.repo.Quiz.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.repo.Quiz.GetByID(ctx, id)
}

func (s *Quiz) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.Quiz, error) {
	quiz, err := s.repo.Quiz.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return nil, err
	}

	if err := canView(ctx, s.repo, actor, course, "GetQuiz"); err != nil {
		return nil, err
	}

	return quiz, nil
}

func (s *Quiz) GetListByCourse(ctx context.Context, actor models.Actor, courseID uuid.UUID) ([]models.Quiz, error) {
	course, err := s.repo.Course.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	if err := canView(ctx, s.repo, actor, course, "GetQuizList"); err != nil {
		return nil, err
	}

	return s.repo.Quiz.GetListByCourseID(ctx, courseID)
}

func (s *Quiz) GetListByModule(ctx context.Context, actor models.Actor, moduleID uuid.UUID) ([]models.Quiz, error) {
	module, err := s.repo.Module.GetByID(ctx, moduleID)
	if err != nil {
		return nil, err
	}

	course, err := s.repo.Course.GetByID(ctx, module.CourseID)
	if err != nil {
		return nil, err
	}

	if err := canView(ctx, s.repo, actor, course, "GetQuizList"); err != nil {
		return nil, err
	}

	return s.repo.Quiz.GetListByModuleID(ctx, moduleID)
}

func (s *Quiz) Update(ctx context.Context, actor models.Actor, id uuid.UUID, req models.UpdateQuiz) (*models.Quiz, error) {
	quiz, err := s.repo.Quiz.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "UpdateQuiz"); err != nil {
		return nil, err
	}

	req.ID = id

	return s.repo.Quiz.Update(ctx, req)
}

// Delete removes the quiz; the attempts already made stay in the history.
func (s *Quiz) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	quiz, err := s.repo.Quiz.GetByID(ctx, id)
	if err != nil {
		return err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return err
	}

	if err := core.CanManage(actor, course.InstructorID, "DeleteQuiz"); err != nil {
		return err
	}

	return s.repo.Quiz.Delete(ctx, id)
}
