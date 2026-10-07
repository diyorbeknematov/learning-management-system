package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type enrollmentRepo struct {
	db DBTX
}

func NewEnrollmentRepository(db DBTX) *enrollmentRepo {
	return &enrollmentRepo{
		db: db,
	}
}

// scanEnrollment scans a row selected with the enrollment columns in the
// order used below.
func scanEnrollment(row pgx.Row, enrollment *models.Enrollment) error {
	return row.Scan(
		&enrollment.ID,
		&enrollment.CourseID,
		&enrollment.StudentID,
		&enrollment.Status,
		&enrollment.CreatedAt,
		&enrollment.UpdatedAt,
	)
}

func (r *enrollmentRepo) Create(
	ctx context.Context,
	enrollment models.CreateEnrollment,
) (uuid.UUID, error) {
	query := `
		INSERT INTO enrollments (
			id,
			course_id,
			student_id
		)
		VALUES ($1, $2, $3)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		enrollment.CourseID,
		enrollment.StudentID,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, handleError(err, "CreateEnrollment", "failed to create enrollment")
	}

	return id, nil
}

func (r *enrollmentRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Enrollment, error) {
	query := `
		SELECT
			id,
			course_id,
			student_id,
			status,
			created_at,
			updated_at
		FROM enrollments
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var enrollment models.Enrollment

	err := scanEnrollment(r.db.QueryRow(ctx, query, id), &enrollment)

	if err != nil {
		return nil, handleError(err, "GetEnrollmentByID", "failed to get enrollment")
	}

	return &enrollment, nil
}

// GetByStudentAndCourse is used to check whether a student is enrolled in a
// course.
func (r *enrollmentRepo) GetByStudentAndCourse(
	ctx context.Context,
	studentID uuid.UUID,
	courseID uuid.UUID,
) (*models.Enrollment, error) {
	query := `
		SELECT
			id,
			course_id,
			student_id,
			status,
			created_at,
			updated_at
		FROM enrollments
		WHERE student_id = $1
			AND course_id = $2
			AND deleted_at IS NULL;
	`

	var enrollment models.Enrollment

	err := scanEnrollment(r.db.QueryRow(ctx, query, studentID, courseID), &enrollment)

	if err != nil {
		return nil, handleError(err, "GetEnrollmentByStudentAndCourse", "failed to get enrollment")
	}

	return &enrollment, nil
}

// GetListByCourseID returns the roster of a course for its instructor.
func (r *enrollmentRepo) GetListByCourseID(
	ctx context.Context,
	courseID uuid.UUID,
	filter models.EnrollmentFilter,
) ([]models.EnrollmentRosterItem, int, error) {
	baseQuery := `
		SELECT
			e.id,
			e.course_id,
			e.student_id,
			e.status,
			e.created_at,
			e.updated_at,
			u.first_name || ' ' || u.last_name AS student_name,
			u.email
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		WHERE e.course_id = $1
			AND e.deleted_at IS NULL
	`

	countQuery := `
		SELECT COUNT(*)
		FROM enrollments e
		WHERE e.course_id = $1
			AND e.deleted_at IS NULL
	`

	conditions := []string{}
	args := []any{courseID}
	argIndex := 2

	if filter.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("e.status = $%d", argIndex),
		)

		args = append(args, *filter.Status)
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")

		baseQuery += whereClause
		countQuery += whereClause
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetEnrollmentList",
			"failed to get enrollments",
			err,
		)
	}
	defer rows.Close()

	enrollments := make([]models.EnrollmentRosterItem, 0)

	for rows.Next() {
		var item models.EnrollmentRosterItem

		err := rows.Scan(
			&item.ID,
			&item.CourseID,
			&item.StudentID,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.StudentName,
			&item.Email,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetEnrollmentList",
				"failed to scan enrollments",
				err,
			)
		}

		enrollments = append(enrollments, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetEnrollmentList",
			"failed to read enrollments",
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
			"GetEnrollmentList",
			"failed to get total count",
			err,
		)
	}

	return enrollments, total, nil
}

// GetMyList returns the enrollments of a student for the dashboard, each one
// with the course info and the lesson progress.
func (r *enrollmentRepo) GetMyList(
	ctx context.Context,
	studentID uuid.UUID,
	filter models.EnrollmentFilter,
) ([]models.MyEnrollment, int, error) {
	baseQuery := `
		SELECT
			e.id,
			e.course_id,
			e.student_id,
			e.status,
			e.created_at,
			e.updated_at,
			c.title AS course_title,
			c.cover AS course_cover,
			COALESCE(ls.total_lessons, 0) AS total_lessons,
			COALESCE(lp.completed_lessons, 0) AS completed_lessons,
			CASE
				WHEN COALESCE(ls.total_lessons, 0) = 0 THEN 0
				ELSE ROUND(COALESCE(lp.completed_lessons, 0) * 100.0 / ls.total_lessons, 2)
			END::float8 AS progress_percent
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		LEFT JOIN (
			SELECT
				m.course_id,
				COUNT(*) AS total_lessons
			FROM lessons l
			JOIN modules m ON m.id = l.module_id
			WHERE l.deleted_at IS NULL
				AND m.deleted_at IS NULL
			GROUP BY m.course_id
		) ls ON ls.course_id = e.course_id
		LEFT JOIN (
			SELECT
				m.course_id,
				COUNT(*) AS completed_lessons
			FROM lesson_progress p
			JOIN lessons l ON l.id = p.lesson_id
			JOIN modules m ON m.id = l.module_id
			WHERE p.student_id = $1
				AND p.completed
				AND l.deleted_at IS NULL
				AND m.deleted_at IS NULL
			GROUP BY m.course_id
		) lp ON lp.course_id = e.course_id
		WHERE e.student_id = $1
			AND e.deleted_at IS NULL
			AND c.deleted_at IS NULL
	`

	countQuery := `
		SELECT COUNT(*)
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1
			AND e.deleted_at IS NULL
			AND c.deleted_at IS NULL
	`

	conditions := []string{}
	args := []any{studentID}
	argIndex := 2

	if filter.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("e.status = $%d", argIndex),
		)

		args = append(args, *filter.Status)
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")

		baseQuery += whereClause
		countQuery += whereClause
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetMyEnrollmentList",
			"failed to get enrollments",
			err,
		)
	}
	defer rows.Close()

	enrollments := make([]models.MyEnrollment, 0)

	for rows.Next() {
		var item models.MyEnrollment

		err := rows.Scan(
			&item.ID,
			&item.CourseID,
			&item.StudentID,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CourseTitle,
			&item.CourseCover,
			&item.TotalLessons,
			&item.CompletedLessons,
			&item.ProgressPercent,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetMyEnrollmentList",
				"failed to scan enrollments",
				err,
			)
		}

		enrollments = append(enrollments, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetMyEnrollmentList",
			"failed to read enrollments",
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
			"GetMyEnrollmentList",
			"failed to get total count",
			err,
		)
	}

	return enrollments, total, nil
}

func (r *enrollmentRepo) UpdateStatus(
	ctx context.Context,
	enrollment models.UpdateEnrollmentStatus,
) error {
	query := `
		UPDATE enrollments
		SET
			status = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		enrollment.ID,
		enrollment.Status,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"UpdateEnrollmentStatus",
			"failed to update enrollment status",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"UpdateEnrollmentStatus",
			"enrollment not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
