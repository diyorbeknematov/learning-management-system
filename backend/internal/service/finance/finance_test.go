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

// The money tests use dates far in the past, so the real data of the database
// cannot change their totals. Test packages run at the same time on one
// database, so each package has a year of its own: the repository tests use
// 2001, these use 2002, and the API tests only read years nobody writes.
var (
	firstDay  = time.Date(2002, 5, 10, 12, 0, 0, 0, time.UTC)
	secondDay = time.Date(2002, 5, 11, 12, 0, 0, 0, time.UTC)
	from      = time.Date(2002, 5, 1, 0, 0, 0, 0, time.UTC)
	to        = time.Date(2002, 5, 31, 23, 59, 59, 0, time.UTC)
)

func window(groupBy string) models.FinanceFilter {
	return models.FinanceFilter{DateRange: models.DateRange{From: &from, To: &to}, GroupBy: groupBy}
}

// paidEnrollment makes an enrollment of a new student in a new course.
func paidEnrollment(t *testing.T, e *testutil.Env) (courseID, enrollmentID uuid.UUID) {
	t.Helper()

	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	student := e.NewUser(t, models.RoleStudent)

	id, err := e.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{CourseID: course.ID, StudentID: student.ID})
	require.NoError(t, err)

	return course.ID, id
}

func payAt(t *testing.T, e *testutil.Env, enrollmentID uuid.UUID, amount float64, at time.Time) {
	t.Helper()

	_, err := e.Repo.Payment.Create(context.Background(), models.CreatePayment{EnrollmentID: enrollmentID, Amount: amount, PaidAt: at})
	require.NoError(t, err)
}

func payoutAt(t *testing.T, e *testutil.Env, courseID, enrollmentID uuid.UUID, amount float64, at time.Time) {
	t.Helper()

	ctx := context.Background()

	course, err := e.Repo.Course.GetByID(ctx, courseID)
	require.NoError(t, err)

	id, err := e.Repo.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     courseID,
		EnrollmentID: enrollmentID,
		Type:         models.PayoutTypeFixed,
		Value:        amount,
		Amount:       amount,
	})
	require.NoError(t, err)

	e.Exec(t, `UPDATE instructor_payouts SET created_at = $2 WHERE id = $1`, id, at)
}

// seed makes two paid enrollments: 30 on the first day and 20 on the second,
// with payouts of 9 and 6.
func seed(t *testing.T, e *testutil.Env) uuid.UUID {
	t.Helper()

	courseID, first := paidEnrollment(t, e)
	_, second := enrollmentIn(t, e, courseID)

	payAt(t, e, first, 30, firstDay)
	payAt(t, e, second, 20, secondDay)
	payoutAt(t, e, courseID, first, 9, firstDay)
	payoutAt(t, e, courseID, second, 6, secondDay)

	return courseID
}

// enrollmentIn adds another student to the course.
func enrollmentIn(t *testing.T, e *testutil.Env, courseID uuid.UUID) (*models.User, uuid.UUID) {
	t.Helper()

	student := e.NewUser(t, models.RoleStudent)

	id, err := e.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{CourseID: courseID, StudentID: student.ID})
	require.NoError(t, err)

	return student, id
}

func TestFinance_GetSummary(t *testing.T) {
	e := testutil.Setup(t)

	seed(t, e)

	summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), window("day"))
	require.NoError(t, err)

	require.Equal(t, 50.0, summary.Revenue)
	require.Equal(t, 15.0, summary.Expenses)
	require.Equal(t, 35.0, summary.NetProfit)
	require.Equal(t, "profit", summary.Status)
	require.Len(t, summary.Points, 2)
	require.Equal(t, 21.0, summary.Points[0].Profit)
	require.Equal(t, 14.0, summary.Points[1].Profit)
}

func TestFinance_GetSummary_GroupBy(t *testing.T) {
	e := testutil.Setup(t)

	seed(t, e)

	for _, groupBy := range []string{"week", "month"} {
		summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), window(groupBy))
		require.NoError(t, err)
		require.Len(t, summary.Points, 1, groupBy)
		require.Equal(t, 50.0, summary.Points[0].Revenue)
	}

	summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), window(""))
	require.NoError(t, err)
	require.Len(t, summary.Points, 2, "the default is by day")
}

func TestFinance_GetSummary_Loss(t *testing.T) {
	e := testutil.Setup(t)

	courseID, enrollmentID := paidEnrollment(t, e)

	payAt(t, e, enrollmentID, 10, firstDay)
	payoutAt(t, e, courseID, enrollmentID, 25, secondDay)

	summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), window("day"))
	require.NoError(t, err)
	require.Equal(t, -15.0, summary.NetProfit)
	require.Equal(t, "loss", summary.Status)
}

