package learning_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func quizRequest() models.CreateQuiz {
	return models.CreateQuiz{
		Title:         "Quiz " + testutil.Uniq(),
		TimeLimit:     30,
		PassThreshold: 70,
		MaxAttempts:   3,
	}
}

func (s shop) finalQuiz(t *testing.T, req models.CreateQuiz) *models.Quiz {
	t.Helper()

	quiz, err := s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.ActorOf(s.owner), s.course.ID, req)
	require.NoError(t, err)

	return quiz
}

func (s shop) enrolledStudent(t *testing.T) *models.User {
	t.Helper()

	student := s.student(t)
	s.enroll(t, student)

	return student
}

func TestQuiz_CreateForCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	description := "Final exam"
	req := quizRequest()
	req.Description = &description

	quiz := s.finalQuiz(t, req)

	require.Equal(t, s.course.ID, *quiz.CourseID)
	require.Nil(t, quiz.ModuleID)
	require.Equal(t, 30, quiz.TimeLimit)
	require.Equal(t, 70, quiz.PassThreshold)
	require.Equal(t, 3, quiz.MaxAttempts)
	require.Equal(t, description, *quiz.Description)
}

func TestQuiz_CreateForCourse_OnlyOne(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	s.finalQuiz(t, quizRequest())

	_, err := s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.ActorOf(s.owner), s.course.ID, quizRequest())
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestQuiz_CreateForCourse_NotTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), s.course.ID, quizRequest())
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.ActorOf(s.student(t)), s.course.ID, quizRequest())
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuiz_CreateForCourse_InvalidValues(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	for name, mutate := range map[string]func(*models.CreateQuiz){
		"threshold over 100": func(q *models.CreateQuiz) { q.PassThreshold = 150 },
		"no time":            func(q *models.CreateQuiz) { q.TimeLimit = 0 },
		"no attempts":        func(q *models.CreateQuiz) { q.MaxAttempts = 0 },
	} {
		req := quizRequest()
		mutate(&req)

		_, err := s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.ActorOf(s.owner), s.course.ID, req)
		require.Error(t, err, name)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestQuiz_CreateForCourse_UnknownCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Quiz.CreateForCourse(context.Background(), testutil.AdminActor(), uuid.New(), quizRequest())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestQuiz_CreateForModule(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()

	first, err := s.e.Svc.Quiz.CreateForModule(ctx, testutil.ActorOf(s.owner), s.module.ID, quizRequest())
	require.NoError(t, err)
	require.Equal(t, s.module.ID, *first.ModuleID)
	require.Nil(t, first.CourseID)

	// a module can have several quizzes
	_, err = s.e.Svc.Quiz.CreateForModule(ctx, testutil.ActorOf(s.owner), s.module.ID, quizRequest())
	require.NoError(t, err)

	_, err = s.e.Svc.Quiz.CreateForModule(ctx, testutil.AdminActor(), uuid.New(), quizRequest())
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = s.e.Svc.Quiz.CreateForModule(ctx, testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), s.module.ID, quizRequest())
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuiz_GetByID(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	student := s.enrolledStudent(t)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(s.owner), testutil.AdminActor()} {
		found, err := s.e.Svc.Quiz.GetByID(ctx, actor, quiz.ID)
		require.NoError(t, err)
		require.Equal(t, quiz.ID, found.ID)
	}
}

func TestQuiz_GetByID_NotEnrolled(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())

	_, err := s.e.Svc.Quiz.GetByID(context.Background(), testutil.ActorOf(s.student(t)), quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Quiz.GetByID(context.Background(), models.Actor{}, quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuiz_GetByID_DraftCourseIsHiddenFromStudents(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	student := s.enrolledStudent(t)

	err := s.e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(s.owner), models.UpdateCourseStatus{ID: s.course.ID, Status: models.CourseStatusDraft})
	require.NoError(t, err)

	_, err = s.e.Svc.Quiz.GetByID(ctx, testutil.ActorOf(student), quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = s.e.Svc.Quiz.GetByID(ctx, testutil.ActorOf(s.owner), quiz.ID)
	require.NoError(t, err)
}

func TestQuiz_GetByID_NotFound(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Quiz.GetByID(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestQuiz_GetList(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	final := s.finalQuiz(t, quizRequest())
	student := s.enrolledStudent(t)

	moduleQuiz, err := s.e.Svc.Quiz.CreateForModule(ctx, testutil.ActorOf(s.owner), s.module.ID, quizRequest())
	require.NoError(t, err)

	byCourse, err := s.e.Svc.Quiz.GetListByCourse(ctx, testutil.ActorOf(student), s.course.ID)
	require.NoError(t, err)
	require.Len(t, byCourse, 1)
	require.Equal(t, final.ID, byCourse[0].ID)

	byModule, err := s.e.Svc.Quiz.GetListByModule(ctx, testutil.ActorOf(student), s.module.ID)
	require.NoError(t, err)
	require.Len(t, byModule, 1)
	require.Equal(t, moduleQuiz.ID, byModule[0].ID)

	_, err = s.e.Svc.Quiz.GetListByCourse(ctx, testutil.ActorOf(s.student(t)), s.course.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Quiz.GetListByModule(ctx, testutil.ActorOf(s.student(t)), s.module.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuiz_Update(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	title := "Renamed"
	attempts := 5

	updated, err := s.e.Svc.Quiz.Update(context.Background(), testutil.ActorOf(s.owner), quiz.ID, models.UpdateQuiz{Title: &title, MaxAttempts: &attempts})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.Title)
	require.Equal(t, 5, updated.MaxAttempts)
	require.Equal(t, 70, updated.PassThreshold)
}

func TestQuiz_Update_NotTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	title := "Hijacked"

	_, err := s.e.Svc.Quiz.Update(context.Background(), testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), quiz.ID, models.UpdateQuiz{Title: &title})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuiz_Delete(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())

	err := s.e.Svc.Quiz.Delete(ctx, testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	require.NoError(t, s.e.Svc.Quiz.Delete(ctx, testutil.ActorOf(s.owner), quiz.ID))

	_, err = s.e.Svc.Quiz.GetByID(ctx, testutil.AdminActor(), quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	// the course can get a new final quiz
	s.finalQuiz(t, quizRequest())
}
