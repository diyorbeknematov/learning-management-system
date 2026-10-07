package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type courseRepo struct {
	db DBTX
}

func NewCourseRepository(db DBTX) *courseRepo {
	return &courseRepo{
		db: db,
	}
}

// scanCourse scans a row selected with the course columns in the order used
// by GetByID and Update.
func scanCourse(row pgx.Row, course *models.Course) error {
	return row.Scan(
		&course.ID,
		&course.InstructorID,
		&course.CategoryID,
		&course.Title,
		&course.Cover,
		&course.Description,
		&course.Difficulty,
		&course.TotalDuration,
		&course.Language,
		&course.Status,
		&course.Price,
		&course.PayoutType,
		&course.PayoutValue,
		&course.CreatedAt,
		&course.UpdatedAt,
	)
}

func (r *courseRepo) Create(
	ctx context.Context,
	course models.CreateCourse,
) (uuid.UUID, error) {
	query := `
		INSERT INTO courses (
			id,
			instructor_id,
			category_id,
			title,
			cover,
			description,
			difficulty,
			total_duration,
			language,
			price,
			payout_type,
			payout_value
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12
		)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		course.InstructorID,
		course.CategoryID,
		course.Title,
		course.Cover,
		course.Description,
		course.Difficulty,
		course.TotalDuration,
		course.Language,
		course.Price,
		course.PayoutType,
		course.PayoutValue,
	).Scan(&id)

	if err != nil {
		if pgerr.IsForeignKeyViolation(err) || pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreateCourse",
				"invalid category, instructor or payout",
				apperror.ErrInvalidInput,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateCourse",
			"failed to create course",
			err,
		)
	}

	return id, nil
}

func (r *courseRepo) CreateLearningOutcome(
	ctx context.Context,
	outcome models.CourseLearningOutcome,
) error {
	query := `
		INSERT INTO course_learning_outcomes (
			id,
			course_id,
			content,
			position
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		uuid.New(),
		outcome.CourseID,
		outcome.Content,
		outcome.Position,
	)

	if err != nil {
		if pgerr.IsForeignKeyViolation(err) {
			return apperror.NotFound(
				"repository",
				"CreateLearningOutcome",
				"course not found",
				apperror.ErrNotFound,
			)
		}

		return apperror.Internal(
			"repository",
			"CreateLearningOutcome",
			"failed to create learning outcome",
			err,
		)
	}

	return nil
}

func (r *courseRepo) CreateRequirement(
	ctx context.Context,
	requirement models.CourseRequirement,
) error {
	query := `
		INSERT INTO course_requirements (
			id,
			course_id,
			content,
			position
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		uuid.New(),
		requirement.CourseID,
		requirement.Content,
		requirement.Position,
	)

	if err != nil {
		if pgerr.IsForeignKeyViolation(err) {
			return apperror.NotFound(
				"repository",
				"CreateRequirement",
				"course not found",
				apperror.ErrNotFound,
			)
		}

		return apperror.Internal(
			"repository",
			"CreateRequirement",
			"failed to create requirement",
			err,
		)
	}

	return nil
}

func (r *courseRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Course, error) {
	query := `
		SELECT
			id,
			instructor_id,
			category_id,
			title,
			cover,
			description,
			difficulty,
			total_duration,
			language,
			status,
			price,
			payout_type,
			payout_value,
			created_at,
			updated_at
		FROM courses
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var course models.Course

	err := scanCourse(r.db.QueryRow(ctx, query, id), &course)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetCourseByID",
				"course not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetCourseByID",
			"failed to get course",
			err,
		)
	}

	return &course, nil
}

func (r *courseRepo) GetLearningOutcomes(
	ctx context.Context,
	courseID uuid.UUID,
) ([]models.CourseLearningOutcome, error) {
	query := `
		SELECT
			id,
			course_id,
			content,
			position,
			created_at
		FROM course_learning_outcomes
		WHERE course_id = $1
		ORDER BY position;
	`

	rows, err := r.db.Query(ctx, query, courseID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetLearningOutcomes",
			"failed to get learning outcomes",
			err,
		)
	}
	defer rows.Close()

	outcomes := make([]models.CourseLearningOutcome, 0)

	for rows.Next() {
		var outcome models.CourseLearningOutcome

		err := rows.Scan(
			&outcome.ID,
			&outcome.CourseID,
			&outcome.Content,
			&outcome.Position,
			&outcome.CreatedAt,
		)
		if err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetLearningOutcomes",
				"failed to scan learning outcomes",
				err,
			)
		}

		outcomes = append(outcomes, outcome)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetLearningOutcomes",
			"failed to read learning outcomes",
			err,
		)
	}

	return outcomes, nil
}

