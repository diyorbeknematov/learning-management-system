package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestAttempt creates the student before the quiz on purpose: cleanups
// run in reverse order, so the quiz cleanup removes the attempt first and the
// student can be deleted afterwards.
func createTestAttempt(t *testing.T, tc *TestContext, studentID, quizID uuid.UUID, attemptNumber int) uuid.UUID {
	t.Helper()

	id, err := tc.Repo.Attempt.Create(context.Background(), models.CreateQuizAttempt{
		StudentID:     studentID,
		QuizID:        quizID,
		AttemptNumber: attemptNumber,
		StartedAt:     time.Now(),
	})
	require.NoError(t, err)

	return id
}

func createTestStudent(t *testing.T, tc *TestContext) uuid.UUID {
	t.Helper()

	id := createTestUser(t, tc)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	return id
}

func TestAttemptRepo_Create(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)

	id := createTestAttempt(t, tc, studentID, quizID, 1)
	require.NotEqual(t, uuid.Nil, id)
}

func TestAttemptRepo_Create_DuplicateNumber(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	createTestAttempt(t, tc, studentID, quizID, 1)

	_, err := tc.Repo.Attempt.Create(context.Background(), models.CreateQuizAttempt{
		StudentID:     studentID,
		QuizID:        quizID,
		AttemptNumber: 1,
		StartedAt:     time.Now(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestAttemptRepo_Create_QuizNotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)

	_, err := tc.Repo.Attempt.Create(context.Background(), models.CreateQuizAttempt{
		StudentID:     studentID,
		QuizID:        uuid.New(),
		AttemptNumber: 1,
		StartedAt:     time.Now(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAttemptRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	id := createTestAttempt(t, tc, studentID, quizID, 1)

	attempt, err := tc.Repo.Attempt.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, attempt.ID)
	require.Equal(t, studentID, attempt.StudentID)
	require.Equal(t, quizID, attempt.QuizID)
	require.Equal(t, 1, attempt.AttemptNumber)
	require.Nil(t, attempt.Score)
	require.Nil(t, attempt.CompletedAt)
}

func TestAttemptRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Attempt.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAttemptRepo_GetList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	student := createTestStudent(t, tc)
	other := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	createTestAttempt(t, tc, student, quizID, 1)
	createTestAttempt(t, tc, student, quizID, 2)
	createTestAttempt(t, tc, other, quizID, 1)

	all, err := tc.Repo.Attempt.GetList(ctx, quizID, nil)
	require.NoError(t, err)
	require.Len(t, all, 3)

	own, err := tc.Repo.Attempt.GetList(ctx, quizID, &student)
	require.NoError(t, err)
	require.Len(t, own, 2)
}

func TestAttemptRepo_CountByStudent(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)

	count, err := tc.Repo.Attempt.CountByStudent(ctx, quizID, studentID)
	require.NoError(t, err)
	require.Zero(t, count)

	createTestAttempt(t, tc, studentID, quizID, 1)
	createTestAttempt(t, tc, studentID, quizID, 2)

	count, err = tc.Repo.Attempt.CountByStudent(ctx, quizID, studentID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestAttemptRepo_Complete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	id := createTestAttempt(t, tc, studentID, quizID, 1)

	require.NoError(t, tc.Repo.Attempt.Complete(ctx, models.CompleteAttempt{
		ID:          id,
		Score:       80,
		CompletedAt: time.Now(),
		TimeSpent:   600,
	}))

	attempt, err := tc.Repo.Attempt.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 80, *attempt.Score)
	require.Equal(t, 600, *attempt.TimeSpent)
	require.NotNil(t, attempt.CompletedAt)
}

func TestAttemptRepo_Complete_Twice(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	id := createTestAttempt(t, tc, studentID, quizID, 1)

	result := models.CompleteAttempt{ID: id, Score: 80, CompletedAt: time.Now(), TimeSpent: 600}

	require.NoError(t, tc.Repo.Attempt.Complete(ctx, result))

	err := tc.Repo.Attempt.Complete(ctx, result)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAttemptRepo_CreateAnswer(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)
	createTestOption(t, tc, questionID, "Go", true)
	attemptID := createTestAttempt(t, tc, studentID, quizID, 1)

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)
	require.NoError(t, err)

	require.NoError(t, tc.Repo.Attempt.CreateAnswer(ctx, models.AttemptAnswer{
		AttemptID:  attemptID,
		QuestionID: questionID,
		OptionID:   options[0].ID,
	}))

	answers, err := tc.Repo.Attempt.GetAnswers(ctx, attemptID)
	require.NoError(t, err)
	require.Len(t, answers, 1)
	require.Equal(t, options[0].ID, answers[0].OptionID)
}

func TestAttemptRepo_CreateAnswer_Duplicate(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)
	createTestOption(t, tc, questionID, "Go", true)
	attemptID := createTestAttempt(t, tc, studentID, quizID, 1)

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)
	require.NoError(t, err)

	answer := models.AttemptAnswer{AttemptID: attemptID, QuestionID: questionID, OptionID: options[0].ID}

	require.NoError(t, tc.Repo.Attempt.CreateAnswer(ctx, answer))

	err = tc.Repo.Attempt.CreateAnswer(ctx, answer)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestAttemptRepo_CreateAnswer_AttemptNotFound(t *testing.T) {
	tc := setupTest(t)

	err := tc.Repo.Attempt.CreateAnswer(context.Background(), models.AttemptAnswer{
		AttemptID:  uuid.New(),
		QuestionID: uuid.New(),
		OptionID:   uuid.New(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestAttemptRepo_GetAnswers(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	quizID, _ := createTestQuiz(t, tc)
	attemptID := createTestAttempt(t, tc, studentID, quizID, 1)

	answers, err := tc.Repo.Attempt.GetAnswers(context.Background(), attemptID)

	require.NoError(t, err)
	require.Empty(t, answers)
}
