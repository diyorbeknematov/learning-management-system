package learning_test

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// exam is a course with a final quiz of three questions:
//
//	0: single choice   - Paris is right
//	1: multiple choice - Two and Three are right
//	2: true or false   - True is right
type exam struct {
	shop
	quiz      *models.Quiz
	questions []models.Question
}

func newExam(t *testing.T, passThreshold, maxAttempts int) exam {
	t.Helper()

	s := newShop(t, 0, 1, nil, 0)

	req := quizRequest()
	req.PassThreshold = passThreshold
	req.MaxAttempts = maxAttempts

	quiz := s.finalQuiz(t, req)

	s.addQuestion(t, quiz.ID, "Capital of France?", models.QuestionTypeSingleChoice, nil,
		opt("Paris", true), opt("Rome", false), opt("Berlin", false))
	s.addQuestion(t, quiz.ID, "Which are primes?", models.QuestionTypeMultipleChoice, nil,
		opt("Two", true), opt("Three", true), opt("Four", false))
	s.addQuestion(t, quiz.ID, "Go is compiled", models.QuestionTypeTrueFalse, nil,
		opt("True", true), opt("False", false))

	// the questions as the author sees them, with the right answers
	questions, err := s.e.Svc.Question.GetListByQuiz(context.Background(), testutil.ActorOf(s.owner), quiz.ID)
	require.NoError(t, err)

	return exam{shop: s, quiz: quiz, questions: questions}
}

// options returns the ids of the options of question i that are right (or
// wrong, when right is false).
func (x exam) options(i int, right bool) []uuid.UUID {
	var ids []uuid.UUID

	for _, option := range x.questions[i].Options {
		if *option.IsCorrect == right {
			ids = append(ids, option.ID)
		}
	}

	return ids
}

func (x exam) answer(i int, optionIDs ...uuid.UUID) models.SubmitAnswer {
	return models.SubmitAnswer{QuestionID: x.questions[i].ID, OptionIDs: optionIDs}
}

// rightAnswers answers all questions right.
func (x exam) rightAnswers() []models.SubmitAnswer {
	return []models.SubmitAnswer{
		x.answer(0, x.options(0, true)...),
		x.answer(1, x.options(1, true)...),
		x.answer(2, x.options(2, true)...),
	}
}

func (x exam) start(t *testing.T, student *models.User) *models.AttemptView {
	t.Helper()

	view, err := x.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(student), x.quiz.ID)
	require.NoError(t, err)

	return view
}

func (x exam) submit(t *testing.T, student *models.User, attemptID uuid.UUID, answers []models.SubmitAnswer) (*models.AttemptResult, error) {
	t.Helper()

	return x.e.Svc.Attempt.Submit(context.Background(), testutil.ActorOf(student), models.SubmitAttempt{AttemptID: attemptID, Answers: answers})
}

func TestAttempt_Start(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	require.Equal(t, student.ID, view.StudentID)
	require.Equal(t, x.quiz.ID, view.QuizID)
	require.Equal(t, 1, view.AttemptNumber)
	require.Nil(t, view.CompletedAt)
	require.Nil(t, view.Score)
	require.Len(t, view.Questions, 3)

	for _, question := range view.Questions {
		require.NotEmpty(t, question.Options)

		for _, option := range question.Options {
			require.Nil(t, option.IsCorrect, "the right answers must not reach the student")
		}
	}
}

func TestAttempt_Start_ContinuesTheRunningAttempt(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)

	first := x.start(t, student)
	again := x.start(t, student)

	require.Equal(t, first.ID, again.ID)
	require.Equal(t, 1, again.AttemptNumber)

	order := func(view *models.AttemptView) []uuid.UUID {
		ids := make([]uuid.UUID, len(view.Questions))
		for i, question := range view.Questions {
			ids[i] = question.ID
		}

		return ids
	}

	require.Equal(t, order(first), order(again), "the order stays the same during an attempt")
}

func TestAttempt_Start_QuestionOrderIsRandomBetweenAttempts(t *testing.T) {
	x := newExam(t, 70, 3)

	orders := map[string]bool{}

	for i := 0; i < 8; i++ {
		view := x.start(t, x.enrolledStudent(t))

		key := ""
		for _, question := range view.Questions {
			key += question.ID.String()
		}

		orders[key] = true
	}

	require.Greater(t, len(orders), 1, "eight attempts cannot all have the same order")
}

