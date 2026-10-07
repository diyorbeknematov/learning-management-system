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

func opt(text string, correct bool) models.CreateQuestionOption {
	return models.CreateQuestionOption{OptionText: text, IsCorrect: correct}
}

func (s shop) addQuestion(t *testing.T, quizID uuid.UUID, text string, questionType models.QuestionType, order *int, options ...models.CreateQuestionOption) *models.Question {
	t.Helper()

	question, err := s.e.Svc.Question.Create(context.Background(), testutil.ActorOf(s.owner), quizID, models.CreateQuestionRequest{
		Text:        text,
		Type:        questionType,
		OrderNumber: order,
		Options:     options,
	})
	require.NoError(t, err)

	return question
}

func (s shop) questionTexts(t *testing.T, quizID uuid.UUID) []string {
	t.Helper()

	questions, err := s.e.Svc.Question.GetListByQuiz(context.Background(), testutil.ActorOf(s.owner), quizID)
	require.NoError(t, err)

	texts := make([]string, len(questions))

	for i, question := range questions {
		require.Equal(t, i+1, question.OrderNumber, "the numbers must be 1..n without gaps")
		texts[i] = question.Text
	}

	return texts
}

func intPtr(n int) *int {
	return &n
}

func TestQuestion_Create_SingleChoice(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Capital of France?", models.QuestionTypeSingleChoice, nil,
		opt("Paris", true), opt("Rome", false), opt("Berlin", false))

	require.Equal(t, quiz.ID, question.QuizID)
	require.Equal(t, 1, question.OrderNumber)
	require.Len(t, question.Options, 3)

	right := 0
	for _, option := range question.Options {
		if *option.IsCorrect {
			right++
			require.Equal(t, "Paris", option.OptionText)
		}
	}

	require.Equal(t, 1, right)
}