func (r *courseRepo) GetRequirements(
	ctx context.Context,
	courseID uuid.UUID,
) ([]models.CourseRequirement, error) {
	query := `
		SELECT
			id,
			course_id,
			content,
			position,
			created_at
		FROM course_requirements
		WHERE course_id = $1
		ORDER BY position;
	`

	rows, err := r.db.Query(ctx, query, courseID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetRequirements",
			"failed to get requirements",
			err,
		)
	}
	defer rows.Close()

	requirements := make([]models.CourseRequirement, 0)

	for rows.Next() {
		var requirement models.CourseRequirement

		err := rows.Scan(
			&requirement.ID,
			&requirement.CourseID,
			&requirement.Content,
			&requirement.Position,
			&requirement.CreatedAt,
		)
		if err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetRequirements",
				"failed to scan requirements",
				err,
			)
		}

		requirements = append(requirements, requirement)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetRequirements",
			"failed to read requirements",
			err,
		)
	}

	return requirements, nil
}

var courseSortOrders = map[string]string{
	"popular": "enrollment_count DESC, c.created_at DESC",
	"rating":  "avg_rating DESC, review_count DESC, c.created_at DESC",
	"newest":  "c.created_at DESC",
	"price":   "c.price ASC, c.created_at DESC",
}

func (r *courseRepo) GetList(
	ctx context.Context,
	filter models.CourseFilter,
) ([]models.CourseListItem, int, error) {
	baseQuery := `
		SELECT
			c.id,
			c.instructor_id,
			c.category_id,
			c.title,
			c.cover,
			c.description,
			c.difficulty,
			c.total_duration,
			c.language,
			c.status,
			c.price,
			c.payout_type,
			c.payout_value,
			c.created_at,
			c.updated_at,
			cat.name AS category_name,
			u.first_name || ' ' || u.last_name AS instructor_name,
			COALESCE(rv.avg_rating, 0) AS avg_rating,
			COALESCE(rv.review_count, 0) AS review_count,
			COALESCE(ls.lesson_count, 0) AS lesson_count,
			COALESCE(en.enrollment_count, 0) AS enrollment_count
		FROM courses c
		JOIN categories cat ON cat.id = c.category_id
		JOIN users u ON u.id = c.instructor_id
		LEFT JOIN (
			SELECT
				course_id,
				AVG(rating)::float8 AS avg_rating,
				COUNT(*) AS review_count
			FROM reviews
			GROUP BY course_id
		) rv ON rv.course_id = c.id
		LEFT JOIN (
			SELECT
				m.course_id,
				COUNT(*) AS lesson_count
			FROM lessons l
			JOIN modules m ON m.id = l.module_id
			WHERE l.deleted_at IS NULL
				AND m.deleted_at IS NULL
			GROUP BY m.course_id
		) ls ON ls.course_id = c.id
		LEFT JOIN (
			SELECT
				course_id,
				COUNT(*) AS enrollment_count
			FROM enrollments
			WHERE deleted_at IS NULL
			GROUP BY course_id
		) en ON en.course_id = c.id
		WHERE c.deleted_at IS NULL
	`

	countQuery := `
		SELECT COUNT(*)
		FROM courses c
		LEFT JOIN (
			SELECT
				course_id,
				AVG(rating)::float8 AS avg_rating
			FROM reviews
			GROUP BY course_id
		) rv ON rv.course_id = c.id
		WHERE c.deleted_at IS NULL
	`

	conditions := []string{}
	args := []any{}
	argIndex := 1

	if filter.Search != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.title ILIKE $%d", argIndex),
		)

		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if filter.CategoryID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.category_id = $%d", argIndex),
		)

		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if filter.InstructorID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.instructor_id = $%d", argIndex),
		)

		args = append(args, *filter.InstructorID)
		argIndex++
	}

	if filter.Difficulty != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.difficulty = $%d", argIndex),
		)

		args = append(args, *filter.Difficulty)
		argIndex++
	}

	if filter.Language != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.language = $%d", argIndex),
		)

		args = append(args, *filter.Language)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("c.status = $%d", argIndex),
		)

		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.MinRating != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("COALESCE(rv.avg_rating, 0) >= $%d", argIndex),
		)

		args = append(args, *filter.MinRating)
		argIndex++
	}

	if filter.PriceType != nil {
		switch *filter.PriceType {
		case "free":
			conditions = append(conditions, "c.price = 0")
		case "paid":
			conditions = append(conditions, "c.price > 0")
		}
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")

		baseQuery += whereClause
		countQuery += whereClause
	}

	orderBy, ok := courseSortOrders[filter.Sort]
	if !ok {
		orderBy = courseSortOrders["newest"]
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY %s LIMIT $%d OFFSET $%d",
		orderBy,
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCourseList",
			"failed to get courses",
			err,
		)
	}
	defer rows.Close()

	courses := make([]models.CourseListItem, 0)

	for rows.Next() {
		var course models.CourseListItem

		err := rows.Scan(
			&course.ID,
			&course.InstructorID,
			&course.CategoryID,
			&course.Title,
			&course.Cover,
			&course.Description,
			&course.Difficulty,
			&course.TotalDuration,
			&course.Language,
			&course.Status,
			&course.Price,
			&course.PayoutType,
			&course.PayoutValue,
			&course.CreatedAt,
			&course.UpdatedAt,
			&course.CategoryName,
			&course.InstructorName,
			&course.AvgRating,
			&course.ReviewCount,
			&course.LessonCount,
			&course.EnrollmentCount,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetCourseList",
				"failed to scan courses",
				err,
			)
		}

		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCourseList",
			"failed to read courses",
			err,
		)
	}

	var total int

	err = r.db.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCourseList",
			"failed to get total count",
			err,
		)
	}

	return courses, total, nil
}

