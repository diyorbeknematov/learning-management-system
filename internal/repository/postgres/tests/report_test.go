package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func courseReportFilter(courseID uuid.UUID) models.ReportFilter {
	return models.ReportFilter{CourseID: &courseID}
}

func setEnrollmentStatus(t *testing.T, tc *TestContext, id uuid.UUID, status models.EnrollmentStatus) {
	t.Helper()

	require.NoError(t, tc.Repo.Enrollment.UpdateStatus(context.Background(), models.UpdateEnrollmentStatus{
		ID:     id,
		Status: status,
	}))
}

func completeLesson(t *testing.T, tc *TestContext, studentID, lessonID uuid.UUID) {
	t.Helper()

	_, err := tc.Repo.Progress.Set(context.Background(), models.SetLessonProgress{
		StudentID: studentID,
		LessonID:  lessonID,
		Completed: true,
	})
	require.NoError(t, err)
}

func TestReportRepo_Enrollments(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	returning := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	otherCourseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	createTestEnrollment(t, tc, returning, otherCourseID)
	completed := createTestEnrollment(t, tc, first, courseID)
	dropped := createTestEnrollment(t, tc, second, courseID)
	createTestEnrollment(t, tc, returning, courseID)

	setEnrollmentStatus(t, tc, completed, models.EnrollmentStatusCompleted)
	setEnrollmentStatus(t, tc, dropped, models.EnrollmentStatusDropped)

	rows, err := tc.Repo.Report.Enrollments(ctx, courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, courseID, rows[0].CourseID)
	require.Equal(t, 3, rows[0].Total)
	require.Equal(t, 1, rows[0].Active)
	require.Equal(t, 1, rows[0].Completed)
	require.Equal(t, 1, rows[0].Dropped)
	require.Equal(t, 2, rows[0].NewStudents)
	require.Equal(t, 1, rows[0].ReturningStudents)
	require.Equal(t, 33.33, rows[0].DropoutRate)
}

func TestReportRepo_Enrollments_CourseWithoutEnrollments(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	rows, err := tc.Repo.Report.Enrollments(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Zero(t, rows[0].Total)
	require.Zero(t, rows[0].DropoutRate)
}

func TestReportRepo_EnrollmentTrend(t *testing.T) {
	tc := setupTest(t)

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestEnrollment(t, tc, first, courseID)
	createTestEnrollment(t, tc, second, courseID)

	filter := courseReportFilter(courseID)
	filter.GroupBy = "month"

	rows, err := tc.Repo.Report.EnrollmentTrend(context.Background(), filter)

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 2, rows[0].Count)
}

func TestReportRepo_Revenue(t *testing.T) {
	tc := setupTest(t)

	paid := createTestStudent(t, tc)
	free := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	paidEnrollment := createTestEnrollment(t, tc, paid, courseID)
	createTestEnrollment(t, tc, free, courseID)
	createTestPayment(t, tc, paidEnrollment, 40)

	rows, err := tc.Repo.Report.Revenue(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, rows[0].PaidEnrollments)
	require.Equal(t, 1, rows[0].FreeEnrollments)
	require.Equal(t, 40.0, rows[0].Revenue)
	require.Equal(t, 20.0, rows[0].AvgEnrollmentValue)
}

func TestReportRepo_Students(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	lessonID := createTestLesson(t, tc, moduleID, 1)
	enrollmentID := createTestEnrollment(t, tc, studentID, courseID)
	setEnrollmentStatus(t, tc, enrollmentID, models.EnrollmentStatusCompleted)

	find := func() models.StudentReportRow {
		rows, err := tc.Repo.Report.Students(ctx, models.ReportFilter{})
		require.NoError(t, err)

		for _, row := range rows {
			if row.StudentID == studentID {
				return row
			}
		}

		t.Fatal("student is missing in the report")

		return models.StudentReportRow{}
	}

	row := find()
	require.Equal(t, 1, row.EnrolledCourses)
	require.Equal(t, 1, row.CompletedCourses)
	require.False(t, row.Active)
	require.Nil(t, row.LastActivityAt)

	completeLesson(t, tc, studentID, lessonID)

	row = find()
	require.True(t, row.Active)
	require.NotNil(t, row.LastActivityAt)
}