func TestQuestion_Create_RulesByType(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())

	valid := []struct {
		name         string
		questionType models.QuestionType
		options      []models.CreateQuestionOption
	}{
		{"single choice", models.QuestionTypeSingleChoice, []models.CreateQuestionOption{opt("a", true), opt("b", false)}},
		{"multiple choice with one right", models.QuestionTypeMultipleChoice, []models.CreateQuestionOption{opt("a", true), opt("b", false)}},
		{"multiple choice with two right", models.QuestionTypeMultipleChoice, []models.CreateQuestionOption{opt("a", true), opt("b", true), opt("c", false)}},
		{"true or false", models.QuestionTypeTrueFalse, []models.CreateQuestionOption{opt("True", true), opt("False", false)}},
	}

	for _, tt := range valid {
		_, err := s.e.Svc.Question.Create(context.Background(), testutil.ActorOf(s.owner), quiz.ID, models.CreateQuestionRequest{
			Text: tt.name, Type: tt.questionType, Options: tt.options,
		})
		require.NoError(t, err, tt.name)
	}

	invalid := []struct {
		name         string
		questionType models.QuestionType
		options      []models.CreateQuestionOption
	}{
		{"one option", models.QuestionTypeSingleChoice, []models.CreateQuestionOption{opt("a", true)}},
		{"single choice without a right option", models.QuestionTypeSingleChoice, []models.CreateQuestionOption{opt("a", false), opt("b", false)}},
		{"single choice with two right options", models.QuestionTypeSingleChoice, []models.CreateQuestionOption{opt("a", true), opt("b", true)}},
		{"multiple choice without a right option", models.QuestionTypeMultipleChoice, []models.CreateQuestionOption{opt("a", false), opt("b", false)}},
		{"true or false with three options", models.QuestionTypeTrueFalse, []models.CreateQuestionOption{opt("a", true), opt("b", false), opt("c", false)}},
		{"true or false with both right", models.QuestionTypeTrueFalse, []models.CreateQuestionOption{opt("True", true), opt("False", true)}},
		{"unknown type", models.QuestionType("essay"), []models.CreateQuestionOption{opt("a", true), opt("b", false)}},
	}

	for _, tt := range invalid {
		_, err := s.e.Svc.Question.Create(context.Background(), testutil.ActorOf(s.owner), quiz.ID, models.CreateQuestionRequest{
			Text: tt.name, Type: tt.questionType, Options: tt.options,
		})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestQuestion_Create_Order(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	options := []models.CreateQuestionOption{opt("a", true), opt("b", false)}

	s.addQuestion(t, quiz.ID, "A", models.QuestionTypeSingleChoice, nil, options...)
	s.addQuestion(t, quiz.ID, "B", models.QuestionTypeSingleChoice, nil, options...)
	inserted := s.addQuestion(t, quiz.ID, "X", models.QuestionTypeSingleChoice, intPtr(2), options...)
	last := s.addQuestion(t, quiz.ID, "Z", models.QuestionTypeSingleChoice, intPtr(99), options...)

	require.Equal(t, 2, inserted.OrderNumber)
	require.Equal(t, 4, last.OrderNumber)
	require.Equal(t, []string{"A", "X", "B", "Z"}, s.questionTexts(t, quiz.ID))
}

func TestQuestion_Create_NotTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())

	_, err := s.e.Svc.Question.Create(context.Background(), testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), quiz.ID, models.CreateQuestionRequest{
		Text: "x", Type: models.QuestionTypeSingleChoice, Options: []models.CreateQuestionOption{opt("a", true), opt("b", false)},
	})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuestion_Create_UnknownQuiz(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Question.Create(context.Background(), testutil.AdminActor(), uuid.New(), models.CreateQuestionRequest{
		Text: "x", Type: models.QuestionTypeSingleChoice, Options: []models.CreateQuestionOption{opt("a", true), opt("b", false)},
	})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestQuestion_StudentsCannotReadQuestionsDirectly(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))
	student := testutil.ActorOf(s.enrolledStudent(t))

	_, err := s.e.Svc.Question.GetByID(ctx, student, question.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = s.e.Svc.Question.GetListByQuiz(ctx, student, quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuestion_GetByID_And_List_ShowTheRightAnswers(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))

	for _, actor := range []models.Actor{testutil.ActorOf(s.owner), testutil.AdminActor()} {
		found, err := s.e.Svc.Question.GetByID(ctx, actor, question.ID)
		require.NoError(t, err)
		require.Len(t, found.Options, 2)
		require.NotNil(t, found.Options[0].IsCorrect)

		list, err := s.e.Svc.Question.GetListByQuiz(ctx, actor, quiz.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Len(t, list[0].Options, 2)
	}

	_, err := s.e.Svc.Question.GetByID(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestQuestion_Update_Text(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Old", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))
	text := "New"

	updated, err := s.e.Svc.Question.Update(context.Background(), testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Text: &text})
	require.NoError(t, err)
	require.Equal(t, "New", updated.Text)
	require.Len(t, updated.Options, 2, "the options stay when none are sent")
}

func TestQuestion_Update_ReplacesOptions(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))

	options := []models.CreateQuestionOption{opt("x", false), opt("y", true), opt("z", false)}

	updated, err := s.e.Svc.Question.Update(context.Background(), testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Options: &options})
	require.NoError(t, err)
	require.Len(t, updated.Options, 3)

	texts := []string{}
	for _, option := range updated.Options {
		texts = append(texts, option.OptionText)
	}

	require.ElementsMatch(t, []string{"x", "y", "z"}, texts)
}

func TestQuestion_Update_NewOptionsMustFitTheType(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))

	options := []models.CreateQuestionOption{opt("x", true), opt("y", true)}

	_, err := s.e.Svc.Question.Update(context.Background(), testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Options: &options})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	found, err := s.e.Svc.Question.GetByID(context.Background(), testutil.ActorOf(s.owner), question.ID)
	require.NoError(t, err)
	require.Len(t, found.Options, 2, "a failed update changes nothing")
	require.Equal(t, "a", found.Options[0].OptionText)
}

