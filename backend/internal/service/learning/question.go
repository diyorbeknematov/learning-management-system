package learning

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

// Question is the part of the quiz the author works with. Students get the
// questions only inside an attempt (see Attempt), so everything here is for
// the owner of the course and the SuperAdmin.
type Question struct {
	repo *repo.Repository
}

func NewQuestion(deps core.Dependencies) *Question {
	return &Question{
		repo: deps.Repo,
	}
}

func questionIDs(questions []models.Question) []uuid.UUID {
	ids := make([]uuid.UUID, len(questions))

	for i, question := range questions {
		ids[i] = question.ID
	}

	return ids
}

func setQuestionOrder(tx *repo.Repository) func(context.Context, uuid.UUID, int) error {
	return func(ctx context.Context, id uuid.UUID, number int) error {
		order := number

		_, err := tx.Question.Update(ctx, models.UpdateQuestion{ID: id, OrderNumber: &order})

		return err
	}
}

// validateOptions checks the options against the type of the question:
// single choice has exactly one right option, multiple choice at least one,
// and true or false has exactly two options with one right.
func validateOptions(questionType models.QuestionType, options []models.CreateQuestionOption, op string) error {
	correct := 0

	for _, option := range options {
		if option.IsCorrect {
			correct++
		}
	}

	invalid := func(message string) error {
		return apperror.InvalidInput("service", op, message, apperror.ErrInvalidInput)
	}

	if len(options) < 2 {
		return invalid("a question needs at least two options")
	}

	switch questionType {
	case models.QuestionTypeSingleChoice:
		if correct != 1 {
			return invalid("a single choice question needs exactly one right option")
		}
	case models.QuestionTypeMultipleChoice:
		if correct < 1 {
			return invalid("a multiple choice question needs at least one right option")
		}
	case models.QuestionTypeTrueFalse:
		if len(options) != 2 || correct != 1 {
			return invalid("a true or false question needs two options and one of them right")
		}
	default:
		return invalid("unknown question type")
	}

	return nil
}

// manageQuiz loads the quiz and checks that the actor owns its course.
func (s *Question) manageQuiz(ctx context.Context, actor models.Actor, quizID uuid.UUID, op string) (*models.Quiz, error) {
	quiz, err := s.repo.Quiz.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	course, err := quizCourse(ctx, s.repo, quiz)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, op); err != nil {
		return nil, err
	}

	return quiz, nil
}

// withOptions loads the options of the questions into them.
func (s *Question) withOptions(ctx context.Context, quizID uuid.UUID, questions []models.Question) ([]models.Question, error) {
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
		if questions[i].Options == nil {
			questions[i].Options = []models.QuestionOption{}
		}
	}

	return questions, nil
}

// Create adds a question with its options to a quiz.
func (s *Question) Create(ctx context.Context, actor models.Actor, quizID uuid.UUID, req models.CreateQuestionRequest) (*models.Question, error) {
	if _, err := s.manageQuiz(ctx, actor, quizID, "CreateQuestion"); err != nil {
		return nil, err
	}

	if err := validateOptions(req.Type, req.Options, "CreateQuestion"); err != nil {
		return nil, err
	}

	var id uuid.UUID

	err := s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		existing, err := tx.Question.GetListByQuizID(ctx, quizID)
		if err != nil {
			return err
		}

		position := len(existing) + 1
		if req.OrderNumber != nil && *req.OrderNumber < position {
			position = *req.OrderNumber
		}

		id, err = tx.Question.Create(ctx, models.CreateQuestion{
			QuizID:      quizID,
			Text:        req.Text,
			Type:        req.Type,
			OrderNumber: core.TempOrderBase * 2,
		})
		if err != nil {
			return err
		}

		if err := createOptions(ctx, tx, id, req.Options); err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(questionIDs(existing), id, position), setQuestionOrder(tx))
	})
	if err != nil {
		return nil, err
	}

	return s.load(ctx, id)
}

