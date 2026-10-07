package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestQuestion(t *testing.T, tc *TestContext, quizID uuid.UUID, orderNumber int) uuid.UUID {
	t.Helper()

	id, err := tc.Repo.Question.Create(context.Background(), models.CreateQuestion{
		QuizID:      quizID,
		Text:        "question_" + uuid.NewString(),
		Type:        models.QuestionTypeSingleChoice,
		OrderNumber: orderNumber,
	})
	require.NoError(t, err)

	return id
}

func createTestOption(t *testing.T, tc *TestContext, questionID uuid.UUID, text string, isCorrect bool) {
	t.Helper()

	err := tc.Repo.Question.CreateOption(context.Background(), models.CreateQuestionOption{
		QuestionID: questionID,
		OptionText: text,
		IsCorrect:  isCorrect,
	})
	require.NoError(t, err)
}

func TestQuestionRepo_Create(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)

	id := createTestQuestion(t, tc, quizID, 1)
	require.NotEqual(t, uuid.Nil, id)
}

func TestQuestionRepo_Create_DuplicateOrder(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	createTestQuestion(t, tc, quizID, 1)

	_, err := tc.Repo.Question.Create(context.Background(), models.CreateQuestion{
		QuizID:      quizID,
		Text:        "duplicate order",
		Type:        models.QuestionTypeTrueFalse,
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestQuestionRepo_Create_QuizNotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Question.Create(context.Background(), models.CreateQuestion{
		QuizID:      uuid.New(),
		Text:        "orphan",
		Type:        models.QuestionTypeTrueFalse,
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuestionRepo_CreateOption(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)

	createTestOption(t, tc, questionID, "Go", true)
	createTestOption(t, tc, questionID, "Java", false)

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)

	require.NoError(t, err)
	require.Len(t, options, 2)
}

func TestQuestionRepo_CreateOption_QuestionNotFound(t *testing.T) {
	tc := setupTest(t)

	err := tc.Repo.Question.CreateOption(context.Background(), models.CreateQuestionOption{
		QuestionID: uuid.New(),
		OptionText: "orphan",
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuestionRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	id := createTestQuestion(t, tc, quizID, 1)

	question, err := tc.Repo.Question.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, question.ID)
	require.Equal(t, quizID, question.QuizID)
	require.Equal(t, models.QuestionTypeSingleChoice, question.Type)
}

func TestQuestionRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Question.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuestionRepo_GetListByQuizID(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	second := createTestQuestion(t, tc, quizID, 2)
	first := createTestQuestion(t, tc, quizID, 1)

	questions, err := tc.Repo.Question.GetListByQuizID(context.Background(), quizID)

	require.NoError(t, err)
	require.Len(t, questions, 2)
	require.Equal(t, first, questions[0].ID)
	require.Equal(t, second, questions[1].ID)
}

func TestQuestionRepo_GetOptions(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)
	createTestOption(t, tc, questionID, "Go", true)

	options, err := tc.Repo.Question.GetOptions(context.Background(), questionID)

	require.NoError(t, err)
	require.Len(t, options, 1)
	require.Equal(t, questionID, options[0].QuestionID)
	require.Equal(t, "Go", options[0].OptionText)
	require.True(t, *options[0].IsCorrect)
}

func TestQuestionRepo_GetOptionsByQuizID(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	first := createTestQuestion(t, tc, quizID, 1)
	second := createTestQuestion(t, tc, quizID, 2)
	createTestOption(t, tc, first, "A", true)
	createTestOption(t, tc, second, "B", false)

	options, err := tc.Repo.Question.GetOptionsByQuizID(context.Background(), quizID)

	require.NoError(t, err)
	require.Len(t, options, 2)
	require.Equal(t, first, options[0].QuestionID)
	require.Equal(t, second, options[1].QuestionID)
}

func TestQuestionRepo_Update(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	id := createTestQuestion(t, tc, quizID, 1)

	text := "updated_" + uuid.NewString()
	questionType := models.QuestionTypeMultipleChoice

	question, err := tc.Repo.Question.Update(context.Background(), models.UpdateQuestion{
		ID:   id,
		Text: &text,
		Type: &questionType,
	})

	require.NoError(t, err)
	require.Equal(t, text, question.Text)
	require.Equal(t, models.QuestionTypeMultipleChoice, question.Type)
	require.Equal(t, 1, question.OrderNumber)
}

func TestQuestionRepo_Update_DuplicateOrder(t *testing.T) {
	tc := setupTest(t)

	quizID, _ := createTestQuiz(t, tc)
	createTestQuestion(t, tc, quizID, 1)
	id := createTestQuestion(t, tc, quizID, 2)

	orderNumber := 1

	_, err := tc.Repo.Question.Update(context.Background(), models.UpdateQuestion{
		ID:          id,
		OrderNumber: &orderNumber,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestQuestionRepo_Update_NotFound(t *testing.T) {
	tc := setupTest(t)

	text := "x"

	_, err := tc.Repo.Question.Update(context.Background(), models.UpdateQuestion{
		ID:   uuid.New(),
		Text: &text,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuestionRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	quizID, _ := createTestQuiz(t, tc)
	id := createTestQuestion(t, tc, quizID, 1)

	require.NoError(t, tc.Repo.Question.Delete(ctx, id))

	_, err := tc.Repo.Question.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestQuestionRepo_DeleteOptions(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)
	createTestOption(t, tc, questionID, "Go", true)

	require.NoError(t, tc.Repo.Question.DeleteOptions(ctx, questionID))

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)
	require.NoError(t, err)
	require.Empty(t, options)
}

func TestQuestionRepo_Options_AreOrderedByPosition(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	quizID, _ := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)

	// written in a different order than they are positioned
	for _, option := range []struct {
		text     string
		position int
	}{{"third", 3}, {"first", 1}, {"fourth", 4}, {"second", 2}} {
		err := tc.Repo.Question.CreateOption(ctx, models.CreateQuestionOption{
			QuestionID: questionID,
			Position:   option.position,
			OptionText: option.text,
		})
		require.NoError(t, err)
	}

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)
	require.NoError(t, err)

	texts := make([]string, len(options))
	for i, option := range options {
		texts[i] = option.OptionText
	}

	require.Equal(t, []string{"first", "second", "third", "fourth"}, texts)

	all, err := tc.Repo.Question.GetOptionsByQuizID(ctx, quizID)
	require.NoError(t, err)
	require.Len(t, all, 4)
	require.Equal(t, "first", all[0].OptionText)
	require.Equal(t, "fourth", all[3].OptionText)
}
