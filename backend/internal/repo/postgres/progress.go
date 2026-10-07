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

type progressRepo struct {
	db DBTX
}

func NewProgressRepository(db DBTX) *progressRepo {
	return &progressRepo{
		db: db,
	}
}

// scanProgress scans a row selected with the lesson progress columns in the
// order used below.
func scanProgress(row pgx.Row, progress *models.LessonProgress) error {
	return row.Scan(
		&progress.ID,
		&progress.StudentID,
		&progress.LessonID,
		&progress.Completed,
		&progress.CompletedAt,
		&progress.CreatedAt,
		&progress.UpdatedAt,
	)
}

// Set creates the progress of a lesson or changes it if it already exists.
// completed_at keeps the time of the first completion and is cleared when the
// lesson is marked as not completed.
func (r *progressRepo) Set(
	ctx context.Context,
	progress models.SetLessonProgress,
) (*models.LessonProgress, error) {
	query := `
		INSERT INTO lesson_progress (
			id,
			student_id,
			lesson_id,
			completed,
			completed_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4::boolean,
			CASE WHEN $4::boolean THEN CURRENT_TIMESTAMP ELSE NULL END
		)
		ON CONFLICT (student_id, lesson_id) DO UPDATE
		SET
			completed = EXCLUDED.completed,
			completed_at = CASE
				WHEN EXCLUDED.completed
					THEN COALESCE(lesson_progress.completed_at, CURRENT_TIMESTAMP)
				ELSE NULL
			END,
			updated_at = CURRENT_TIMESTAMP
		RETURNING
			id,
			student_id,
			lesson_id,
			completed,
			completed_at,
			created_at,
			updated_at;
	`

	var result models.LessonProgress

	err := scanProgress(
		r.db.QueryRow(
			ctx,
			query,
			uuid.New(),
			progress.StudentID,
			progress.LessonID,
			progress.Completed,
		),
		&result,
	)

	if err != nil {
		return nil, handleError(err, "SetLessonProgress", "failed to set lesson progress")
	}

	return &result, nil
}

func (r *progressRepo) GetByLesson(
	ctx context.Context,
	studentID uuid.UUID,
	lessonID uuid.UUID,
) (*models.LessonProgress, error) {
	query := `
		SELECT
			id,
			student_id,
			lesson_id,
			completed,
			completed_at,
			created_at,
			updated_at
		FROM lesson_progress
		WHERE student_id = $1
			AND lesson_id = $2;
	`

	var progress models.LessonProgress

	err := scanProgress(r.db.QueryRow(ctx, query, studentID, lessonID), &progress)

	if err != nil {
		return nil, handleError(err, "GetLessonProgress", "failed to get lesson progress")
	}

	return &progress, nil
}

// GetCourseProgress counts the lessons of a course and how many of them the
// student completed. A student without any progress gets zero completed.
func (r *progressRepo) GetCourseProgress(
	ctx context.Context,
	studentID uuid.UUID,
	courseID uuid.UUID,
) (*models.CourseProgress, error) {
	query := `
		SELECT
			COUNT(l.id) AS total_lessons,
			COUNT(p.id) FILTER (WHERE p.completed) AS completed_lessons,
			CASE
				WHEN COUNT(l.id) = 0 THEN 0
				ELSE ROUND(COUNT(p.id) FILTER (WHERE p.completed) * 100.0 / COUNT(l.id), 2)
			END::float8 AS progress_percent
		FROM lessons l
		JOIN modules m ON m.id = l.module_id
		LEFT JOIN lesson_progress p
			ON p.lesson_id = l.id
			AND p.student_id = $2
		WHERE m.course_id = $1
			AND m.deleted_at IS NULL
			AND l.deleted_at IS NULL;
	`

	progress := models.CourseProgress{
		CourseID: courseID,
	}

	err := r.db.QueryRow(
		ctx,
		query,
		courseID,
		studentID,
	).Scan(
		&progress.TotalLessons,
		&progress.CompletedLessons,
		&progress.ProgressPercent,
	)

	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetCourseProgress",
			"failed to get course progress",
			err,
		)
	}

	return &progress, nil
}

// GetCompletedLessonIDs returns the lessons of a course the student completed,
// so the client can mark them on the syllabus.
func (r *progressRepo) GetCompletedLessonIDs(
	ctx context.Context,
	studentID uuid.UUID,
	courseID uuid.UUID,
) ([]uuid.UUID, error) {
	query := `
		SELECT p.lesson_id
		FROM lesson_progress p
		JOIN lessons l ON l.id = p.lesson_id
		JOIN modules m ON m.id = l.module_id
		WHERE p.student_id = $1
			AND p.completed
			AND m.course_id = $2
			AND m.deleted_at IS NULL
			AND l.deleted_at IS NULL
		ORDER BY p.completed_at;
	`

	rows, err := r.db.Query(ctx, query, studentID, courseID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetCompletedLessonIDs",
			"failed to get completed lessons",
			err,
		)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)

	for rows.Next() {
		var id uuid.UUID

		if err := rows.Scan(&id); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetCompletedLessonIDs",
				"failed to scan completed lessons",
				err,
			)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetCompletedLessonIDs",
			"failed to read completed lessons",
			err,
		)
	}

	return ids, nil
}

// GetStudentList returns the enrolled students of a course with their progress
// for the instructor.
func (r *progressRepo) GetStudentList(
	ctx context.Context,
	courseID uuid.UUID,
	filter models.EnrollmentFilter,
) ([]models.StudentProgress, int, error) {
	baseQuery := `
		SELECT
			e.student_id,
			u.first_name || ' ' || u.last_name AS full_name,
			u.email,
			e.course_id,
			tl.total_lessons,
			COALESCE(sp.completed_lessons, 0) AS completed_lessons,
			CASE
				WHEN tl.total_lessons = 0 THEN 0
				ELSE ROUND(COALESCE(sp.completed_lessons, 0) * 100.0 / tl.total_lessons, 2)
			END::float8 AS progress_percent,
			sp.last_activity_at
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		CROSS JOIN (
			SELECT COUNT(*) AS total_lessons
			FROM lessons l
			JOIN modules m ON m.id = l.module_id
			WHERE m.course_id = $1
				AND m.deleted_at IS NULL
				AND l.deleted_at IS NULL
		) tl
		LEFT JOIN (
			SELECT
				p.student_id,
				COUNT(*) FILTER (WHERE p.completed) AS completed_lessons,
				MAX(p.updated_at) AS last_activity_at
			FROM lesson_progress p
			JOIN lessons l ON l.id = p.lesson_id
			JOIN modules m ON m.id = l.module_id
			WHERE m.course_id = $1
				AND m.deleted_at IS NULL
				AND l.deleted_at IS NULL
			GROUP BY p.student_id
		) sp ON sp.student_id = e.student_id
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
		" ORDER BY u.first_name, u.last_name LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetStudentProgressList",
			"failed to get students progress",
			err,
		)
	}
	defer rows.Close()

	students := make([]models.StudentProgress, 0)

	for rows.Next() {
		var student models.StudentProgress

		err := rows.Scan(
			&student.StudentID,
			&student.FullName,
			&student.Email,
			&student.CourseID,
			&student.TotalLessons,
			&student.CompletedLessons,
			&student.ProgressPercent,
			&student.LastActivityAt,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetStudentProgressList",
				"failed to scan students progress",
				err,
			)
		}

		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetStudentProgressList",
			"failed to read students progress",
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
			"GetStudentProgressList",
			"failed to get total count",
			err,
		)
	}

	return students, total, nil
}
