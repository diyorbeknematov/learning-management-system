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

// payment makes a paid enrollment of a known student.
func payment(t *testing.T, e *testutil.Env) (student *models.User, courseID, enrollmentID uuid.UUID, paymentID uuid.UUID) {
	t.Helper()

	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	student = e.NewUser(t, models.RoleStudent)

	enrollmentID, err := e.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{CourseID: course.ID, StudentID: student.ID})
	require.NoError(t, err)

	paymentID, err = e.Repo.Payment.Create(context.Background(), models.CreatePayment{EnrollmentID: enrollmentID, Amount: 40, PaidAt: time.Now()})
	require.NoError(t, err)

	return student, course.ID, enrollmentID, paymentID
}

func TestPayment_GetList(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	_, courseID, _, paymentID := payment(t, e)

	list, err := e.Svc.Payment.GetList(ctx, testutil.AdminActor(), models.PaymentFilter{CourseID: &courseID})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, paymentID, list.Items[0].ID)
	require.Equal(t, 1, list.Page)

	other := uuid.New()

	list, err = e.Svc.Payment.GetList(ctx, testutil.AdminActor(), models.PaymentFilter{CourseID: &other})
	require.NoError(t, err)
	require.Zero(t, list.Total)
}

func TestPayment_GetList_OnlyTheSuperAdmin(t *testing.T) {
	e := testutil.Setup(t)

	student, _, _, _ := payment(t, e)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(e.NewUser(t, models.RoleInstructor)), {}} {
		_, err := e.Svc.Payment.GetList(context.Background(), actor, models.PaymentFilter{})
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestPayment_GetList_FromAfterTo(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Payment.GetList(context.Background(), testutil.AdminActor(), models.PaymentFilter{DateRange: models.DateRange{From: &to, To: &from}})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestPayment_GetByID(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	student, _, _, paymentID := payment(t, e)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.AdminActor()} {
		found, err := e.Svc.Payment.GetByID(ctx, actor, paymentID)
		require.NoError(t, err)
		require.Equal(t, 40.0, found.Amount)
	}

	for _, other := range []models.Actor{testutil.ActorOf(e.NewUser(t, models.RoleStudent)), {}} {
		_, err := e.Svc.Payment.GetByID(ctx, other, paymentID)
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}

	_, err := e.Svc.Payment.GetByID(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestPayment_GetByID_TheInstructorHasNoAccessToTheMoney(t *testing.T) {
	e := testutil.Setup(t)

	_, courseID, _, paymentID := payment(t, e)

	course, err := e.Repo.Course.GetByID(context.Background(), courseID)
	require.NoError(t, err)

	owner := models.Actor{UserID: course.InstructorID, RoleName: models.RoleInstructor}

	_, err = e.Svc.Payment.GetByID(context.Background(), owner, paymentID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestPayment_GetByEnrollment(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	student, _, enrollmentID, paymentID := payment(t, e)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.AdminActor()} {
		found, err := e.Svc.Payment.GetByEnrollment(ctx, actor, enrollmentID)
		require.NoError(t, err)
		require.Equal(t, paymentID, found.ID)
	}

	_, err := e.Svc.Payment.GetByEnrollment(ctx, testutil.ActorOf(e.NewUser(t, models.RoleStudent)), enrollmentID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = e.Svc.Payment.GetByEnrollment(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestPayment_GetByEnrollment_FreeCourseHasNoPayment(t *testing.T) {
	e := testutil.Setup(t)

	instructor := e.NewUser(t, models.RoleInstructor)
	course := e.NewCourse(t, instructor, e.NewCategory(t))
	student := e.NewUser(t, models.RoleStudent)

	enrollmentID, err := e.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{CourseID: course.ID, StudentID: student.ID})
	require.NoError(t, err)

	_, err = e.Svc.Payment.GetByEnrollment(context.Background(), testutil.ActorOf(student), enrollmentID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}
