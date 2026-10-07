package postgres

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Every report takes the same filter: $1 is the start and $2 the end of the
// period (both optional) and $3 is an optional course.

type reportRepo struct {
	db DBTX
}

func NewReportRepository(db DBTX) *reportRepo {
	return &reportRepo{
		db: db,
	}
}

// collect runs a report query and scans every row with scan.
func collect[T any](
	ctx context.Context,
	db DBTX,
	op string,
	query string,
	scan func(row pgx.Row, item *T) error,
	args ...any,
) ([]T, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to run report",
			err,
		)
	}
	defer rows.Close()

	items := make([]T, 0)

	for rows.Next() {
		var item T

		if err := scan(rows, &item); err != nil {
			return nil, apperror.Internal(
				"repository",
				op,
				"failed to scan report",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			op,
			"failed to read report",
			err,
		)
	}

	return items, nil
}

// scanTrend scans a period and its count.
func scanTrend(row pgx.Row, item *models.TrendRow) error {
	return row.Scan(&item.Period, &item.Count)
}

// Enrollments returns the enrollments of every course by status. A student is
// new when the enrollment is their first one on the platform, otherwise
// returning. The dropout rate is the share of dropped enrollments.
func (r *reportRepo) Enrollments(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.EnrollmentReportRow, error) {
	query := `
		SELECT
			c.id AS course_id,
			c.title AS course_title,
			COUNT(e.id) AS total,
			COUNT(e.id) FILTER (WHERE e.status = 'active') AS active,
			COUNT(e.id) FILTER (WHERE e.status = 'completed') AS completed,
			COUNT(e.id) FILTER (WHERE e.status = 'dropped') AS dropped,
			COUNT(e.id) FILTER (
				WHERE NOT EXISTS (
					SELECT 1
					FROM enrollments x
					WHERE x.student_id = e.student_id
						AND x.created_at < e.created_at
						AND x.deleted_at IS NULL
				)
			) AS new_students,
			COUNT(e.id) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM enrollments x
					WHERE x.student_id = e.student_id
						AND x.created_at < e.created_at
						AND x.deleted_at IS NULL
				)
			) AS returning_students,
			COALESCE(
				ROUND(
					COUNT(e.id) FILTER (WHERE e.status = 'dropped') * 100.0
						/ NULLIF(COUNT(e.id), 0),
					2
				),
				0
			)::float8 AS dropout_rate
		FROM courses c
		LEFT JOIN enrollments e
			ON e.course_id = c.id
			AND e.deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR e.created_at >= $1)
			AND ($2::timestamptz IS NULL OR e.created_at <= $2)
		WHERE c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		GROUP BY c.id, c.title
		ORDER BY total DESC, c.title;
	`

	return collect(
		ctx,
		r.db,
		"EnrollmentReport",
		query,
		func(row pgx.Row, item *models.EnrollmentReportRow) error {
			return row.Scan(
				&item.CourseID,
				&item.CourseTitle,
				&item.Total,
				&item.Active,
				&item.Completed,
				&item.Dropped,
				&item.NewStudents,
				&item.ReturningStudents,
				&item.DropoutRate,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// EnrollmentTrend returns the number of enrollments per day, week or month.
func (r *reportRepo) EnrollmentTrend(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.TrendRow, error) {
	query := `
		SELECT
			date_trunc($4, created_at) AS period,
			COUNT(*) AS count
		FROM enrollments
		WHERE deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR created_at >= $1)
			AND ($2::timestamptz IS NULL OR created_at <= $2)
			AND ($3::uuid IS NULL OR course_id = $3)
		GROUP BY 1
		ORDER BY 1;
	`

	return collect(
		ctx,
		r.db,
		"EnrollmentTrendReport",
		query,
		scanTrend,
		filter.From,
		filter.To,
		filter.CourseID,
		helpers.PeriodUnit(filter.GroupBy),
	)
}

// Revenue returns the paid and the free enrollments of every course and the
// money they brought. An enrollment without a payment is a free one, and the
// average enrollment value is the revenue divided by all enrollments.
func (r *reportRepo) Revenue(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.RevenueReportRow, error) {
	query := `
		SELECT
			c.id AS course_id,
			c.title AS course_title,
			COUNT(p.id) AS paid_enrollments,
			COUNT(e.id) FILTER (WHERE p.id IS NULL) AS free_enrollments,
			COALESCE(SUM(p.amount), 0)::float8 AS revenue,
			COALESCE(
				ROUND(SUM(p.amount) / NULLIF(COUNT(e.id), 0), 2),
				0
			)::float8 AS avg_enrollment_value
		FROM courses c
		LEFT JOIN enrollments e
			ON e.course_id = c.id
			AND e.deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR e.created_at >= $1)
			AND ($2::timestamptz IS NULL OR e.created_at <= $2)
		LEFT JOIN payments p ON p.enrollment_id = e.id
		WHERE c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		GROUP BY c.id, c.title
		ORDER BY revenue DESC, c.title;
	`

	return collect(
		ctx,
		r.db,
		"RevenueReport",
		query,
		func(row pgx.Row, item *models.RevenueReportRow) error {
			return row.Scan(
				&item.CourseID,
				&item.CourseTitle,
				&item.PaidEnrollments,
				&item.FreeEnrollments,
				&item.Revenue,
				&item.AvgEnrollmentValue,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// Students returns the students registered in the period, the best completers
// first. A student is active when they made progress in the last 7 days.
func (r *reportRepo) Students(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.StudentReportRow, error) {
	query := `
		SELECT
			u.id AS student_id,
			u.first_name || ' ' || u.last_name AS full_name,
			u.email,
			u.created_at AS registered_at,
			COUNT(e.id) AS enrolled_courses,
			COUNT(e.id) FILTER (WHERE e.status = 'completed') AS completed_courses,
			la.last_activity_at,
			COALESCE(la.last_activity_at >= NOW() - INTERVAL '7 days', FALSE) AS active
		FROM users u
		JOIN roles r ON r.id = u.role_id AND r.name = 'Student'
		LEFT JOIN enrollments e
			ON e.student_id = u.id
			AND e.deleted_at IS NULL
			AND ($3::uuid IS NULL OR e.course_id = $3)
		LEFT JOIN (
			SELECT
				student_id,
				MAX(updated_at) AS last_activity_at
			FROM lesson_progress
			GROUP BY student_id
		) la ON la.student_id = u.id
		WHERE u.deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR u.created_at >= $1)
			AND ($2::timestamptz IS NULL OR u.created_at <= $2)
		GROUP BY u.id, u.first_name, u.last_name, u.email, u.created_at, la.last_activity_at
		ORDER BY completed_courses DESC, enrolled_courses DESC, u.created_at;
	`

	return collect(
		ctx,
		r.db,
		"StudentReport",
		query,
		func(row pgx.Row, item *models.StudentReportRow) error {
			return row.Scan(
				&item.StudentID,
				&item.FullName,
				&item.Email,
				&item.RegisteredAt,
				&item.EnrolledCourses,
				&item.CompletedCourses,
				&item.LastActivityAt,
				&item.Active,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// StudentGrowth returns the number of registered students per day, week or
// month.
func (r *reportRepo) StudentGrowth(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.TrendRow, error) {
	query := `
		SELECT
			date_trunc($3, u.created_at) AS period,
			COUNT(*) AS count
		FROM users u
		JOIN roles r ON r.id = u.role_id AND r.name = 'Student'
		WHERE u.deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR u.created_at >= $1)
			AND ($2::timestamptz IS NULL OR u.created_at <= $2)
		GROUP BY 1
		ORDER BY 1;
	`

	return collect(
		ctx,
		r.db,
		"StudentGrowthReport",
		query,
		scanTrend,
		filter.From,
		filter.To,
		helpers.PeriodUnit(filter.GroupBy),
	)
}

// Progress returns the average completion rate of the students of every
// course (dropped enrollments excluded) and how many of the active students
// made no progress in the last 7 days.
func (r *reportRepo) Progress(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.ProgressReportRow, error) {
	query := `
		SELECT
			c.id AS course_id,
			c.title AS course_title,
			COUNT(e.id) AS students,
			COALESCE(
				ROUND(
					AVG(
						CASE
							WHEN COALESCE(tl.total_lessons, 0) = 0 THEN 0
							ELSE COALESCE(sp.completed_lessons, 0) * 100.0 / tl.total_lessons
						END
					),
					2
				),
				0
			)::float8 AS avg_completion_rate,
			COUNT(e.id) FILTER (
				WHERE e.status = 'active'
					AND (
						sp.last_activity_at IS NULL
						OR sp.last_activity_at < NOW() - INTERVAL '7 days'
					)
			) AS inactive_students
		FROM courses c
		LEFT JOIN enrollments e
			ON e.course_id = c.id
			AND e.deleted_at IS NULL
			AND e.status <> 'dropped'
			AND ($1::timestamptz IS NULL OR e.created_at >= $1)
			AND ($2::timestamptz IS NULL OR e.created_at <= $2)
		LEFT JOIN (
			SELECT
				m.course_id,
				COUNT(*) AS total_lessons
			FROM lessons l
			JOIN modules m ON m.id = l.module_id
			WHERE l.deleted_at IS NULL
				AND m.deleted_at IS NULL
			GROUP BY m.course_id
		) tl ON tl.course_id = c.id
		LEFT JOIN (
			SELECT
				m.course_id,
				p.student_id,
				COUNT(*) FILTER (WHERE p.completed) AS completed_lessons,
				MAX(p.updated_at) AS last_activity_at
			FROM lesson_progress p
			JOIN lessons l ON l.id = p.lesson_id
			JOIN modules m ON m.id = l.module_id
			WHERE l.deleted_at IS NULL
				AND m.deleted_at IS NULL
			GROUP BY m.course_id, p.student_id
		) sp ON sp.course_id = c.id AND sp.student_id = e.student_id
		WHERE c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		GROUP BY c.id, c.title
		ORDER BY c.title;
	`

	return collect(
		ctx,
		r.db,
		"ProgressReport",
		query,
		func(row pgx.Row, item *models.ProgressReportRow) error {
			return row.Scan(
				&item.CourseID,
				&item.CourseTitle,
				&item.Students,
				&item.AvgCompletionRate,
				&item.InactiveStudents,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// ProgressFunnel returns how many students completed each lesson of a course,
// in the order of the syllabus, to show where students drop off.
func (r *reportRepo) ProgressFunnel(
	ctx context.Context,
	courseID uuid.UUID,
) ([]models.ProgressFunnelRow, error) {
	query := `
		SELECT
			l.id AS lesson_id,
			l.title AS lesson_title,
			m.order_number AS module_order,
			l.order_number AS lesson_order,
			COUNT(p.id) FILTER (WHERE p.completed) AS students_completed
		FROM lessons l
		JOIN modules m ON m.id = l.module_id
		LEFT JOIN lesson_progress p ON p.lesson_id = l.id
		WHERE m.course_id = $1
			AND m.deleted_at IS NULL
			AND l.deleted_at IS NULL
		GROUP BY l.id, l.title, m.order_number, l.order_number
		ORDER BY m.order_number, l.order_number;
	`

	return collect(
		ctx,
		r.db,
		"ProgressFunnelReport",
		query,
		func(row pgx.Row, item *models.ProgressFunnelRow) error {
			return row.Scan(
				&item.LessonID,
				&item.LessonTitle,
				&item.ModuleOrder,
				&item.LessonOrder,
				&item.StudentsCompleted,
			)
		},
		courseID,
	)
}

// Quizzes returns the finished attempts of every quiz of the period. An
// attempt passes when its score reaches the pass threshold of the quiz.
func (r *reportRepo) Quizzes(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.QuizReportRow, error) {
	query := `
		SELECT
			q.id AS quiz_id,
			q.title AS quiz_title,
			COUNT(a.id) AS attempts,
			COALESCE(ROUND(AVG(a.score), 2), 0)::float8 AS avg_score,
			COALESCE(
				ROUND(
					COUNT(a.id) FILTER (WHERE a.score >= q.pass_threshold) * 100.0
						/ NULLIF(COUNT(a.id), 0),
					2
				),
				0
			)::float8 AS pass_rate,
			COALESCE(
				ROUND(
					COUNT(a.id) FILTER (WHERE a.score < q.pass_threshold) * 100.0
						/ NULLIF(COUNT(a.id), 0),
					2
				),
				0
			)::float8 AS fail_rate,
			COALESCE(
				ROUND(
					COUNT(a.id) * 1.0 / NULLIF(COUNT(DISTINCT a.student_id), 0),
					2
				),
				0
			)::float8 AS avg_attempts_per_user
		FROM quizzes q
		LEFT JOIN modules m ON m.id = q.module_id
		LEFT JOIN quiz_attempts a
			ON a.quiz_id = q.id
			AND a.completed_at IS NOT NULL
			AND ($1::timestamptz IS NULL OR a.started_at >= $1)
			AND ($2::timestamptz IS NULL OR a.started_at <= $2)
		WHERE q.deleted_at IS NULL
			AND ($3::uuid IS NULL OR q.course_id = $3 OR m.course_id = $3)
		GROUP BY q.id, q.title, q.pass_threshold
		ORDER BY q.title;
	`

	return collect(
		ctx,
		r.db,
		"QuizReport",
		query,
		func(row pgx.Row, item *models.QuizReportRow) error {
			return row.Scan(
				&item.QuizID,
				&item.QuizTitle,
				&item.Attempts,
				&item.AvgScore,
				&item.PassRate,
				&item.FailRate,
				&item.AvgAttemptsPerUser,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// MostFailedQuestions returns the 10 questions answered wrongly most often in
// finished attempts. A question is answered right only when the chosen
// options are exactly the correct ones; a skipped question counts as wrong.
func (r *reportRepo) MostFailedQuestions(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.MostFailedQuestionRow, error) {
	query := `
		WITH answered AS (
			SELECT
				aa.attempt_id,
				aa.question_id,
				COUNT(*) AS chosen,
				BOOL_AND(o.is_correct) AS all_chosen_correct
			FROM attempt_answers aa
			JOIN question_options o ON o.id = aa.option_id
			GROUP BY aa.attempt_id, aa.question_id
		),
		correct_options AS (
			SELECT
				question_id,
				COUNT(*) AS correct_total
			FROM question_options
			WHERE is_correct
				AND deleted_at IS NULL
			GROUP BY question_id
		)
		SELECT
			q.id AS question_id,
			qz.title AS quiz_title,
			q.text AS question_text,
			COUNT(*) FILTER (
				WHERE an.attempt_id IS NULL
					OR NOT (
						an.all_chosen_correct
						AND an.chosen = COALESCE(co.correct_total, 0)
					)
			) AS fail_count
		FROM quiz_attempts qa
		JOIN quizzes qz ON qz.id = qa.quiz_id AND qz.deleted_at IS NULL
		LEFT JOIN modules m ON m.id = qz.module_id
		JOIN questions q ON q.quiz_id = qa.quiz_id AND q.deleted_at IS NULL
		LEFT JOIN answered an ON an.attempt_id = qa.id AND an.question_id = q.id
		LEFT JOIN correct_options co ON co.question_id = q.id
		WHERE qa.completed_at IS NOT NULL
			AND ($1::timestamptz IS NULL OR qa.started_at >= $1)
			AND ($2::timestamptz IS NULL OR qa.started_at <= $2)
			AND ($3::uuid IS NULL OR qz.course_id = $3 OR m.course_id = $3)
		GROUP BY q.id, qz.title, q.text
		HAVING COUNT(*) FILTER (
			WHERE an.attempt_id IS NULL
				OR NOT (
					an.all_chosen_correct
					AND an.chosen = COALESCE(co.correct_total, 0)
				)
		) > 0
		ORDER BY fail_count DESC, q.text
		LIMIT 10;
	`

	return collect(
		ctx,
		r.db,
		"MostFailedQuestionsReport",
		query,
		func(row pgx.Row, item *models.MostFailedQuestionRow) error {
			return row.Scan(
				&item.QuestionID,
				&item.QuizTitle,
				&item.QuestionText,
				&item.FailCount,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// Certificates returns the number of certificates issued for every course.
func (r *reportRepo) Certificates(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.CertificateReportRow, error) {
	query := `
		SELECT
			c.id AS course_id,
			c.title AS course_title,
			COUNT(ct.id) AS issued
		FROM courses c
		LEFT JOIN certificates ct
			ON ct.course_id = c.id
			AND ($1::timestamptz IS NULL OR ct.created_at >= $1)
			AND ($2::timestamptz IS NULL OR ct.created_at <= $2)
		WHERE c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		GROUP BY c.id, c.title
		ORDER BY issued DESC, c.title;
	`

	return collect(
		ctx,
		r.db,
		"CertificateReport",
		query,
		func(row pgx.Row, item *models.CertificateReportRow) error {
			return row.Scan(
				&item.CourseID,
				&item.CourseTitle,
				&item.Issued,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// CertificateTrend returns the number of certificates issued per day, week or
// month.
func (r *reportRepo) CertificateTrend(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.TrendRow, error) {
	query := `
		SELECT
			date_trunc($4, created_at) AS period,
			COUNT(*) AS count
		FROM certificates
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
			AND ($2::timestamptz IS NULL OR created_at <= $2)
			AND ($3::uuid IS NULL OR course_id = $3)
		GROUP BY 1
		ORDER BY 1;
	`

	return collect(
		ctx,
		r.db,
		"CertificateTrendReport",
		query,
		scanTrend,
		filter.From,
		filter.To,
		filter.CourseID,
		helpers.PeriodUnit(filter.GroupBy),
	)
}

// Instructors returns, for every instructor, the number of courses, the
// distinct students reached, the revenue of their courses and the average
// rating of their reviews.
func (r *reportRepo) Instructors(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.InstructorReportRow, error) {
	query := `
		SELECT
			u.id AS instructor_id,
			u.first_name || ' ' || u.last_name AS full_name,
			COUNT(DISTINCT c.id) AS courses,
			COUNT(DISTINCT e.student_id) AS students,
			COALESCE(SUM(p.amount), 0)::float8 AS revenue,
			COALESCE(rt.avg_rating, 0)::float8 AS avg_rating
		FROM users u
		JOIN roles r ON r.id = u.role_id AND r.name = 'Instructor'
		LEFT JOIN courses c
			ON c.instructor_id = u.id
			AND c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		LEFT JOIN enrollments e
			ON e.course_id = c.id
			AND e.deleted_at IS NULL
			AND ($1::timestamptz IS NULL OR e.created_at >= $1)
			AND ($2::timestamptz IS NULL OR e.created_at <= $2)
		LEFT JOIN payments p ON p.enrollment_id = e.id
		LEFT JOIN (
			SELECT
				c2.instructor_id,
				ROUND(AVG(rv.rating), 2) AS avg_rating
			FROM reviews rv
			JOIN courses c2 ON c2.id = rv.course_id
			GROUP BY c2.instructor_id
		) rt ON rt.instructor_id = u.id
		WHERE u.deleted_at IS NULL
		GROUP BY u.id, u.first_name, u.last_name, rt.avg_rating
		ORDER BY revenue DESC, full_name;
	`

	return collect(
		ctx,
		r.db,
		"InstructorReport",
		query,
		func(row pgx.Row, item *models.InstructorReportRow) error {
			return row.Scan(
				&item.InstructorID,
				&item.FullName,
				&item.Courses,
				&item.Students,
				&item.Revenue,
				&item.AvgRating,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}

// Reviews returns the rating of every course, how many students completed it
// without leaving a review (review gap) and flags courses whose average is
// below 3 as low rating.
func (r *reportRepo) Reviews(
	ctx context.Context,
	filter models.ReportFilter,
) ([]models.ReviewReportRow, error) {
	query := `
		SELECT
			c.id AS course_id,
			c.title AS course_title,
			COALESCE(ROUND(AVG(rv.rating), 2), 0)::float8 AS avg_rating,
			COUNT(rv.id) AS review_count,
			COALESCE(ec.completed_count, 0) AS completed_count,
			COALESCE(rg.review_gap, 0) AS review_gap,
			COALESCE(AVG(rv.rating), 5) < 3 AS low_rating
		FROM courses c
		LEFT JOIN reviews rv
			ON rv.course_id = c.id
			AND ($1::timestamptz IS NULL OR rv.created_at >= $1)
			AND ($2::timestamptz IS NULL OR rv.created_at <= $2)
		LEFT JOIN (
			SELECT
				course_id,
				COUNT(*) AS completed_count
			FROM enrollments
			WHERE status = 'completed'
				AND deleted_at IS NULL
			GROUP BY course_id
		) ec ON ec.course_id = c.id
		LEFT JOIN (
			SELECT
				e.course_id,
				COUNT(*) AS review_gap
			FROM enrollments e
			WHERE e.status = 'completed'
				AND e.deleted_at IS NULL
				AND NOT EXISTS (
					SELECT 1
					FROM reviews x
					WHERE x.student_id = e.student_id
						AND x.course_id = e.course_id
				)
			GROUP BY e.course_id
		) rg ON rg.course_id = c.id
		WHERE c.deleted_at IS NULL
			AND ($3::uuid IS NULL OR c.id = $3)
		GROUP BY c.id, c.title, ec.completed_count, rg.review_gap
		ORDER BY avg_rating, c.title;
	`

	return collect(
		ctx,
		r.db,
		"ReviewReport",
		query,
		func(row pgx.Row, item *models.ReviewReportRow) error {
			return row.Scan(
				&item.CourseID,
				&item.CourseTitle,
				&item.AvgRating,
				&item.ReviewCount,
				&item.CompletedCount,
				&item.ReviewGap,
				&item.LowRating,
			)
		},
		filter.From,
		filter.To,
		filter.CourseID,
	)
}
