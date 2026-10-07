package tests

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestPaidEnrollment creates a student, a course and an enrollment, in
// the order their cleanups need.
func createTestPaidEnrollment(t *testing.T, tc *TestContext) (studentID, courseID, enrollmentID uuid.UUID) {
	t.Helper()

	studentID = createTestStudent(t, tc)
	courseID = createTestCourse(t, tc, "course_"+uuid.NewString())
	enrollmentID = createTestEnrollment(t, tc, studentID, courseID)

	return studentID, courseID, enrollmentID
}

func createTestPayment(t *testing.T, tc *TestContext, enrollmentID uuid.UUID, amount float64) uuid.UUID {
	t.Helper()

	return createTestPaymentAt(t, tc, enrollmentID, amount, time.Now())
}

func createTestPaymentAt(t *testing.T, tc *TestContext, enrollmentID uuid.UUID, amount float64, paidAt time.Time) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	id, err := tc.Repo.Payment.Create(ctx, models.CreatePayment{
		EnrollmentID: enrollmentID,
		Amount:       amount,
		PaidAt:       paidAt,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM payments WHERE id = $1`, id)
	})

	return id
}

func TestPaymentRepo_Create(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)

	id := createTestPayment(t, tc, enrollmentID, 49.5)
	require.NotEqual(t, uuid.Nil, id)
}

func TestPaymentRepo_Create_AlreadyPaid(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)
	createTestPayment(t, tc, enrollmentID, 49.5)

	_, err := tc.Repo.Payment.Create(context.Background(), models.CreatePayment{
		EnrollmentID: enrollmentID,
		Amount:       49.5,
		PaidAt:       time.Now(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestPaymentRepo_Create_NegativeAmount(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)

	_, err := tc.Repo.Payment.Create(context.Background(), models.CreatePayment{
		EnrollmentID: enrollmentID,
		Amount:       -1,
		PaidAt:       time.Now(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestPaymentRepo_Create_EnrollmentNotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Payment.Create(context.Background(), models.CreatePayment{
		EnrollmentID: uuid.New(),
		Amount:       10,
		PaidAt:       time.Now(),
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestPaymentRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)
	id := createTestPayment(t, tc, enrollmentID, 49.5)

	payment, err := tc.Repo.Payment.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, payment.ID)
	require.Equal(t, enrollmentID, payment.EnrollmentID)
	require.Equal(t, 49.5, payment.Amount)
	require.Equal(t, models.PaymentStatusPaid, payment.Status)
	require.NotNil(t, payment.PaidAt)
}

func TestPaymentRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Payment.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestPaymentRepo_GetByEnrollmentID(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)
	id := createTestPayment(t, tc, enrollmentID, 49.5)

	payment, err := tc.Repo.Payment.GetByEnrollmentID(context.Background(), enrollmentID)

	require.NoError(t, err)
	require.Equal(t, id, payment.ID)
}

func TestPaymentRepo_GetByEnrollmentID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, _, enrollmentID := createTestPaidEnrollment(t, tc)

	_, err := tc.Repo.Payment.GetByEnrollmentID(context.Background(), enrollmentID)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestPaymentRepo_GetList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	_, courseID, enrollmentID := createTestPaidEnrollment(t, tc)
	id := createTestPayment(t, tc, enrollmentID, 49.5)

	payments, total, err := tc.Repo.Payment.GetList(ctx, models.PaymentFilter{CourseID: &courseID})

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, id, payments[0].ID)

	otherCourse := uuid.New()
	_, total, err = tc.Repo.Payment.GetList(ctx, models.PaymentFilter{CourseID: &otherCourse})
	require.NoError(t, err)
	require.Zero(t, total)
}

func TestPaymentRepo_GetList_DateRange(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	_, courseID, enrollmentID := createTestPaidEnrollment(t, tc)
	createTestPayment(t, tc, enrollmentID, 49.5)

	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	_, total, err := tc.Repo.Payment.GetList(ctx, models.PaymentFilter{
		DateRange: models.DateRange{From: &from, To: &to},
		CourseID:  &courseID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)

	future := time.Now().Add(time.Hour)

	_, total, err = tc.Repo.Payment.GetList(ctx, models.PaymentFilter{
		DateRange: models.DateRange{From: &future},
		CourseID:  &courseID,
	})
	require.NoError(t, err)
	require.Zero(t, total)
}

func TestPaymentRepo_CreatePayout(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	_, courseID, enrollmentID := createTestPaidEnrollment(t, tc)

	course, err := tc.Repo.Course.GetByID(ctx, courseID)
	require.NoError(t, err)

	id, err := tc.Repo.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     courseID,
		EnrollmentID: enrollmentID,
		Type:         models.PayoutTypePercentage,
		Value:        30,
		Amount:       14.85,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM instructor_payouts WHERE id = $1`, id)
	})

	_, err = tc.Repo.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     courseID,
		EnrollmentID: enrollmentID,
		Type:         models.PayoutTypeFixed,
		Value:        5,
		Amount:       5,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestPaymentRepo_CreatePayout_InvalidPercentage(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	_, courseID, enrollmentID := createTestPaidEnrollment(t, tc)

	course, err := tc.Repo.Course.GetByID(ctx, courseID)
	require.NoError(t, err)

	_, err = tc.Repo.Payment.CreatePayout(ctx, models.CreateInstructorPayout{
		InstructorID: course.InstructorID,
		CourseID:     courseID,
		EnrollmentID: enrollmentID,
		Type:         models.PayoutTypePercentage,
		Value:        150,
		Amount:       10,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}
