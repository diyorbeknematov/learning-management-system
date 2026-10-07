package postgres

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type attemptRepo struct {
	db DBTX
}

func NewAttemptRepository(db DBTX) *attemptRepo {
	return &attemptRepo{
		db: db,
	}
}

// scanAttempt scans a row selected with the attempt columns in the order used
// below. It works for both a single row and rows of a list.
func scanAttempt(row pgx.Row, attempt *models.QuizAttempt) error {
	return row.Scan(
		&attempt.ID,
		&attempt.StudentID,
		&attempt.QuizID,
		&attempt.AttemptNumber,
		&attempt.Score,
		&attempt.StartedAt,
		&attempt.CompletedAt,
		&attempt.TimeSpent,
		&attempt.CreatedAt,
	)
}

func (r *attemptRepo) Create(
	ctx context.Context,
	attempt models.CreateQuizAttempt,
) (uuid.UUID, error) {
	query := `
		INSERT INTO quiz_attempts (
			id,
			student_id,
			quiz_id,
			attempt_number,
			started_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		attempt.StudentID,
		attempt.QuizID,
		attempt.AttemptNumber,
		attempt.StartedAt,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, handleError(err, "CreateQuizAttempt", "failed to create quiz attempt")
	}

	return id, nil
}

func (r *attemptRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.QuizAttempt, error) {
	query := `
		SELECT
			id,
			student_id,
			quiz_id,
			attempt_number,
			score,
			started_at,
			completed_at,
			time_spent,
			created_at
		FROM quiz_attempts
		WHERE id = $1;
	`

	var attempt models.QuizAttempt

	err := scanAttempt(r.db.QueryRow(ctx, query, id), &attempt)

	if err != nil {
		return nil, handleError(err, "GetQuizAttemptByID", "failed to get quiz attempt")
	}

	return &attempt, nil
}

// GetList returns the attempts of a quiz. When studentID is set only that
// student's attempts are returned.
func (r *attemptRepo) GetList(
	ctx context.Context,
	quizID uuid.UUID,
	studentID *uuid.UUID,
) ([]models.QuizAttempt, error) {
	query := `
		SELECT
			id,
			student_id,
			quiz_id,
			attempt_number,
			score,
			started_at,
			completed_at,
			time_spent,
			created_at
		FROM quiz_attempts
		WHERE quiz_id = $1
			AND ($2::uuid IS NULL OR student_id = $2)
		ORDER BY started_at DESC;
	`

	rows, err := r.db.Query(ctx, query, quizID, studentID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetQuizAttemptList",
			"failed to get quiz attempts",
			err,
		)
	}
	defer rows.Close()

	attempts := make([]models.QuizAttempt, 0)

	for rows.Next() {
		var attempt models.QuizAttempt

		if err := scanAttempt(rows, &attempt); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetQuizAttemptList",
				"failed to scan quiz attempts",
				err,
			)
		}

		attempts = append(attempts, attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetQuizAttemptList",
			"failed to read quiz attempts",
			err,
		)
	}

	return attempts, nil
}

// CountByStudent returns how many attempts the student already made on the
// quiz, used to check max_attempts and to number the next attempt.
func (r *attemptRepo) CountByStudent(
	ctx context.Context,
	quizID uuid.UUID,
	studentID uuid.UUID,
) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM quiz_attempts
		WHERE quiz_id = $1
			AND student_id = $2;
	`

	var count int

	err := r.db.QueryRow(
		ctx,
		query,
		quizID,
		studentID,
	).Scan(&count)

	if err != nil {
		return 0, apperror.Internal(
			"repository",
			"CountQuizAttempts",
			"failed to count quiz attempts",
			err,
		)
	}

	return count, nil
}

// Complete stores the result of an attempt. An attempt can be completed only
// once, so a second call finds nothing to update.
func (r *attemptRepo) Complete(
	ctx context.Context,
	attempt models.CompleteAttempt,
) error {
	query := `
		UPDATE quiz_attempts
		SET
			score = $2,
			completed_at = $3,
			time_spent = $4
		WHERE id = $1
			AND completed_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		attempt.ID,
		attempt.Score,
		attempt.CompletedAt,
		attempt.TimeSpent,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"CompleteQuizAttempt",
			"failed to complete quiz attempt",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"CompleteQuizAttempt",
			"quiz attempt not found or already completed",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *attemptRepo) CreateAnswer(
	ctx context.Context,
	answer models.AttemptAnswer,
) error {
	query := `
		INSERT INTO attempt_answers (
			id,
			attempt_id,
			question_id,
			option_id
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		uuid.New(),
		answer.AttemptID,
		answer.QuestionID,
		answer.OptionID,
	)

	if err != nil {
		return handleError(err, "CreateAttemptAnswer", "failed to create attempt answer")
	}

	return nil
}

func (r *attemptRepo) GetAnswers(
	ctx context.Context,
	attemptID uuid.UUID,
) ([]models.AttemptAnswer, error) {
	query := `
		SELECT
			id,
			attempt_id,
			question_id,
			option_id,
			created_at
		FROM attempt_answers
		WHERE attempt_id = $1
		ORDER BY created_at;
	`

	rows, err := r.db.Query(ctx, query, attemptID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetAttemptAnswers",
			"failed to get attempt answers",
			err,
		)
	}
	defer rows.Close()

	answers := make([]models.AttemptAnswer, 0)

	for rows.Next() {
		var answer models.AttemptAnswer

		err := rows.Scan(
			&answer.ID,
			&answer.AttemptID,
			&answer.QuestionID,
			&answer.OptionID,
			&answer.CreatedAt,
		)
		if err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetAttemptAnswers",
				"failed to scan attempt answers",
				err,
			)
		}

		answers = append(answers, answer)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetAttemptAnswers",
			"failed to read attempt answers",
			err,
		)
	}

	return answers, nil
}
