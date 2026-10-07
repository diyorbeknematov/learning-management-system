package api_test

import (
	"net/http"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func quizBody() map[string]any {
	return map[string]any{"title": "Final exam", "time_limit": 30, "pass_threshold": 70, "max_attempts": 3}
}

func (s school) createQuiz() string {
	s.t.Helper()

	got := s.as(s.owner, http.MethodPost, s.path("/quizzes"), quizBody())
	require.Equal(s.t, http.StatusCreated, got.Status, got.Raw)

	return got.data()["id"].(string)
}

func option(text string, correct bool) map[string]any {
	return map[string]any{"option_text": text, "is_correct": correct}
}

// addQuestions gives the quiz two questions (single choice, then multiple
// choice) and returns the questions as the author sees them.
func (s school) addQuestions(quizID string) []any {
	s.t.Helper()

	path := "/api/v1/quizzes/" + quizID + "/questions"

	one := s.as(s.owner, http.MethodPost, path, map[string]any{
		"text": "Capital of France?", "type": "single_choice",
		"options": []any{option("Paris", true), option("Rome", false), option("Berlin", false)},
	})
	require.Equal(s.t, http.StatusCreated, one.Status, one.Raw)

	two := s.as(s.owner, http.MethodPost, path, map[string]any{
		"text": "Which are primes?", "type": "multiple_choice",
		"options": []any{option("Two", true), option("Three", true), option("Four", false)},
	})
	require.Equal(s.t, http.StatusCreated, two.Status, two.Raw)

	list := s.as(s.owner, http.MethodGet, path, nil)
	require.Equal(s.t, http.StatusOK, list.Status)

	return list.Body["data"].([]any)
}

// answers builds the answers to the questions: the right options, or (when
// right is false) a wrong one.
func answers(questions []any, right bool) []any {
	var result []any

	for _, raw := range questions {
		question := raw.(map[string]any)

		var chosen []any

		for _, rawOption := range question["options"].([]any) {
			option := rawOption.(map[string]any)

			if option["is_correct"].(bool) == right {
				chosen = append(chosen, option["id"])

				if !right {
					break
				}
			}
		}

		result = append(result, map[string]any{"question_id": question["id"], "option_ids": chosen})
	}

	return result
}

func TestQuiz_Create(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := s.enrolled()

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPost, s.path("/quizzes"), quizBody(), "").Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodPost, s.path("/quizzes"), quizBody()).Status)
	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleInstructor), http.MethodPost, s.path("/quizzes"), quizBody()).Status)

	got := c.as(s.owner, http.MethodPost, s.path("/quizzes"), quizBody())
	require.Equal(t, http.StatusCreated, got.Status, got.Raw)
	require.Equal(t, s.id.String(), got.data()["course_id"])
	require.EqualValues(t, 70, got.data()["pass_threshold"])

	second := c.as(s.owner, http.MethodPost, s.path("/quizzes"), quizBody())
	require.Equal(t, http.StatusConflict, second.Status, "a course has one final quiz")
}

func TestQuiz_CreateValidation(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	got := c.as(s.owner, http.MethodPost, s.path("/quizzes"), map[string]any{"title": "x", "time_limit": 0, "pass_threshold": 150, "max_attempts": -1})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Contains(t, got.fields(), "time_limit")
	require.Contains(t, got.fields(), "pass_threshold")
	require.Contains(t, got.fields(), "max_attempts")

	empty := c.as(s.owner, http.MethodPost, s.path("/quizzes"), map[string]any{})
	require.Contains(t, empty.fields(), "title")
}

func TestQuiz_ModuleQuizAndReading(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := s.enrolled()
	outsider := c.e.NewUser(t, models.RoleStudent)

	modules := c.dataList(c.send(http.MethodGet, s.path("/modules"), nil, ""))
	moduleID := modules[0]["id"].(string)

	created := c.as(s.owner, http.MethodPost, "/api/v1/modules/"+moduleID+"/quizzes", quizBody())
	require.Equal(t, http.StatusCreated, created.Status, created.Raw)

	quizID := created.data()["id"].(string)

	require.Equal(t, http.StatusOK, c.as(student, http.MethodGet, "/api/v1/quizzes/"+quizID, nil).Status)
	require.Len(t, c.dataList(c.as(student, http.MethodGet, "/api/v1/modules/"+moduleID+"/quizzes", nil)), 1)

	require.Equal(t, http.StatusForbidden, c.as(outsider, http.MethodGet, "/api/v1/quizzes/"+quizID, nil).Status)
	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, "/api/v1/quizzes/"+quizID, nil, "").Status)
	require.Equal(t, http.StatusNotFound, c.as(student, http.MethodGet, "/api/v1/quizzes/"+uuid.NewString(), nil).Status)
}