func TestAttempt_Start_TrueFalseKeepsItsOptionOrder(t *testing.T) {
	x := newExam(t, 70, 3)

	view := x.start(t, x.enrolledStudent(t))

	for _, question := range view.Questions {
		if question.Type == models.QuestionTypeTrueFalse {
			require.Equal(t, "True", question.Options[0].OptionText)
			require.Equal(t, "False", question.Options[1].OptionText)
		}
	}
}

func TestAttempt_Start_NotEnrolled(t *testing.T) {
	x := newExam(t, 70, 3)

	_, err := x.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(x.student(t)), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = x.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(x.owner), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAttempt_Start_DroppedStudent(t *testing.T) {
	x := newExam(t, 70, 3)

	ctx := context.Background()
	student := x.student(t)
	enrollment := x.enroll(t, student)

	err := x.e.Svc.Enrollment.UpdateStatus(ctx, testutil.ActorOf(student), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	_, err = x.e.Svc.Attempt.Start(ctx, testutil.ActorOf(student), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAttempt_Start_QuizWithoutQuestions(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	quiz := s.finalQuiz(t, quizRequest())

	_, err := s.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(s.enrolledStudent(t)), quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestAttempt_Start_UnknownQuiz(t *testing.T) {
	x := newExam(t, 70, 3)

	_, err := x.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(x.enrolledStudent(t)), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestAttempt_Start_NoAttemptsLeft(t *testing.T) {
	x := newExam(t, 70, 2)

	student := x.enrolledStudent(t)

	for i := 1; i <= 2; i++ {
		view := x.start(t, student)
		require.Equal(t, i, view.AttemptNumber)

		_, err := x.submit(t, student, view.ID, nil)
		require.NoError(t, err)
	}

	_, err := x.e.Svc.Attempt.Start(context.Background(), testutil.ActorOf(student), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAttempt_Submit_AllRight(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)

	require.Equal(t, 100, *result.Score)
	require.True(t, *result.Passed)
	require.NotNil(t, result.CompletedAt)
	require.NotNil(t, result.TimeSpent)
	require.Len(t, result.Results, 3)

	for _, outcome := range result.Results {
		require.True(t, outcome.Correct)
		require.NotEmpty(t, outcome.SelectedOptionIDs)
	}
}

func TestAttempt_Submit_AllWrong(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, []models.SubmitAnswer{
		x.answer(0, x.options(0, false)[0]),
		x.answer(1, x.options(1, false)[0]),
		x.answer(2, x.options(2, false)[0]),
	})
	require.NoError(t, err)
	require.Zero(t, *result.Score)
	require.False(t, *result.Passed)
}

func TestAttempt_Submit_PartialScoreAndThreshold(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	// two of three right: 67 points, below the threshold of 70
	result, err := x.submit(t, student, view.ID, []models.SubmitAnswer{
		x.answer(0, x.options(0, true)...),
		x.answer(1, x.options(1, true)...),
		x.answer(2, x.options(2, false)...),
	})
	require.NoError(t, err)
	require.Equal(t, 67, *result.Score)
	require.False(t, *result.Passed)
	require.Equal(t, []bool{true, true, false}, outcomes(result, x))

	lenient := newExam(t, 60, 3)
	other := lenient.enrolledStudent(t)
	otherView := lenient.start(t, other)

	result, err = lenient.submit(t, other, otherView.ID, []models.SubmitAnswer{
		lenient.answer(0, lenient.options(0, true)...),
		lenient.answer(1, lenient.options(1, true)...),
		lenient.answer(2, lenient.options(2, false)...),
	})
	require.NoError(t, err)
	require.Equal(t, 67, *result.Score)
	require.True(t, *result.Passed, "the same score passes a threshold of 60")
}

// outcomes returns whether each question of the exam was answered right, in
// the order of the exam.
func outcomes(result *models.AttemptResult, x exam) []bool {
	byQuestion := map[uuid.UUID]bool{}
	for _, outcome := range result.Results {
		byQuestion[outcome.QuestionID] = outcome.Correct
	}

	correct := make([]bool, len(x.questions))
	for i, question := range x.questions {
		correct[i] = byQuestion[question.ID]
	}

	return correct
}

func TestAttempt_Submit_MultipleChoiceNeedsTheExactSet(t *testing.T) {
	x := newExam(t, 0, 5)

	cases := map[string][]uuid.UUID{
		"only one of the right options":   x.options(1, true)[:1],
		"the right options and a wrong":   append(x.options(1, true), x.options(1, false)[0]),
		"only the wrong option":           x.options(1, false),
		"all the options of the question": append(x.options(1, true), x.options(1, false)...),
	}

	for name, chosen := range cases {
		student := x.enrolledStudent(t)
		view := x.start(t, student)

		result, err := x.submit(t, student, view.ID, []models.SubmitAnswer{x.answer(1, chosen...)})
		require.NoError(t, err, name)
		require.False(t, outcomes(result, x)[1], name)
	}
}

func TestAttempt_Submit_UnansweredQuestionsAreWrong(t *testing.T) {
	x := newExam(t, 0, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, []models.SubmitAnswer{x.answer(0, x.options(0, true)...)})
	require.NoError(t, err)
	require.Equal(t, 33, *result.Score)
	require.Equal(t, []bool{true, false, false}, outcomes(result, x))
	require.Empty(t, result.Results[1].SelectedOptionIDs)
}

func TestAttempt_Submit_NothingAnswered(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, nil)
	require.NoError(t, err)
	require.Zero(t, *result.Score)
	require.False(t, *result.Passed)
}

func TestAttempt_Submit_InvalidAnswers(t *testing.T) {
	x := newExam(t, 70, 5)

	otherQuestionOption := x.options(1, true)[0]

	cases := map[string][]models.SubmitAnswer{
		"a question that is not in the quiz": {{QuestionID: uuid.New(), OptionIDs: x.options(0, true)}},
		"an option of another question":      {x.answer(0, otherQuestionOption)},
		"an option that does not exist":      {x.answer(0, uuid.New())},
		"two answers for one question":       {x.answer(0, x.options(0, true)...), x.answer(0, x.options(0, false)[0])},
		"two options for a single choice":    {x.answer(0, x.options(0, true)[0], x.options(0, false)[0])},
		"no option for a single choice":      {x.answer(0)},
		"the same option twice":              {x.answer(1, x.options(1, true)[0], x.options(1, true)[0])},
	}

	for name, answers := range cases {
		student := x.enrolledStudent(t)
		view := x.start(t, student)

		_, err := x.submit(t, student, view.ID, answers)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)

		// a rejected submission does not close the attempt
		_, err = x.submit(t, student, view.ID, x.rightAnswers())
		require.NoError(t, err, name)
	}
}

func TestAttempt_Submit_Twice(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	_, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)

	_, err = x.submit(t, student, view.ID, x.rightAnswers())
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestAttempt_Submit_SomeoneElsesAttempt(t *testing.T) {
	x := newExam(t, 70, 3)

	view := x.start(t, x.enrolledStudent(t))

	_, err := x.submit(t, x.enrolledStudent(t), view.ID, x.rightAnswers())
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAttempt_Submit_UnknownAttempt(t *testing.T) {
	x := newExam(t, 70, 3)

	_, err := x.submit(t, x.enrolledStudent(t), uuid.New(), nil)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func (x exam) rewind(t *testing.T, attemptID uuid.UUID, by time.Duration) {
	t.Helper()

	x.e.Exec(t, `UPDATE quiz_attempts SET started_at = $2 WHERE id = $1`, attemptID, time.Now().Add(-by))
}

func TestAttempt_Submit_TimeIsOver(t *testing.T) {
	x := newExam(t, 70, 3)

	ctx := context.Background()
	student := x.enrolledStudent(t)
	view := x.start(t, student)

	x.rewind(t, view.ID, 2*time.Hour)

	_, err := x.submit(t, student, view.ID, x.rightAnswers())
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	detail, err := x.e.Svc.Attempt.GetByID(ctx, testutil.ActorOf(student), view.ID)
	require.NoError(t, err)
	require.NotNil(t, detail.CompletedAt, "a late attempt is closed")
	require.Zero(t, *detail.Score)
	require.False(t, *detail.Passed)
	require.Empty(t, detail.Answers, "the late answers are not saved")
}

func TestAttempt_Submit_JustInsideTheGracePeriod(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	view := x.start(t, student)

	// the limit is 30 minutes; the submission comes 10 seconds late
	x.rewind(t, view.ID, 30*time.Minute+10*time.Second)

	result, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)
	require.Equal(t, 100, *result.Score)
}

func TestAttempt_Start_AfterTheTimeIsOverOpensANewAttempt(t *testing.T) {
	x := newExam(t, 70, 2)

	ctx := context.Background()
	student := x.enrolledStudent(t)
	abandoned := x.start(t, student)

	x.rewind(t, abandoned.ID, 2*time.Hour)

	next := x.start(t, student)
	require.NotEqual(t, abandoned.ID, next.ID)
	require.Equal(t, 2, next.AttemptNumber)

	old, err := x.e.Svc.Attempt.GetByID(ctx, testutil.ActorOf(student), abandoned.ID)
	require.NoError(t, err)
	require.NotNil(t, old.CompletedAt)
	require.Zero(t, *old.Score)

	// the abandoned attempt counted: the second was the last one
	_, err = x.submit(t, student, next.ID, nil)
	require.NoError(t, err)

	_, err = x.e.Svc.Attempt.Start(ctx, testutil.ActorOf(student), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAttempt_GetList(t *testing.T) {
	x := newExam(t, 70, 3)

	ctx := context.Background()
	first, second := x.enrolledStudent(t), x.enrolledStudent(t)

	firstView := x.start(t, first)
	_, err := x.submit(t, first, firstView.ID, x.rightAnswers())
	require.NoError(t, err)

	secondView := x.start(t, second)
	_, err = x.submit(t, second, secondView.ID, nil)
	require.NoError(t, err)

	own, err := x.e.Svc.Attempt.GetList(ctx, testutil.ActorOf(first), x.quiz.ID)
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, firstView.ID, own[0].ID)
	require.True(t, *own[0].Passed)

	for _, actor := range []models.Actor{testutil.ActorOf(x.owner), testutil.AdminActor()} {
		all, err := x.e.Svc.Attempt.GetList(ctx, actor, x.quiz.ID)
		require.NoError(t, err)
		require.Len(t, all, 2)
	}

	_, err = x.e.Svc.Attempt.GetList(ctx, testutil.ActorOf(x.student(t)), x.quiz.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = x.e.Svc.Attempt.GetList(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestAttempt_GetByID(t *testing.T) {
	x := newExam(t, 70, 3)

	ctx := context.Background()
	student := x.enrolledStudent(t)
	view := x.start(t, student)

	_, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(x.owner), testutil.AdminActor()} {
		detail, err := x.e.Svc.Attempt.GetByID(ctx, actor, view.ID)
		require.NoError(t, err)
		require.Equal(t, 100, *detail.Score)
		require.True(t, *detail.Passed)
		require.Len(t, detail.Answers, 4, "one saved answer per chosen option")
	}

	_, err = x.e.Svc.Attempt.GetByID(ctx, testutil.ActorOf(x.enrolledStudent(t)), view.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = x.e.Svc.Attempt.GetByID(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestAttempt_EditingTheQuizKeepsOldResults(t *testing.T) {
	x := newExam(t, 70, 3)

	ctx := context.Background()
	student := x.enrolledStudent(t)
	view := x.start(t, student)

	_, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)

	// the author rewrites the options of the first question afterwards
	options := []models.CreateQuestionOption{opt("Lyon", false), opt("Paris", true)}

	_, err = x.e.Svc.Question.Update(ctx, testutil.ActorOf(x.owner), x.questions[0].ID, models.UpdateQuestion{Options: &options})
	require.NoError(t, err)

	detail, err := x.e.Svc.Attempt.GetByID(ctx, testutil.ActorOf(student), view.ID)
	require.NoError(t, err)
	require.Equal(t, 100, *detail.Score)
	require.Len(t, detail.Answers, 4)
}