func TestReportRepo_StudentGrowth(t *testing.T) {
	tc := setupTest(t)

	createTestStudent(t, tc)

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	rows, err := tc.Repo.Report.StudentGrowth(context.Background(), models.ReportFilter{
		DateRange: models.DateRange{From: &from, To: &to},
		GroupBy:   "month",
	})

	require.NoError(t, err)

	total := 0
	for _, row := range rows {
		total += row.Count
	}

	require.GreaterOrEqual(t, total, 1)
}

func TestReportRepo_Progress(t *testing.T) {
	tc := setupTest(t)

	active := createTestStudent(t, tc)
	idle := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	firstLesson := createTestLesson(t, tc, moduleID, 1)
	createTestLesson(t, tc, moduleID, 2)
	createTestEnrollment(t, tc, active, courseID)
	createTestEnrollment(t, tc, idle, courseID)

	completeLesson(t, tc, active, firstLesson)

	rows, err := tc.Repo.Report.Progress(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 2, rows[0].Students)
	require.Equal(t, 25.0, rows[0].AvgCompletionRate)
	require.Equal(t, 1, rows[0].InactiveStudents)
}

func TestReportRepo_ProgressFunnel(t *testing.T) {
	tc := setupTest(t)

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	firstLesson := createTestLesson(t, tc, moduleID, 1)
	secondLesson := createTestLesson(t, tc, moduleID, 2)

	completeLesson(t, tc, first, firstLesson)
	completeLesson(t, tc, first, secondLesson)
	completeLesson(t, tc, second, firstLesson)

	rows, err := tc.Repo.Report.ProgressFunnel(context.Background(), courseID)

	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, firstLesson, rows[0].LessonID)
	require.Equal(t, 2, rows[0].StudentsCompleted)
	require.Equal(t, secondLesson, rows[1].LessonID)
	require.Equal(t, 1, rows[1].StudentsCompleted)
}

func completeTestAttempt(t *testing.T, tc *TestContext, attemptID uuid.UUID, score int) {
	t.Helper()

	require.NoError(t, tc.Repo.Attempt.Complete(context.Background(), models.CompleteAttempt{
		ID:          attemptID,
		Score:       score,
		CompletedAt: time.Now(),
		TimeSpent:   60,
	}))
}

func TestReportRepo_Quizzes(t *testing.T) {
	tc := setupTest(t)

	passed := createTestStudent(t, tc)
	failed := createTestStudent(t, tc)
	quizID, courseID := createTestQuiz(t, tc)

	completeTestAttempt(t, tc, createTestAttempt(t, tc, passed, quizID, 1), 80)
	completeTestAttempt(t, tc, createTestAttempt(t, tc, failed, quizID, 1), 50)
	createTestAttempt(t, tc, failed, quizID, 2)

	rows, err := tc.Repo.Report.Quizzes(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, quizID, rows[0].QuizID)
	require.Equal(t, 2, rows[0].Attempts)
	require.Equal(t, 65.0, rows[0].AvgScore)
	require.Equal(t, 50.0, rows[0].PassRate)
	require.Equal(t, 50.0, rows[0].FailRate)
	require.Equal(t, 1.0, rows[0].AvgAttemptsPerUser)
}

