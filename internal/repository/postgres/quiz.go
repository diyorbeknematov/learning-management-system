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

type quizRepo struct {
	db DBTX
}

func NewQuizRepository(db DBTX) *quizRepo {
	return &quizRepo{
		db: db,
	}
}

// scanQuiz scans a row selected with the quiz columns in the order used
// below. It works for both a single row and rows of a list.
func scanQuiz(row pgx.Row, quiz *models.Quiz) error {
	return row.Scan(
		&quiz.ID,
		&quiz.CourseID,
		&quiz.ModuleID,
		&quiz.Title,
		&quiz.Description,
		&quiz.TimeLimit,
		&quiz.PassThreshold,
		&quiz.MaxAttempts,
		&quiz.CreatedAt,
		&quiz.UpdatedAt,
	)
}

func (r *quizRepo) Create(
	ctx context.Context,
	quiz models.CreateQuiz,
) (uuid.UUID, error) {
	query := `
		INSERT INTO quizzes (
			id,
			course_id,
			module_id,
			title,
			description,
			time_limit,
			pass_threshold,
			max_attempts
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		quiz.CourseID,
		quiz.ModuleID,
		quiz.Title,
		quiz.Description,
		quiz.TimeLimit,
		quiz.PassThreshold,
		quiz.MaxAttempts,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateQuiz",
				"course already has a final quiz",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreateQuiz",
				"invalid quiz owner, time limit, threshold or attempts",
				apperror.ErrInvalidInput,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateQuiz",
				"course or module not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateQuiz",
			"failed to create quiz",
			err,
		)
	}

	return id, nil
}

func (r *quizRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Quiz, error) {
	query := `
		SELECT
			id,
			course_id,
			module_id,
			title,
			description,
			time_limit,
			pass_threshold,
			max_attempts,
			created_at,
			updated_at
		FROM quizzes
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var quiz models.Quiz

	err := scanQuiz(r.db.QueryRow(ctx, query, id), &quiz)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetQuizByID",
				"quiz not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetQuizByID",
			"failed to get quiz",
			err,
		)
	}

	return &quiz, nil
}

func (r *quizRepo) GetListByCourseID(
	ctx context.Context,
	courseID uuid.UUID,
) ([]models.Quiz, error) {
	query := `
		SELECT
			id,
			course_id,
			module_id,
			title,
			description,
			time_limit,
			pass_threshold,
			max_attempts,
			created_at,
			updated_at
		FROM quizzes
		WHERE course_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at;
	`

	return r.getList(ctx, "GetQuizListByCourseID", query, courseID)
}

func (r *quizRepo) GetListByModuleID(
	ctx context.Context,
	moduleID uuid.UUID,
) ([]models.Quiz, error) {
	query := `
		SELECT
			id,
			course_id,
			module_id,
			title,
			description,
			time_limit,
			pass_threshold,
			max_attempts,
			created_at,
			updated_at
		FROM quizzes
		WHERE module_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at;
	`

	return r.getList(ctx, "GetQuizListByModuleID", query, moduleID)
}

func (r *quizRepo) getList(
	ctx context.Context,
	op string,
	query string,
	ownerID uuid.UUID,
) ([]models.Quiz, error) {
	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to get quizzes",
			err,
		)
	}
	defer rows.Close()

	quizzes := make([]models.Quiz, 0)

	for rows.Next() {
		var quiz models.Quiz

		if err := scanQuiz(rows, &quiz); err != nil {
			return nil, apperror.Internal(
				"repository",
				op,
				"failed to scan quizzes",
				err,
			)
		}

		quizzes = append(quizzes, quiz)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to read quizzes",
			err,
		)
	}

	return quizzes, nil
}

func (r *quizRepo) Update(
	ctx context.Context,
	quiz models.UpdateQuiz,
) (*models.Quiz, error) {
	query := `
		UPDATE quizzes
		SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			time_limit = COALESCE($4, time_limit),
			pass_threshold = COALESCE($5, pass_threshold),
			max_attempts = COALESCE($6, max_attempts),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			course_id,
			module_id,
			title,
			description,
			time_limit,
			pass_threshold,
			max_attempts,
			created_at,
			updated_at;
	`

	var updatedQuiz models.Quiz

	err := scanQuiz(
		r.db.QueryRow(
			ctx,
			query,
			quiz.ID,
			quiz.Title,
			quiz.Description,
			quiz.TimeLimit,
			quiz.PassThreshold,
			quiz.MaxAttempts,
		),
		&updatedQuiz,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateQuiz",
				"quiz not found",
				apperror.ErrNotFound,
			)
		}

		if pgerr.IsCheckViolation(err) {
			return nil, apperror.InvalidInput(
				"repository",
				"UpdateQuiz",
				"invalid time limit, threshold or attempts",
				apperror.ErrInvalidInput,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateQuiz",
			"failed to update quiz",
			err,
		)
	}

	return &updatedQuiz, nil
}

func (r *quizRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE quizzes
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
			"DeleteQuiz",
			"failed to delete quiz",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteQuiz",
			"quiz not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