func createOptions(ctx context.Context, tx *repo.Repository, questionID uuid.UUID, options []models.CreateQuestionOption) error {
	for i, option := range options {
		option.QuestionID = questionID
		option.Position = i + 1

		if err := tx.Question.CreateOption(ctx, option); err != nil {
			return err
		}
	}

	return nil
}

// load returns a question with its options and the right answers marked.
func (s *Question) load(ctx context.Context, id uuid.UUID) (*models.Question, error) {
	question, err := s.repo.Question.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	options, err := s.repo.Question.GetOptions(ctx, id)
	if err != nil {
		return nil, err
	}

	question.Options = options

	return question, nil
}

func (s *Question) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.Question, error) {
	question, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.manageQuiz(ctx, actor, question.QuizID, "GetQuestion"); err != nil {
		return nil, err
	}

	return question, nil
}

// GetListByQuiz returns the questions in the order of the author, with the
// right options marked.
func (s *Question) GetListByQuiz(ctx context.Context, actor models.Actor, quizID uuid.UUID) ([]models.Question, error) {
	if _, err := s.manageQuiz(ctx, actor, quizID, "GetQuestionList"); err != nil {
		return nil, err
	}

	questions, err := s.repo.Question.GetListByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	return s.withOptions(ctx, quizID, questions)
}

// Update changes a question. Options that are sent replace the old ones; the
// old options stay hidden in the storage because the past attempts point at
// them. Whatever the result is, the options must still fit the type.
func (s *Question) Update(ctx context.Context, actor models.Actor, id uuid.UUID, req models.UpdateQuestion) (*models.Question, error) {
	question, err := s.repo.Question.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.manageQuiz(ctx, actor, question.QuizID, "UpdateQuestion"); err != nil {
		return nil, err
	}

	if err := s.checkUpdatedOptions(ctx, question, req); err != nil {
		return nil, err
	}

	newOptions, newOrder := req.Options, req.OrderNumber

	// the order is changed separately, by shifting the other questions
	req.ID, req.OrderNumber, req.Options = id, nil, nil

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if _, err := tx.Question.Update(ctx, req); err != nil {
			return err
		}

		if newOptions != nil {
			if err := tx.Question.DeleteOptions(ctx, id); err != nil {
				return err
			}

			if err := createOptions(ctx, tx, id, *newOptions); err != nil {
				return err
			}
		}

		if newOrder == nil {
			return nil
		}

		questions, err := tx.Question.GetListByQuizID(ctx, question.QuizID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, core.MoveTo(questionIDs(questions), id, *newOrder), setQuestionOrder(tx))
	})
	if err != nil {
		return nil, err
	}

	return s.load(ctx, id)
}

// checkUpdatedOptions makes sure the question is still valid after the update:
// the new options must fit the type, and so must the old options when only
// the type changes.
func (s *Question) checkUpdatedOptions(ctx context.Context, question *models.Question, req models.UpdateQuestion) error {
	if req.Options == nil && req.Type == nil {
		return nil
	}

	questionType := question.Type
	if req.Type != nil {
		questionType = *req.Type
	}

	if req.Options != nil {
		return validateOptions(questionType, *req.Options, "UpdateQuestion")
	}

	current, err := s.repo.Question.GetOptions(ctx, question.ID)
	if err != nil {
		return err
	}

	options := make([]models.CreateQuestionOption, len(current))

	for i, option := range current {
		options[i] = models.CreateQuestionOption{
			OptionText: option.OptionText,
			IsCorrect:  option.IsCorrect != nil && *option.IsCorrect,
		}
	}

	return validateOptions(questionType, options, "UpdateQuestion")
}

// Delete removes the question and closes the gap in the order.
func (s *Question) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	question, err := s.repo.Question.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if _, err := s.manageQuiz(ctx, actor, question.QuizID, "DeleteQuestion"); err != nil {
		return err
	}

	return s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if err := tx.Question.Delete(ctx, id); err != nil {
			return err
		}

		remaining, err := tx.Question.GetListByQuizID(ctx, question.QuizID)
		if err != nil {
			return err
		}

		return core.Renumber(ctx, questionIDs(remaining), setQuestionOrder(tx))
	})
}