func TestQuiz_UpdateAndDelete(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	quizID := s.createQuiz()
	path := "/api/v1/quizzes/" + quizID

	updated := c.as(s.owner, http.MethodPut, path, map[string]any{"max_attempts": 5})
	require.Equal(t, http.StatusOK, updated.Status)
	require.EqualValues(t, 5, updated.data()["max_attempts"])

	bad := c.as(s.owner, http.MethodPut, path, map[string]any{"pass_threshold": 101, "time_limit": 0})
	require.Equal(t, http.StatusBadRequest, bad.Status)

	other := c.e.NewUser(t, models.RoleInstructor)
	require.Equal(t, http.StatusForbidden, c.as(other, http.MethodPut, path, map[string]any{"max_attempts": 9}).Status)
	require.Equal(t, http.StatusForbidden, c.as(other, http.MethodDelete, path, nil).Status)

	require.Equal(t, http.StatusNoContent, c.as(s.owner, http.MethodDelete, path, nil).Status)
	require.Equal(t, http.StatusNotFound, c.as(s.owner, http.MethodGet, path, nil).Status)
}

func TestQuestions(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	quizID := s.createQuiz()
	path := "/api/v1/quizzes/" + quizID + "/questions"
	student := s.enrolled()

	questions := s.addQuestions(quizID)
	require.Len(t, questions, 2)

	first := questions[0].(map[string]any)
	require.Equal(t, "Capital of France?", first["text"])
	require.EqualValues(t, 1, first["order_number"])
	require.Contains(t, first["options"].([]any)[0], "is_correct", "the author sees the right answers")

	// students get the questions only inside an attempt
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, path, nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, "/api/v1/questions/"+first["id"].(string), nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodPost, path, map[string]any{}).Status)

	one := c.as(s.owner, http.MethodGet, "/api/v1/questions/"+first["id"].(string), nil)
	require.Equal(t, http.StatusOK, one.Status)

	updated := c.as(s.owner, http.MethodPut, "/api/v1/questions/"+first["id"].(string), map[string]any{"text": "Capital city of France?", "order_number": 2})
	require.Equal(t, http.StatusOK, updated.Status, updated.Raw)
	require.Equal(t, "Capital city of France?", updated.data()["text"])

	require.Equal(t, http.StatusNoContent, c.as(s.owner, http.MethodDelete, "/api/v1/questions/"+first["id"].(string), nil).Status)
	require.Len(t, c.dataList(c.as(s.owner, http.MethodGet, path, nil)), 1)
}

func TestQuestions_Validation(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	path := "/api/v1/quizzes/" + s.createQuiz() + "/questions"

	missing := c.as(s.owner, http.MethodPost, path, map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status)

	for _, field := range []string{"text", "type", "options"} {
		require.Contains(t, missing.fields(), field)
	}

	badType := c.as(s.owner, http.MethodPost, path, map[string]any{"text": "x", "type": "essay", "options": []any{option("a", true), option("b", false)}})
	require.Equal(t, http.StatusBadRequest, badType.Status)
	require.Contains(t, badType.fields()["type"], "one of")

	oneOption := c.as(s.owner, http.MethodPost, path, map[string]any{"text": "x", "type": "single_choice", "options": []any{option("a", true)}})
	require.Equal(t, http.StatusBadRequest, oneOption.Status)

	noText := c.as(s.owner, http.MethodPost, path, map[string]any{"text": "x", "type": "single_choice", "options": []any{map[string]any{"is_correct": true}, option("b", false)}})
	require.Equal(t, http.StatusBadRequest, noText.Status)

	twoRight := c.as(s.owner, http.MethodPost, path, map[string]any{"text": "x", "type": "single_choice", "options": []any{option("a", true), option("b", true)}})
	require.Equal(t, http.StatusBadRequest, twoRight.Status, "the service checks the right answers against the type")
	require.Contains(t, twoRight.errorMessage(), "exactly one")
}

