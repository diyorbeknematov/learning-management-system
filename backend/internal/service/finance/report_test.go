package finance_test

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func forCourse(courseID uuid.UUID) models.ReportFilter {
	return models.ReportFilter{CourseID: &courseID}
}

func TestReport_OnlyTheSuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	filter := models.ReportFilter{}

	for _, actor := range []models.Actor{testutil.ActorOf(e.NewUser(t, models.RoleInstructor)), testutil.ActorOf(e.NewUser(t, models.RoleStudent)), {}} {
		calls := map[string]func() error{
			"enrollments":  func() error { _, err := e.Svc.Report.Enrollments(ctx, actor, filter); return err },
			"revenue":      func() error { _, err := e.Svc.Report.Revenue(ctx, actor, filter); return err },
			"students":     func() error { _, err := e.Svc.Report.Students(ctx, actor, filter); return err },
			"progress":     func() error { _, err := e.Svc.Report.Progress(ctx, actor, filter); return err },
			"quizzes":      func() error { _, err := e.Svc.Report.Quizzes(ctx, actor, filter); return err },
			"certificates": func() error { _, err := e.Svc.Report.Certificates(ctx, actor, filter); return err },
			"instructors":  func() error { _, err := e.Svc.Report.Instructors(ctx, actor, filter); return err },
			"reviews":      func() error { _, err := e.Svc.Report.Reviews(ctx, actor, filter); return err },
		}

		for name, call := range calls {
			err := call()
			require.Error(t, err, name)
			testutil.RequireCode(t, err, apperror.CodeForbidden)
		}
	}
}

func TestReport_FromAfterToIsRejected(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Report.Revenue(context.Background(), testutil.AdminActor(), models.ReportFilter{DateRange: models.DateRange{From: &to, To: &from}})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestReport_Enrollments(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	courseID, _ := paidEnrollment(t, e)
	_, dropped := enrollmentIn(t, e, courseID)

	err := e.Repo.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{ID: dropped, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	filter := forCourse(courseID)
	filter.GroupBy = "month"

	report, err := e.Svc.Report.Enrollments(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Equal(t, 2, report.Rows[0].Total)
	require.Equal(t, 1, report.Rows[0].Dropped)
	require.Equal(t, 50.0, report.Rows[0].DropoutRate)
	require.Len(t, report.Trend, 1)
	require.Equal(t, 2, report.Trend[0].Count)
}

func TestReport_Revenue(t *testing.T) {
	e := testutil.Setup(t)

	courseID, paid := paidEnrollment(t, e)
	enrollmentIn(t, e, courseID)

	payAt(t, e, paid, 40, time.Now())

	rows, err := e.Svc.Report.Revenue(context.Background(), testutil.AdminActor(), forCourse(courseID))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, rows[0].PaidEnrollments)
	require.Equal(t, 1, rows[0].FreeEnrollments)
	require.Equal(t, 40.0, rows[0].Revenue)
	require.Equal(t, 20.0, rows[0].AvgEnrollmentValue)
}

func TestReport_Progress_FunnelOnlyForOneCourse(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	lessonID := e.AddLesson(t, course.ID)

	student, _ := enrollmentIn(t, e, course.ID)

	_, err := e.Repo.Progress.Set(ctx, models.SetLessonProgress{StudentID: student.ID, LessonID: lessonID, Completed: true})
	require.NoError(t, err)

	report, err := e.Svc.Report.Progress(ctx, testutil.AdminActor(), forCourse(course.ID))
	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Equal(t, 100.0, report.Rows[0].AvgCompletionRate)
	require.Len(t, report.Funnel, 1)
	require.Equal(t, 1, report.Funnel[0].StudentsCompleted)

	all, err := e.Svc.Report.Progress(ctx, testutil.AdminActor(), models.ReportFilter{})
	require.NoError(t, err)
	require.NotNil(t, all.Funnel, "no funnel without a course, but an empty list, not null")
	require.Empty(t, all.Funnel)
}

func TestReport_Certificates(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	student := e.NewUser(t, models.RoleStudent)

	_, err := e.Repo.Certificate.Create(ctx, models.CreateCertificate{
		StudentID: student.ID, CourseID: course.ID, CompletionDate: time.Now(), UniqueID: "LMS-" + uuid.NewString(),
	})
	require.NoError(t, err)

	filter := forCourse(course.ID)
	filter.GroupBy = "month"

	report, err := e.Svc.Report.Certificates(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Equal(t, 1, report.Rows[0].Issued)
	require.Len(t, report.Trend, 1)
	require.Equal(t, 1, report.Trend[0].Count)
}

func TestReport_Reviews(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	courseID, _ := paidEnrollment(t, e)
	student, _ := enrollmentIn(t, e, courseID)

	_, err := e.Repo.Review.Create(ctx, models.CreateReview{StudentID: student.ID, CourseID: courseID, Rating: 2})
	require.NoError(t, err)

	rows, err := e.Svc.Report.Reviews(ctx, testutil.AdminActor(), forCourse(courseID))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 2.0, rows[0].AvgRating)
	require.True(t, rows[0].LowRating)
}

func TestReport_StudentsInstructorsAndQuizzesRun(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	courseID, _ := paidEnrollment(t, e)
	filter := forCourse(courseID)
	filter.GroupBy = "week"

	students, err := e.Svc.Report.Students(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.NotNil(t, students.Rows)
	require.NotNil(t, students.Growth)

	instructors, err := e.Svc.Report.Instructors(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.NotNil(t, instructors)

	quizzes, err := e.Svc.Report.Quizzes(ctx, testutil.AdminActor(), filter)
	require.NoError(t, err)
	require.NotNil(t, quizzes.Rows)
	require.NotNil(t, quizzes.MostFailedQuestions)
}
