package postgres

import (
	"context"
	"errors"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type reviewRepo struct {
	db DBTX
}

func NewReviewRepository(db DBTX) *reviewRepo {
	return &reviewRepo{
		db: db,
	}
}

// scanReview scans a row selected with the review columns and the student
// name in the order used below. It works for both a single row and rows of a
// list.
func scanReview(row pgx.Row, review *models.Review) error {
	return row.Scan(
		&review.ID,
		&review.StudentID,
		&review.CourseID,
		&review.StudentName,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
		&review.UpdatedAt,
	)
}

func (r *reviewRepo) Create(
	ctx context.Context,
	review models.CreateReview,
) (uuid.UUID, error) {
	query := `
		INSERT INTO reviews (
			id,
			student_id,
			course_id,
			rating,
			comment
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		review.StudentID,
		review.CourseID,
		review.Rating,
		review.Comment,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateReview",
				"student already reviewed this course",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreateReview",
				"rating must be between 1 and 5",
				apperror.ErrInvalidInput,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateReview",
				"student or course not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateReview",
			"failed to create review",
			err,
		)
	}

	return id, nil
}

func (r *reviewRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Review, error) {
	query := `
		SELECT
			rv.id,
			rv.student_id,
			rv.course_id,
			u.first_name || ' ' || u.last_name AS student_name,
			rv.rating,
			rv.comment,
			rv.created_at,
			rv.updated_at
		FROM reviews rv
		JOIN users u ON u.id = rv.student_id
		WHERE rv.id = $1;
	`

	var review models.Review

	err := scanReview(r.db.QueryRow(ctx, query, id), &review)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetReviewByID",
				"review not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetReviewByID",
			"failed to get review",
			err,
		)
	}

	return &review, nil
}

func (r *reviewRepo) GetListByCourseID(
	ctx context.Context,
	courseID uuid.UUID,
	filter models.ReviewFilter,
) ([]models.Review, int, error) {
	query := `
		SELECT
			rv.id,
			rv.student_id,
			rv.course_id,
			u.first_name || ' ' || u.last_name AS student_name,
			rv.rating,
			rv.comment,
			rv.created_at,
			rv.updated_at
		FROM reviews rv
		JOIN users u ON u.id = rv.student_id
		WHERE rv.course_id = $1
		ORDER BY rv.created_at DESC
		LIMIT $2 OFFSET $3;
	`

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	rows, err := r.db.Query(ctx, query, courseID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetReviewList",
			"failed to get reviews",
			err,
		)
	}
	defer rows.Close()

	reviews := make([]models.Review, 0)

	for rows.Next() {
		var review models.Review

		if err := scanReview(rows, &review); err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetReviewList",
				"failed to scan reviews",
				err,
			)
		}

		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetReviewList",
			"failed to read reviews",
			err,
		)
	}

	var total int

	err = r.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM reviews WHERE course_id = $1;`,
		courseID,
	).Scan(&total)

	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetReviewList",
			"failed to get total count",
			err,
		)
	}

	return reviews, total, nil
}

// GetAverageRating returns 0 for a course without reviews.
func (r *reviewRepo) GetAverageRating(
	ctx context.Context,
	courseID uuid.UUID,
) (float64, error) {
	query := `
		SELECT COALESCE(AVG(rating), 0)::float8
		FROM reviews
		WHERE course_id = $1;
	`

	var average float64

	err := r.db.QueryRow(
		ctx,
		query,
		courseID,
	).Scan(&average)

	if err != nil {
		return 0, apperror.Internal(
			"repository",
			"GetAverageRating",
			"failed to get average rating",
			err,
		)
	}

	return average, nil
}

func (r *reviewRepo) Update(
	ctx context.Context,
	review models.UpdateReview,
) (*models.Review, error) {
	query := `
		UPDATE reviews
		SET
			rating = COALESCE($2, rating),
			comment = COALESCE($3, comment),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		review.ID,
		review.Rating,
		review.Comment,
	)

	if err != nil {
		if pgerr.IsCheckViolation(err) {
			return nil, apperror.InvalidInput(
				"repository",
				"UpdateReview",
				"rating must be between 1 and 5",
				apperror.ErrInvalidInput,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateReview",
			"failed to update review",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return nil, apperror.NotFound(
			"repository",
			"UpdateReview",
			"review not found",
			apperror.ErrNotFound,
		)
	}

	return r.GetByID(ctx, review.ID)
}

func (r *reviewRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		DELETE FROM reviews
		WHERE id = $1;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		id,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteReview",
			"failed to delete review",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteReview",
			"review not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
