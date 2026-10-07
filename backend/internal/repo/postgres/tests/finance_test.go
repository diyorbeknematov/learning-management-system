package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The finance tests use dates far in the past so the amounts of other data in
// the database cannot change the totals.
var (
	// Test packages run at the same time on one database, so each has a year
	// of its own: these tests use 2001 (see also service/finance, which uses 2002).
	financeFirstDay  = time.Date(2001, 5, 10, 12, 0, 0, 0, time.UTC)
	financeSecondDay = time.Date(2001, 5, 11, 12, 0, 0, 0, time.UTC)
	financeFrom      = time.Date(2001, 5, 1, 0, 0, 0, 0, time.UTC)
	financeTo        = time.Date(2001, 5, 31, 23, 59, 59, 0, time.UTC)
)

func createTestPayoutAt(t *testing.T, tc *TestContext, courseID, enrollmentID uuid.UUID, amount float64, createdAt time.Time) {
	t.Helper()

	ctx := context.Background()

	course, err := tc.Repo.Course.GetByID(ctx, courseID)
	require.NoError(t, err)

	id, err := tc.Repo.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     courseID,
		EnrollmentID: enrollmentID,
		Type:         models.PayoutTypeFixed,
		Value:        amount,
		Amount:       amount,
	})
	require.NoError(t, err)

	_, err = tc.DB.Pool.Exec(ctx, `UPDATE instructor_payouts SET created_at = $2 WHERE id = $1`, id, createdAt)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM instructor_payouts WHERE id = $1`, id)
	})
}

// setupFinance creates a course with two paid enrollments: 30 paid on the
// first day and 20 on the second, with payouts of 9 and 6.
func setupFinance(t *testing.T, tc *TestContext) uuid.UUID {
	t.Helper()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	firstEnrollment := createTestEnrollment(t, tc, first, courseID)
	secondEnrollment := createTestEnrollment(t, tc, second, courseID)

	createTestPaymentAt(t, tc, firstEnrollment, 30, financeFirstDay)
	createTestPaymentAt(t, tc, secondEnrollment, 20, financeSecondDay)
	createTestPayoutAt(t, tc, courseID, firstEnrollment, 9, financeFirstDay)
	createTestPayoutAt(t, tc, courseID, secondEnrollment, 6, financeSecondDay)

	return courseID
}

func financeFilter(groupBy string) models.FinanceFilter {
	return models.FinanceFilter{
		DateRange: models.DateRange{From: &financeFrom, To: &financeTo},
		GroupBy:   groupBy,
	}
}

func TestFinanceRepo_GetSummary(t *testing.T) {
	tc := setupTest(t)

	setupFinance(t, tc)

	summary, err := tc.Repo.Finance.GetSummary(context.Background(), financeFilter("day"))

	require.NoError(t, err)
	require.Equal(t, 50.0, summary.Revenue)
	require.Equal(t, 15.0, summary.Expenses)
	require.Equal(t, 35.0, summary.NetProfit)
	require.Len(t, summary.Points, 2)
	require.Equal(t, 30.0, summary.Points[0].Revenue)
	require.Equal(t, 9.0, summary.Points[0].Expenses)
	require.Equal(t, 21.0, summary.Points[0].Profit)
	require.Equal(t, 14.0, summary.Points[1].Profit)
}

func TestFinanceRepo_GetSummary_GroupByMonth(t *testing.T) {
	tc := setupTest(t)

	setupFinance(t, tc)

	summary, err := tc.Repo.Finance.GetSummary(context.Background(), financeFilter("month"))

	require.NoError(t, err)
	require.Len(t, summary.Points, 1)
	require.Equal(t, 50.0, summary.Points[0].Revenue)
	require.Equal(t, 15.0, summary.Points[0].Expenses)
}

func TestFinanceRepo_GetSummary_Loss(t *testing.T) {
	tc := setupTest(t)

	student := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	enrollmentID := createTestEnrollment(t, tc, student, courseID)

	createTestPaymentAt(t, tc, enrollmentID, 10, financeFirstDay)
	createTestPayoutAt(t, tc, courseID, enrollmentID, 25, financeSecondDay)

	summary, err := tc.Repo.Finance.GetSummary(context.Background(), financeFilter("day"))

	require.NoError(t, err)
	require.Equal(t, -15.0, summary.NetProfit)
	require.Len(t, summary.Points, 2)
	require.Equal(t, -25.0, summary.Points[1].Profit)
}

func TestFinanceRepo_GetSummary_EmptyPeriod(t *testing.T) {
	tc := setupTest(t)

	from := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC)

	summary, err := tc.Repo.Finance.GetSummary(context.Background(), models.FinanceFilter{
		DateRange: models.DateRange{From: &from, To: &to},
	})

	require.NoError(t, err)
	require.Zero(t, summary.Revenue)
	require.Zero(t, summary.NetProfit)
	require.Empty(t, summary.Points)
}

func TestFinanceRepo_GetRevenue(t *testing.T) {
	tc := setupTest(t)

	setupFinance(t, tc)

	revenue, err := tc.Repo.Finance.GetRevenue(context.Background(), financeFilter("day"))

	require.NoError(t, err)
	require.Equal(t, 50.0, revenue.Total)
	require.Len(t, revenue.Points, 2)
	require.Equal(t, 30.0, revenue.Points[0].Revenue)
	require.Equal(t, 20.0, revenue.Points[1].Revenue)
}

func TestFinanceRepo_GetExpenses(t *testing.T) {
	tc := setupTest(t)

	setupFinance(t, tc)

	expenses, err := tc.Repo.Finance.GetExpenses(context.Background(), financeFilter("day"))

	require.NoError(t, err)
	require.Equal(t, 15.0, expenses.Total)
	require.Len(t, expenses.Points, 2)
	require.Equal(t, 9.0, expenses.Points[0].Expenses)
	require.Equal(t, 6.0, expenses.Points[1].Expenses)
}

func TestFinanceRepo_GetPayouts(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	courseID := setupFinance(t, tc)

	payouts, total, err := tc.Repo.Finance.GetPayouts(ctx, models.PaymentFilter{
		DateRange: models.DateRange{From: &financeFrom, To: &financeTo},
		CourseID:  &courseID,
	})

	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, payouts, 2)
	require.Equal(t, 6.0, payouts[0].Amount)
	require.Equal(t, 9.0, payouts[1].Amount)

	payouts, total, err = tc.Repo.Finance.GetPayouts(ctx, models.PaymentFilter{
		CourseID: &courseID,
		Limit:    1,
		Page:     2,
	})

	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, payouts, 1)
}
