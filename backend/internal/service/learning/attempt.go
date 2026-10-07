package learning

import (
	"context"
	"encoding/binary"
	"math"
	"math/rand"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

// submitGrace is how long after the time limit a submission is still taken,
// for the delay of the network.
const submitGrace = 30 * time.Second

type Attempt struct {
	repo *repo.Repository
	cfg  core.Config
}

func NewAttempt(deps core.Dependencies) *Attempt {
	return &Attempt{
		repo: deps.Repo,
		cfg:  deps.Config,
	}
}

func deadline(attempt *models.QuizAttempt, quiz *models.Quiz) time.Time {
	return attempt.StartedAt.Add(time.Duration(quiz.TimeLimit) * time.Minute)
}

// fillPassed marks which finished attempts reached the pass threshold.
func fillPassed(attempts []models.QuizAttempt, quiz *models.Quiz) {
	for i := range attempts {
		if attempts[i].Score != nil {
			passed := *attempts[i].Score >= quiz.PassThreshold
			attempts[i].Passed = &passed
		}
	}
}

// expire closes an attempt whose time is over with no points. Such an attempt
// is not deleted, so it still counts against max_attempts.
func (s *Attempt) expire(ctx context.Context, attempt *models.QuizAttempt, quiz *models.Quiz) error {
	err := s.repo.Attempt.Complete(ctx, models.CompleteAttempt{
		ID:          attempt.ID,
		Score:       0,
		CompletedAt: time.Now(),
		TimeSpent:   quiz.TimeLimit * 60,
	})
	if err != nil && !core.IsNotFound(err) {
		return err
	}

	return nil
}

// Start begins an attempt for the student, or continues the one that is still
// running. The student gets the questions in a random order that stays the
// same for this attempt, and never the right answers.
func (s *Attempt) Start(ctx context.Context, actor models.Actor, quizID uuid.UUID) (*models.AttemptView, error) {
	quiz, err := s.repo.Quiz.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return nil, err
	}

	if course.Status != models.CourseStatusPublished {
		return nil, apperror.NotFound("service", "StartAttempt", "not found", apperror.ErrNotFound)
	}

	if _, err := core.ActiveEnrollment(ctx, s.repo, actor.UserID, course.ID, "StartAttempt"); err != nil {
		return nil, err
	}

	attempts, err := s.repo.Attempt.GetList(ctx, quizID, &actor.UserID)
	if err != nil {
		return nil, err
	}

	for i := range attempts {
		if attempts[i].CompletedAt != nil {
			continue
		}

		if time.Now().Before(deadline(&attempts[i], quiz)) {
			return s.view(ctx, &attempts[i], quiz)
		}

		if err := s.expire(ctx, &attempts[i], quiz); err != nil {
			return nil, err
		}
	}

	if len(attempts) >= quiz.MaxAttempts {
		return nil, apperror.Forbidden("service", "StartAttempt", "you have no attempts left", apperror.ErrForbidden)
	}

	id, err := s.repo.Attempt.Create(ctx, models.CreateQuizAttempt{
		StudentID:     actor.UserID,
		QuizID:        quizID,
		AttemptNumber: len(attempts) + 1,
		StartedAt:     time.Now(),
	})
	if err != nil {
		return nil, err
	}

	attempt, err := s.repo.Attempt.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.view(ctx, attempt, quiz)
}

// view builds what the student sees during an attempt.
func (s *Attempt) view(ctx context.Context, attempt *models.QuizAttempt, quiz *models.Quiz) (*models.AttemptView, error) {
	questions, err := s.questions(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, apperror.InvalidInput("service", "StartAttempt", "the quiz has no questions yet", apperror.ErrInvalidInput)
	}

	shuffle(attempt.ID, questions)

	for i := range questions {
		// true or false keeps its natural order, the choices are shuffled
		if questions[i].Type != models.QuestionTypeTrueFalse {
			shuffleOptions(attempt.ID, questions[i].ID, questions[i].Options)
		}

		for j := range questions[i].Options {
			questions[i].Options[j].IsCorrect = nil
		}
	}

	return &models.AttemptView{QuizAttempt: *attempt, Questions: questions}, nil
}

