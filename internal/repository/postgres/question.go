package postgres

import (
	"context"
	"errors"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type questionRepo struct {
	db DBTX
}

func NewQuestionRepository(db DBTX) *questionRepo {
	return &questionRepo{
		db: db,
	}
}

// scanQuestion scans a row selected with the question columns in the order
// used below. It works for both a single row and rows of a list.
func scanQuestion(row pgx.Row, question *models.Question) error {
	return row.Scan(
		&question.ID,
		&question.QuizID,
		&question.Text,
		&question.Type,
		&question.OrderNumber,
		&question.CreatedAt,
		&question.UpdatedAt,
	)
}

func (r *questionRepo) Create(
	ctx context.Context,
	question models.CreateQuestion,
) (uuid.UUID, error) {
	query := `
		INSERT INTO questions (
			id,
			quiz_id,
			text,
			type,
			order_number
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		question.QuizID,
		question.Text,
		question.Type,
		question.OrderNumber,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateQuestion",
				"question order number already exists in this quiz",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateQuestion",
				"quiz not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateQuestion",
			"failed to create question",
			err,
		)
	}

	return id, nil
}

func (r *questionRepo) CreateOption(
	ctx context.Context,
	option models.CreateQuestionOption,
) error {
	query := `
		INSERT INTO question_options (
			id,
			question_id,
			option_text,
			is_correct
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		uuid.New(),
		option.QuestionID,
		option.OptionText,
		option.IsCorrect,
	)

	if err != nil {
		if pgerr.IsForeignKeyViolation(err) {
			return apperror.NotFound(
				"repository",
				"CreateQuestionOption",
				"question not found",
				apperror.ErrNotFound,
			)
		}

		return apperror.Internal(
			"repository",
			"CreateQuestionOption",
			"failed to create question option",
			err,
		)
	}

	return nil
}

func (r *questionRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Question, error) {
	query := `
		SELECT
			id,
			quiz_id,
			text,
			type,
			order_number,
			created_at,
			updated_at
		FROM questions
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var question models.Question

	err := scanQuestion(r.db.QueryRow(ctx, query, id), &question)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetQuestionByID",
				"question not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetQuestionByID",
			"failed to get question",
			err,
		)
	}

	return &question, nil
}

func (r *questionRepo) GetListByQuizID(
	ctx context.Context,
	quizID uuid.UUID,
) ([]models.Question, error) {
	query := `
		SELECT
			id,
			quiz_id,
			text,
			type,
			order_number,
			created_at,
			updated_at
		FROM questions
		WHERE quiz_id = $1
			AND deleted_at IS NULL
		ORDER BY order_number;
	`

	rows, err := r.db.Query(ctx, query, quizID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetQuestionList",
			"failed to get questions",
			err,
		)
	}
	defer rows.Close()

	questions := make([]models.Question, 0)

	for rows.Next() {
		var question models.Question

		if err := scanQuestion(rows, &question); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetQuestionList",
				"failed to scan questions",
				err,
			)
		}

		questions = append(questions, question)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetQuestionList",
			"failed to read questions",
			err,
		)
	}

	return questions, nil
}

func (r *questionRepo) GetOptions(
	ctx context.Context,
	questionID uuid.UUID,
) ([]models.QuestionOption, error) {
	query := `
		SELECT
			id,
			question_id,
			option_text,
			is_correct
		FROM question_options
		WHERE question_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at;
	`

	return r.getOptions(ctx, "GetQuestionOptions", query, questionID)
}

// GetOptionsByQuizID returns the options of every question of the quiz, used
// to build the quiz and to score an attempt.
func (r *questionRepo) GetOptionsByQuizID(
	ctx context.Context,
	quizID uuid.UUID,
) ([]models.QuestionOption, error) {
	query := `
		SELECT
			o.id,
			o.question_id,
			o.option_text,
			o.is_correct
		FROM question_options o
		JOIN questions q ON q.id = o.question_id
		WHERE q.quiz_id = $1
			AND q.deleted_at IS NULL
			AND o.deleted_at IS NULL
		ORDER BY q.order_number, o.created_at;
	`

	return r.getOptions(ctx, "GetQuizOptions", query, quizID)
}

func (r *questionRepo) getOptions(
	ctx context.Context,
	op string,
	query string,
	ownerID uuid.UUID,
) ([]models.QuestionOption, error) {
	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to get question options",
			err,
		)
	}
	defer rows.Close()

	options := make([]models.QuestionOption, 0)

	for rows.Next() {
		var option models.QuestionOption

		err := rows.Scan(
			&option.ID,
			&option.QuestionID,
			&option.OptionText,
			&option.IsCorrect,
		)
		if err != nil {
			return nil, apperror.Internal(
				"repository",
				op,
				"failed to scan question options",
				err,
			)
		}

		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to read question options",
			err,
		)
	}

	return options, nil
}

func (r *questionRepo) Update(
	ctx context.Context,
	question models.UpdateQuestion,
) (*models.Question, error) {
	query := `
		UPDATE questions
		SET
			text = COALESCE($2, text),
			type = COALESCE($3, type),
			order_number = COALESCE($4, order_number),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			quiz_id,
			text,
			type,
			order_number,
			created_at,
			updated_at;
	`

	var updatedQuestion models.Question

	err := scanQuestion(
		r.db.QueryRow(
			ctx,
			query,
			question.ID,
			question.Text,
			question.Type,
			question.OrderNumber,
		),
		&updatedQuestion,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateQuestion",
				"question not found",
				apperror.ErrNotFound,
			)
		}

		if pgerr.IsUniqueViolation(err) {
			return nil, apperror.Conflict(
				"repository",
				"UpdateQuestion",
				"question order number already exists in this quiz",
				apperror.ErrAlreadyExists,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateQuestion",
			"failed to update question",
			err,
		)
	}

	return &updatedQuestion, nil
}

func (r *questionRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE questions
		SET
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		id,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteQuestion",
			"failed to delete question",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteQuestion",
			"question not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

// DeleteOptions soft-deletes the options of a question. They are not removed
// for real because past attempt answers still reference them.
func (r *questionRepo) DeleteOptions(
	ctx context.Context,
	questionID uuid.UUID,
) error {
	query := `
		UPDATE question_options
		SET
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE question_id = $1
			AND deleted_at IS NULL;
	`

	_, err := r.db.Exec(
		ctx,
		query,
		questionID,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteQuestionOptions",
			"failed to delete question options",
			err,
		)
	}

	return nil
}
