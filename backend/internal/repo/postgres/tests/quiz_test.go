package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestQuiz(courseID, moduleID *uuid.UUID) models.CreateQuiz {
	return models.CreateQuiz{
		CourseID:      courseID,
		ModuleID:      moduleID,
		Title:         "quiz_" + uuid.NewString(),
		TimeLimit:     30,
		PassThreshold: 70,
		MaxAttempts:   3,
	}
}

func cleanupTestQuiz(t *testing.T, tc *TestContext, id uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `
			DELETE FROM attempt_answers
			WHERE attempt_id IN (SELECT id FROM quiz_attempts WHERE quiz_id = $1)`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM quiz_attempts WHERE quiz_id = $1`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `
			DELETE FROM question_options
			WHERE question_id IN (SELECT id FROM questions WHERE quiz_id = $1)`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM questions WHERE quiz_id = $1`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM quizzes WHERE id = $1`, id)
	})
}

func createTestQuiz(t *testing.T, tc *TestContext) (uuid.UUID, uuid.UUID) {
	t.Helper()

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	id, err := tc.Repo.Quiz.Create(context.Background(), newTestQuiz(&courseID, nil))
	require.NoError(t, err)

	cleanupTestQuiz(t, tc, id)

	return id, courseID
}

func TestQuizRepo_Create(t *testing.T) {
	tc := setupTest(t)

	id, _ := createTestQuiz(t, tc)
	require.NotEqual(t, uuid.Nil, id)
}

func TestQuizRepo_Create_ModuleLevel(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)

	id, err := tc.Repo.Quiz.Create(context.Background(), newTestQuiz(nil, &moduleID))
	require.NoError(t, err)

	cleanupTestQuiz(t, tc, id)

	quizzes, err := tc.Repo.Quiz.GetListByModuleID(context.Background(), moduleID)
	require.NoError(t, err)
	require.Len(t, quizzes, 1)
	require.Equal(t, id, quizzes[0].ID)
}

func TestQuizRepo_Create_SecondCourseQuiz(t *testing.T) {
	tc := setupTest(t)

	_, courseID := createTestQuiz(t, tc)

	_, err := tc.Repo.Quiz.Create(context.Background(), newTestQuiz(&courseID, nil))

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestQuizRepo_Create_NoOwner(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Quiz.Create(context.Background(), newTestQuiz(nil, nil))

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestQuizRepo_Create_CourseNotFound(t *testing.T) {
	tc := setupTest(t)

	courseID := uuid.New()

	_, err := tc.Repo.Quiz.Create(context.Background(), newTestQuiz(&courseID, nil))

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuizRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	id, courseID := createTestQuiz(t, tc)

	quiz, err := tc.Repo.Quiz.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, quiz.ID)
	require.Equal(t, courseID, *quiz.CourseID)
	require.Nil(t, quiz.ModuleID)
	require.Equal(t, 30, quiz.TimeLimit)
	require.Equal(t, 70, quiz.PassThreshold)
	require.Equal(t, 3, quiz.MaxAttempts)
}

func TestQuizRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Quiz.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuizRepo_GetListByCourseID(t *testing.T) {
	tc := setupTest(t)

	id, courseID := createTestQuiz(t, tc)

	quizzes, err := tc.Repo.Quiz.GetListByCourseID(context.Background(), courseID)

	require.NoError(t, err)
	require.Len(t, quizzes, 1)
	require.Equal(t, id, quizzes[0].ID)
}

func TestQuizRepo_GetListByModuleID(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)

	quizzes, err := tc.Repo.Quiz.GetListByModuleID(context.Background(), moduleID)

	require.NoError(t, err)
	require.Empty(t, quizzes)
}

func TestQuizRepo_Update(t *testing.T) {
	tc := setupTest(t)

	id, _ := createTestQuiz(t, tc)

	title := "renamed_" + uuid.NewString()
	maxAttempts := 5

	quiz, err := tc.Repo.Quiz.Update(context.Background(), models.UpdateQuiz{
		ID:          id,
		Title:       &title,
		MaxAttempts: &maxAttempts,
	})

	require.NoError(t, err)
	require.Equal(t, title, quiz.Title)
	require.Equal(t, 5, quiz.MaxAttempts)
	require.Equal(t, 70, quiz.PassThreshold)
}

func TestQuizRepo_Update_InvalidThreshold(t *testing.T) {
	tc := setupTest(t)

	id, _ := createTestQuiz(t, tc)

	threshold := 150

	_, err := tc.Repo.Quiz.Update(context.Background(), models.UpdateQuiz{
		ID:            id,
		PassThreshold: &threshold,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestQuizRepo_Update_NotFound(t *testing.T) {
	tc := setupTest(t)

	title := "x"

	_, err := tc.Repo.Quiz.Update(context.Background(), models.UpdateQuiz{
		ID:    uuid.New(),
		Title: &title,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuizRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id, courseID := createTestQuiz(t, tc)

	require.NoError(t, tc.Repo.Quiz.Delete(ctx, id))

	_, err := tc.Repo.Quiz.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)

	// the course can get a new final quiz once the old one is soft-deleted
	newID, err := tc.Repo.Quiz.Create(ctx, newTestQuiz(&courseID, nil))
	require.NoError(t, err)

	cleanupTestQuiz(t, tc, newID)
}