// questions returns the questions of a quiz with their options, right answers
// included.
func (s *Attempt) questions(ctx context.Context, quizID uuid.UUID) ([]models.Question, error) {
	questions, err := s.repo.Question.GetListByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	options, err := s.repo.Question.GetOptionsByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	byQuestion := map[uuid.UUID][]models.QuestionOption{}
	for _, option := range options {
		byQuestion[option.QuestionID] = append(byQuestion[option.QuestionID], option)
	}

	for i := range questions {
		questions[i].Options = byQuestion[questions[i].ID]
	}

	return questions, nil
}

// random returns a generator that gives the same numbers for the same ids, so
// an attempt shows the same order every time it is opened.
func random(ids ...uuid.UUID) *rand.Rand {
	var seed uint64

	for _, id := range ids {
		seed = seed*31 + binary.BigEndian.Uint64(id[:8]) ^ binary.BigEndian.Uint64(id[8:])
	}

	return rand.New(rand.NewSource(int64(seed)))
}

func shuffle(attemptID uuid.UUID, questions []models.Question) {
	random(attemptID).Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})
}

func shuffleOptions(attemptID, questionID uuid.UUID, options []models.QuestionOption) {
	random(attemptID, questionID).Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})
}

// GetList returns the attempts at a quiz: a student sees their own, the owner
// of the course and the SuperAdmin see everybody's.
func (s *Attempt) GetList(ctx context.Context, actor models.Actor, quizID uuid.UUID) ([]models.QuizAttempt, error) {
	quiz, err := s.repo.Quiz.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return nil, err
	}

	var studentID *uuid.UUID

	if core.CanManage(actor, course.InstructorID, "GetAttemptList") != nil {
		if _, err := core.ActiveEnrollment(ctx, s.repo, actor.UserID, course.ID, "GetAttemptList"); err != nil {
			return nil, err
		}

		studentID = &actor.UserID
	}

	attempts, err := s.repo.Attempt.GetList(ctx, quizID, studentID)
	if err != nil {
		return nil, err
	}

	fillPassed(attempts, quiz)

	return attempts, nil
}

// GetByID returns an attempt with the options the student chose, to its owner,
// the owner of the course and the SuperAdmin.
func (s *Attempt) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.AttemptDetail, error) {
	attempt, err := s.repo.Attempt.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	quiz, err := s.repo.Quiz.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}

	if attempt.StudentID != actor.UserID {
		course, err := quizCourse(ctx, s.repo, quiz)
		if err != nil {
			return nil, err
		}

		if err := core.CanManage(actor, course.InstructorID, "GetAttempt"); err != nil {
			return nil, err
		}
	}

	answers, err := s.repo.Attempt.GetAnswers(ctx, id)
	if err != nil {
		return nil, err
	}

	attempts := []models.QuizAttempt{*attempt}
	fillPassed(attempts, quiz)

	return &models.AttemptDetail{QuizAttempt: attempts[0], Answers: answers}, nil
}