func TestAttempts_TheWholeWay(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	quizID := s.createQuiz()
	questions := s.addQuestions(quizID)
	student := s.enrolled()

	start := "/api/v1/quizzes/" + quizID + "/attempts"

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPost, start, nil, "").Status)
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodPost, start, nil).Status, "the author does not take the quiz")
	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleStudent), http.MethodPost, start, nil).Status, "not enrolled")

	started := c.as(student, http.MethodPost, start, nil)
	require.Equal(t, http.StatusOK, started.Status, started.Raw)
	require.Len(t, started.data()["questions"], 2)
	require.NotContains(t, started.Raw, "is_correct", "the right answers must not reach the student")

	attemptID := started.data()["id"].(string)

	resumed := c.as(student, http.MethodPost, start, nil)
	require.Equal(t, attemptID, resumed.data()["id"], "the running attempt is continued")

	wrongOption := c.as(student, http.MethodPost, "/api/v1/attempts/"+attemptID+"/submit", map[string]any{
		"answers": []any{map[string]any{"question_id": questions[0].(map[string]any)["id"], "option_ids": []any{uuid.NewString()}}},
	})
	require.Equal(t, http.StatusBadRequest, wrongOption.Status)

	noOptions := c.as(student, http.MethodPost, "/api/v1/attempts/"+attemptID+"/submit", map[string]any{
		"answers": []any{map[string]any{"question_id": questions[0].(map[string]any)["id"], "option_ids": []any{}}},
	})
	require.Equal(t, http.StatusBadRequest, noOptions.Status)
	require.NotEmpty(t, noOptions.fields())

	other := c.as(s.enrolled(), http.MethodPost, "/api/v1/attempts/"+attemptID+"/submit", map[string]any{"answers": []any{}})
	require.Equal(t, http.StatusForbidden, other.Status, "this is not their attempt")

	submitted := c.as(student, http.MethodPost, "/api/v1/attempts/"+attemptID+"/submit", map[string]any{"answers": answers(questions, true)})
	require.Equal(t, http.StatusOK, submitted.Status, submitted.Raw)
	require.EqualValues(t, 100, submitted.data()["score"])
	require.Equal(t, true, submitted.data()["passed"])
	require.Len(t, submitted.data()["results"], 2)

	again := c.as(student, http.MethodPost, "/api/v1/attempts/"+attemptID+"/submit", map[string]any{"answers": answers(questions, true)})
	require.Equal(t, http.StatusConflict, again.Status)

	history := c.dataList(c.as(student, http.MethodGet, start, nil))
	require.Len(t, history, 1)
	require.Equal(t, true, history[0]["passed"])

	detail := c.as(student, http.MethodGet, "/api/v1/attempts/"+attemptID, nil)
	require.Equal(t, http.StatusOK, detail.Status)
	require.Len(t, detail.data()["answers"], 3, "one saved answer for every chosen option")

	// the author sees everybody's attempts; another student only their own
	require.Len(t, c.dataList(c.as(s.owner, http.MethodGet, start, nil)), 1)
	require.Equal(t, http.StatusOK, c.as(s.owner, http.MethodGet, "/api/v1/attempts/"+attemptID, nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(s.enrolled(), http.MethodGet, "/api/v1/attempts/"+attemptID, nil).Status)
}

func TestAttempts_WrongAnswersFailAndAttemptsRunOut(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	req := quizBody()
	req["max_attempts"] = 1

	created := c.as(s.owner, http.MethodPost, s.path("/quizzes"), req)
	require.Equal(t, http.StatusCreated, created.Status)

	quizID := created.data()["id"].(string)
	questions := s.addQuestions(quizID)
	student := s.enrolled()

	start := "/api/v1/quizzes/" + quizID + "/attempts"

	attempt := c.as(student, http.MethodPost, start, nil)

	result := c.as(student, http.MethodPost, "/api/v1/attempts/"+attempt.data()["id"].(string)+"/submit", map[string]any{"answers": answers(questions, false)})
	require.Equal(t, http.StatusOK, result.Status, result.Raw)
	require.EqualValues(t, 0, result.data()["score"])
	require.Equal(t, false, result.data()["passed"])

	noMore := c.as(student, http.MethodPost, start, nil)
	require.Equal(t, http.StatusForbidden, noMore.Status)
	require.Contains(t, noMore.errorMessage(), "no attempts left")
}

func TestAttempts_QuizWithoutQuestions(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	quizID := s.createQuiz()
	student := s.enrolled()

	got := c.as(student, http.MethodPost, "/api/v1/quizzes/"+quizID+"/attempts", nil)
	require.Equal(t, http.StatusBadRequest, got.Status)

	_ = testutil.Password
}