func TestQuestion_Update_TypeChangeChecksTheOldOptions(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false), opt("c", false))

	toTrueFalse := models.QuestionTypeTrueFalse

	_, err := s.e.Svc.Question.Update(ctx, testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Type: &toTrueFalse})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	toMultiple := models.QuestionTypeMultipleChoice

	updated, err := s.e.Svc.Question.Update(ctx, testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Type: &toMultiple})
	require.NoError(t, err)
	require.Equal(t, models.QuestionTypeMultipleChoice, updated.Type)
}

func TestQuestion_Update_Order(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	options := []models.CreateQuestionOption{opt("a", true), opt("b", false)}

	s.addQuestion(t, quiz.ID, "A", models.QuestionTypeSingleChoice, nil, options...)
	s.addQuestion(t, quiz.ID, "B", models.QuestionTypeSingleChoice, nil, options...)
	third := s.addQuestion(t, quiz.ID, "C", models.QuestionTypeSingleChoice, nil, options...)

	_, err := s.e.Svc.Question.Update(context.Background(), testutil.ActorOf(s.owner), third.ID, models.UpdateQuestion{OrderNumber: intPtr(1)})
	require.NoError(t, err)
	require.Equal(t, []string{"C", "A", "B"}, s.questionTexts(t, quiz.ID))
}

func TestQuestion_Update_NotTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())
	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeSingleChoice, nil, opt("a", true), opt("b", false))
	text := "Hijacked"

	_, err := s.e.Svc.Question.Update(context.Background(), testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), question.ID, models.UpdateQuestion{Text: &text})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestQuestion_Delete_ClosesTheGap(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())
	options := []models.CreateQuestionOption{opt("a", true), opt("b", false)}

	s.addQuestion(t, quiz.ID, "A", models.QuestionTypeSingleChoice, nil, options...)
	middle := s.addQuestion(t, quiz.ID, "B", models.QuestionTypeSingleChoice, nil, options...)
	s.addQuestion(t, quiz.ID, "C", models.QuestionTypeSingleChoice, nil, options...)

	err := s.e.Svc.Question.Delete(ctx, testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), middle.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	require.NoError(t, s.e.Svc.Question.Delete(ctx, testutil.ActorOf(s.owner), middle.ID))
	require.Equal(t, []string{"A", "C"}, s.questionTexts(t, quiz.ID))

	_, err = s.e.Svc.Question.GetByID(ctx, testutil.ActorOf(s.owner), middle.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func optionTexts(question *models.Question) []string {
	texts := make([]string, len(question.Options))

	for i, option := range question.Options {
		texts[i] = option.OptionText
	}

	return texts
}

func TestQuestion_OptionsKeepTheOrderOfTheAuthor(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	quiz := s.finalQuiz(t, quizRequest())

	question := s.addQuestion(t, quiz.ID, "Q", models.QuestionTypeMultipleChoice, nil,
		opt("Delta", false), opt("Alpha", true), opt("Charlie", false), opt("Bravo", true), opt("Echo", false))

	require.Equal(t, []string{"Delta", "Alpha", "Charlie", "Bravo", "Echo"}, optionTexts(question))

	found, err := s.e.Svc.Question.GetByID(ctx, testutil.ActorOf(s.owner), question.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"Delta", "Alpha", "Charlie", "Bravo", "Echo"}, optionTexts(found))

	list, err := s.e.Svc.Question.GetListByQuiz(ctx, testutil.ActorOf(s.owner), quiz.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"Delta", "Alpha", "Charlie", "Bravo", "Echo"}, optionTexts(&list[0]))

	replaced := []models.CreateQuestionOption{opt("Zulu", false), opt("Yankee", true), opt("X-ray", false)}

	updated, err := s.e.Svc.Question.Update(ctx, testutil.ActorOf(s.owner), question.ID, models.UpdateQuestion{Options: &replaced})
	require.NoError(t, err)
	require.Equal(t, []string{"Zulu", "Yankee", "X-ray"}, optionTexts(updated), "the new options replace the old ones, in the new order")
}