func (r *courseRepo) Update(
	ctx context.Context,
	course models.UpdateCourse,
) (*models.Course, error) {
	query := `
		UPDATE courses
		SET
			category_id = COALESCE($2, category_id),
			title = COALESCE($3, title),
			cover = COALESCE($4, cover),
			description = COALESCE($5, description),
			difficulty = COALESCE($6, difficulty),
			total_duration = COALESCE($7, total_duration),
			language = COALESCE($8, language),
			price = COALESCE($9, price),
			payout_type = COALESCE($10, payout_type),
			payout_value = COALESCE($11, payout_value),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			instructor_id,
			category_id,
			title,
			cover,
			description,
			difficulty,
			total_duration,
			language,
			status,
			price,
			payout_type,
			payout_value,
			created_at,
			updated_at;
	`

	var updatedCourse models.Course

	err := scanCourse(
		r.db.QueryRow(
			ctx,
			query,
			course.ID,
			course.CategoryID,
			course.Title,
			course.Cover,
			course.Description,
			course.Difficulty,
			course.TotalDuration,
			course.Language,
			course.Price,
			course.PayoutType,
			course.PayoutValue,
		),
		&updatedCourse,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateCourse",
				"course not found",
				apperror.ErrNotFound,
			)
		}

		if pgerr.IsForeignKeyViolation(err) || pgerr.IsCheckViolation(err) {
			return nil, apperror.InvalidInput(
				"repository",
				"UpdateCourse",
				"invalid category or payout",
				apperror.ErrInvalidInput,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateCourse",
			"failed to update course",
			err,
		)
	}

	return &updatedCourse, nil
}

func (r *courseRepo) UpdateStatus(
	ctx context.Context,
	status models.UpdateCourseStatus,
) error {
	query := `
		UPDATE courses
		SET
			status = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		status.ID,
		status.Status,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"UpdateCourseStatus",
			"failed to update course status",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"UpdateCourseStatus",
			"course not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *courseRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE courses
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
			"DeleteCourse",
			"failed to delete course",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteCourse",
			"course not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *courseRepo) DeleteLearningOutcomes(
	ctx context.Context,
	courseID uuid.UUID,
) error {
	query := `
		DELETE FROM course_learning_outcomes
		WHERE course_id = $1;
	`

	_, err := r.db.Exec(
		ctx,
		query,
		courseID,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteLearningOutcomes",
			"failed to delete learning outcomes",
			err,
		)
	}

	return nil
}

func (r *courseRepo) DeleteRequirements(
	ctx context.Context,
	courseID uuid.UUID,
) error {
	query := `
		DELETE FROM course_requirements
		WHERE course_id = $1;
	`

	_, err := r.db.Exec(
		ctx,
		query,
		courseID,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteRequirements",
			"failed to delete requirements",
			err,
		)
	}

	return nil
}