func TestFinance_GetSummary_NothingEarnedIsNotALoss(t *testing.T) {
	e := testutil.Setup(t)

	empty := models.DateRange{
		From: ptr(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
		To:   ptr(time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC)),
	}

	summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), models.FinanceFilter{DateRange: empty})
	require.NoError(t, err)
	require.Zero(t, summary.NetProfit)
	require.Equal(t, "profit", summary.Status)
	require.Empty(t, summary.Points)
}

func ptr[T any](value T) *T {
	return &value
}

func TestFinance_GetSummary_CentsAreRounded(t *testing.T) {
	e := testutil.Setup(t)

	_, first := paidEnrollment(t, e)
	_, second := paidEnrollment(t, e)

	payAt(t, e, first, 0.1, firstDay)
	payAt(t, e, second, 0.2, firstDay)

	summary, err := e.Svc.Finance.GetSummary(context.Background(), testutil.AdminActor(), window("day"))
	require.NoError(t, err)
	require.Equal(t, 0.3, summary.Revenue, "0.1 + 0.2 must not become 0.30000000000000004")
	require.Equal(t, 0.3, summary.Points[0].Revenue)
}

func TestFinance_DefaultPeriodIsTheLast30Days(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	_, enrollmentID := paidEnrollment(t, e)

	payAt(t, e, enrollmentID, 12.34, time.Now().Add(-time.Hour))

	summary, err := e.Svc.Finance.GetSummary(ctx, testutil.AdminActor(), models.FinanceFilter{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, summary.Revenue, 12.34, "a payment of the last hour is in the default period")

	revenue, err := e.Svc.Finance.GetRevenue(ctx, testutil.AdminActor(), models.FinanceFilter{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, revenue.Total, 12.34)

	// a payment from 2002 is far outside it
	_, old := paidEnrollment(t, e)
	payAt(t, e, old, 7777777, firstDay)

	after, err := e.Svc.Finance.GetRevenue(ctx, testutil.AdminActor(), models.FinanceFilter{})
	require.NoError(t, err)
	require.Less(t, after.Total, 7777777.0)
}

func TestFinance_GetRevenue(t *testing.T) {
	e := testutil.Setup(t)

	seed(t, e)

	revenue, err := e.Svc.Finance.GetRevenue(context.Background(), testutil.AdminActor(), window("day"))
	require.NoError(t, err)
	require.Equal(t, 50.0, revenue.Total)
	require.Len(t, revenue.Points, 2)
	require.Equal(t, 30.0, revenue.Points[0].Revenue)
	require.Equal(t, 20.0, revenue.Points[1].Revenue)
}

func TestFinance_GetExpenses(t *testing.T) {
	e := testutil.Setup(t)

	seed(t, e)

	expenses, err := e.Svc.Finance.GetExpenses(context.Background(), testutil.AdminActor(), window("day"))
	require.NoError(t, err)
	require.Equal(t, 15.0, expenses.Total)
	require.Len(t, expenses.Points, 2)
	require.Equal(t, 9.0, expenses.Points[0].Expenses)

	require.Len(t, expenses.Payouts, 2)
	require.Equal(t, 6.0, expenses.Payouts[0].Amount, "the newest payout comes first")
	require.Equal(t, 9.0, expenses.Payouts[1].Amount)
}

func TestFinance_OnlyTheSuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()

	actors := []models.Actor{
		testutil.ActorOf(e.NewUser(t, models.RoleInstructor)),
		testutil.ActorOf(e.NewUser(t, models.RoleStudent)),
		{},
	}

	for _, actor := range actors {
		_, err := e.Svc.Finance.GetSummary(ctx, actor, window("day"))
		testutil.RequireCode(t, err, apperror.CodeForbidden)

		_, err = e.Svc.Finance.GetRevenue(ctx, actor, window("day"))
		testutil.RequireCode(t, err, apperror.CodeForbidden)

		_, err = e.Svc.Finance.GetExpenses(ctx, actor, window("day"))
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestFinance_FromAfterToIsRejected(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	backwards := models.FinanceFilter{DateRange: models.DateRange{From: &to, To: &from}}

	_, err := e.Svc.Finance.GetSummary(ctx, testutil.AdminActor(), backwards)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	_, err = e.Svc.Finance.GetRevenue(ctx, testutil.AdminActor(), backwards)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	_, err = e.Svc.Finance.GetExpenses(ctx, testutil.AdminActor(), backwards)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}