// Submit takes the answers of the student, scores the attempt and closes it.
// An answer is right only when the chosen options are exactly the right ones;
// a question left unanswered is wrong.
func (s *Attempt) Submit(ctx context.Context, actor models.Actor, req models.SubmitAttempt) (*models.AttemptResult, error) {
	attempt, err := s.repo.Attempt.GetByID(ctx, req.AttemptID)
	if err != nil {
		return nil, err
	}

	if attempt.StudentID != actor.UserID {
		return nil, apperror.Forbidden("service", "SubmitAttempt", "this is not your attempt", apperror.ErrForbidden)
	}

	if attempt.CompletedAt != nil {
		return nil, apperror.Conflict("service", "SubmitAttempt", "the attempt is already submitted", apperror.ErrAlreadyExists)
	}

	quiz, err := s.repo.Quiz.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(deadline(attempt, quiz).Add(submitGrace)) {
		if err := s.expire(ctx, attempt, quiz); err != nil {
			return nil, err
		}

		return nil, apperror.Forbidden("service", "SubmitAttempt", "the time is over", apperror.ErrForbidden)
	}

	questions, err := s.questions(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}

	chosen, err := chosenOptions(questions, req.Answers)
	if err != nil {
		return nil, err
	}

	results, correct := score(questions, chosen)

	points := 0
	if len(questions) > 0 {
		points = int(math.Round(float64(correct) * 100 / float64(len(questions))))
	}

	now := time.Now()

	spent := int(now.Sub(attempt.StartedAt).Seconds())
	if limit := quiz.TimeLimit * 60; spent > limit {
		spent = limit
	}

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		for questionID, optionIDs := range chosen {
			for _, optionID := range optionIDs {
				err := tx.Attempt.CreateAnswer(ctx, models.AttemptAnswer{
					AttemptID:  attempt.ID,
					QuestionID: questionID,
					OptionID:   optionID,
				})
				if err != nil {
					return err
				}
			}
		}

		err := tx.Attempt.Complete(ctx, models.CompleteAttempt{
			ID:          attempt.ID,
			Score:       points,
			CompletedAt: now,
			TimeSpent:   spent,
		})
		if core.IsNotFound(err) {
			// another request submitted this attempt first
			return apperror.Conflict("service", "SubmitAttempt", "the attempt is already submitted", apperror.ErrAlreadyExists)
		}

		if err != nil {
			return err
		}

		// passing the final quiz can complete the course
		course, err := quizCourse(ctx, tx, quiz)
		if err != nil {
			return err
		}

		return syncCompletion(ctx, tx, s.cfg, actor.UserID, course)
	})
	if err != nil {
		return nil, err
	}

	finished, err := s.repo.Attempt.GetByID(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}

	attempts := []models.QuizAttempt{*finished}
	fillPassed(attempts, quiz)

	return &models.AttemptResult{QuizAttempt: attempts[0], Results: results}, nil
}

// chosenOptions checks the answers and returns the chosen options of every
// answered question. An answer for a question that is not in the quiz, an
// option of another question, two answers for one question, or several
// options for a question with one right answer are rejected.
func chosenOptions(questions []models.Question, answers []models.SubmitAnswer) (map[uuid.UUID][]uuid.UUID, error) {
	invalid := func(message string) error {
		return apperror.InvalidInput("service", "SubmitAttempt", message, apperror.ErrInvalidInput)
	}

	byID := map[uuid.UUID]models.Question{}
	for _, question := range questions {
		byID[question.ID] = question
	}

	chosen := map[uuid.UUID][]uuid.UUID{}

	for _, answer := range answers {
		question, ok := byID[answer.QuestionID]
		if !ok {
			return nil, invalid("an answer is for a question that is not in this quiz")
		}

		if _, answered := chosen[answer.QuestionID]; answered {
			return nil, invalid("a question is answered twice")
		}

		if question.Type != models.QuestionTypeMultipleChoice && len(answer.OptionIDs) != 1 {
			return nil, invalid("this question takes exactly one option")
		}

		belongs := map[uuid.UUID]bool{}
		for _, option := range question.Options {
			belongs[option.ID] = true
		}

		seen := map[uuid.UUID]bool{}

		for _, optionID := range answer.OptionIDs {
			if !belongs[optionID] {
				return nil, invalid("an option does not belong to its question")
			}

			if seen[optionID] {
				return nil, invalid("an option is chosen twice")
			}

			seen[optionID] = true
		}

		chosen[answer.QuestionID] = answer.OptionIDs
	}

	return chosen, nil
}

// score marks every question and counts the right ones.
func score(questions []models.Question, chosen map[uuid.UUID][]uuid.UUID) ([]models.QuestionResult, int) {
	results := make([]models.QuestionResult, 0, len(questions))
	correct := 0

	for _, question := range questions {
		selected := map[uuid.UUID]bool{}
		for _, optionID := range chosen[question.ID] {
			selected[optionID] = true
		}

		right := len(selected) > 0

		for _, option := range question.Options {
			isCorrect := option.IsCorrect != nil && *option.IsCorrect

			if isCorrect != selected[option.ID] {
				right = false
			}
		}

		if right {
			correct++
		}

		ids := chosen[question.ID]
		if ids == nil {
			ids = []uuid.UUID{}
		}

		results = append(results, models.QuestionResult{QuestionID: question.ID, Correct: right, SelectedOptionIDs: ids})
	}

	return results, correct
}