func TestReportRepo_MostFailedQuestions(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	right := createTestStudent(t, tc)
	wrong := createTestStudent(t, tc)
	skipped := createTestStudent(t, tc)
	quizID, courseID := createTestQuiz(t, tc)
	questionID := createTestQuestion(t, tc, quizID, 1)
	createTestOption(t, tc, questionID, "Go", true)
	createTestOption(t, tc, questionID, "Java", false)

	options, err := tc.Repo.Question.GetOptions(ctx, questionID)
	require.NoError(t, err)

	optionID := func(text string) uuid.UUID {
		for _, option := range options {
			if option.OptionText == text {
				return option.ID
			}
		}

		t.Fatal("option is missing")

		return uuid.Nil
	}

	answer := func(studentID uuid.UUID, text string) {
		attemptID := createTestAttempt(t, tc, studentID, quizID, 1)

		if text != "" {
			require.NoError(t, tc.Repo.Attempt.CreateAnswer(ctx, models.AttemptAnswer{
				AttemptID:  attemptID,
				QuestionID: questionID,
				OptionID:   optionID(text),
			}))
		}

		completeTestAttempt(t, tc, attemptID, 0)
	}

	answer(right, "Go")
	answer(wrong, "Java")
	answer(skipped, "")

	rows, err := tc.Repo.Report.MostFailedQuestions(ctx, courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, questionID, rows[0].QuestionID)
	require.Equal(t, 2, rows[0].FailCount)
}

func TestReportRepo_Certificates(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestCertificate(t, tc, first, courseID)
	createTestCertificate(t, tc, second, courseID)

	rows, err := tc.Repo.Report.Certificates(ctx, courseReportFilter(courseID))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 2, rows[0].Issued)

	filter := courseReportFilter(courseID)
	filter.GroupBy = "month"

	trend, err := tc.Repo.Report.CertificateTrend(ctx, filter)
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, 2, trend[0].Count)
}

func TestReportRepo_Instructors(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	paid := createTestStudent(t, tc)
	free := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	course, err := tc.Repo.Course.GetByID(ctx, courseID)
	require.NoError(t, err)

	_, err = tc.DB.Pool.Exec(ctx, `
		UPDATE users
		SET role_id = (SELECT id FROM roles WHERE name = 'Instructor')
		WHERE id = $1`, course.InstructorID)
	require.NoError(t, err)

	paidEnrollment := createTestEnrollment(t, tc, paid, courseID)
	createTestEnrollment(t, tc, free, courseID)
	createTestPayment(t, tc, paidEnrollment, 40)
	createTestReview(t, tc, paid, courseID, 4)

	rows, err := tc.Repo.Report.Instructors(ctx, courseReportFilter(courseID))
	require.NoError(t, err)

	var found *models.InstructorReportRow
	for i := range rows {
		if rows[i].InstructorID == course.InstructorID {
			found = &rows[i]
		}
	}

	require.NotNil(t, found)
	require.Equal(t, 1, found.Courses)
	require.Equal(t, 2, found.Students)
	require.Equal(t, 40.0, found.Revenue)
	require.Equal(t, 4.0, found.AvgRating)
}

func TestReportRepo_Reviews(t *testing.T) {
	tc := setupTest(t)

	reviewed := createTestStudent(t, tc)
	silent := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	reviewedEnrollment := createTestEnrollment(t, tc, reviewed, courseID)
	silentEnrollment := createTestEnrollment(t, tc, silent, courseID)
	setEnrollmentStatus(t, tc, reviewedEnrollment, models.EnrollmentStatusCompleted)
	setEnrollmentStatus(t, tc, silentEnrollment, models.EnrollmentStatusCompleted)
	createTestReview(t, tc, reviewed, courseID, 2)

	rows, err := tc.Repo.Report.Reviews(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 2.0, rows[0].AvgRating)
	require.Equal(t, 1, rows[0].ReviewCount)
	require.Equal(t, 2, rows[0].CompletedCount)
	require.Equal(t, 1, rows[0].ReviewGap)
	require.True(t, rows[0].LowRating)
}

func TestReportRepo_Reviews_NoReviews(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	rows, err := tc.Repo.Report.Reviews(context.Background(), courseReportFilter(courseID))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Zero(t, rows[0].ReviewCount)
	require.False(t, rows[0].LowRating)
}
