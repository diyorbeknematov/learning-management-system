package learning_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// shop is an instructor with a published course that has lessons.
type shop struct {
	e       *testutil.Env
	owner   *models.User
	course  *models.CourseDetail
	module  *models.Module
	lessons []uuid.UUID
}

// newShop creates a published course with the given price and number of
// lessons. A payout is set when payoutType is not nil (only a SuperAdmin can).
func newShop(t *testing.T, price float64, lessons int, payoutType *models.PayoutType, payoutValue float64) shop {
	t.Helper()

	ctx := context.Background()
	e := testutil.Setup(t)
	owner := e.NewUser(t, models.RoleInstructor)
	category := e.NewCategory(t)

	req := models.CreateCourseRequest{InstructorID: &owner.ID, CategoryID: category.ID, Title: "Course " + testutil.Uniq(), Price: price}
	if payoutType != nil {
		req.PayoutType = payoutType
		req.PayoutValue = &payoutValue
	}

	course, err := e.Svc.Course.Create(ctx, testutil.AdminActor(), req)
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	module, err := e.Svc.Module.Create(ctx, testutil.ActorOf(owner), course.ID, models.CreateModuleRequest{Title: "Module"})
	require.NoError(t, err)

	ids := make([]uuid.UUID, lessons)

	for i := range ids {
		lesson, err := e.Svc.Lesson.Create(ctx, testutil.ActorOf(owner), module.ID, models.CreateLessonRequest{Title: "Lesson"})
		require.NoError(t, err)

		ids[i] = lesson.ID
	}

	err = e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(owner), models.UpdateCourseStatus{ID: course.ID, Status: models.CourseStatusPublished})
	require.NoError(t, err)

	return shop{e: e, owner: owner, course: course, module: module, lessons: ids}
}

func (s shop) student(t *testing.T) *models.User {
	t.Helper()

	return s.e.NewUser(t, models.RoleStudent)
}

func (s shop) enroll(t *testing.T, student *models.User) *models.Enrollment {
	t.Helper()

	enrollment, err := s.e.Svc.Enrollment.Enroll(context.Background(), testutil.ActorOf(student), s.course.ID)
	require.NoError(t, err)

	return enrollment
}

func (s shop) payouts(t *testing.T) []models.InstructorPayout {
	t.Helper()

	payouts, _, err := s.e.Repo.Finance.GetPayouts(context.Background(), models.PaymentFilter{CourseID: &s.course.ID})
	require.NoError(t, err)

	return payouts
}

func payoutType(t models.PayoutType) *models.PayoutType {
	return &t
}

func TestEnrollment_Enroll_FreeCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.student(t)
	enrollment := s.enroll(t, student)

	require.Equal(t, student.ID, enrollment.StudentID)
	require.Equal(t, s.course.ID, enrollment.CourseID)
	require.Equal(t, models.EnrollmentStatusActive, enrollment.Status)

	_, err := s.e.Repo.Payment.GetByEnrollmentID(context.Background(), enrollment.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestEnrollment_Enroll_PaidCourse(t *testing.T) {
	s := newShop(t, 49.5, 1, nil, 0)

	enrollment := s.enroll(t, s.student(t))

	payment, err := s.e.Repo.Payment.GetByEnrollmentID(context.Background(), enrollment.ID)
	require.NoError(t, err)
	require.Equal(t, 49.5, payment.Amount)
	require.Equal(t, models.PaymentStatusPaid, payment.Status)
	require.NotNil(t, payment.PaidAt)

	require.Empty(t, s.payouts(t), "no payout is set on the course, so there is no instructor share")
}

func TestEnrollment_Enroll_PercentagePayout(t *testing.T) {
	s := newShop(t, 99.99, 1, payoutType(models.PayoutTypePercentage), 30)

	enrollment := s.enroll(t, s.student(t))

	payouts := s.payouts(t)
	require.Len(t, payouts, 1)
	require.Equal(t, enrollment.ID, payouts[0].EnrollmentID)
	require.Equal(t, s.owner.ID, payouts[0].InstructorID)
	require.Equal(t, models.PayoutTypePercentage, payouts[0].Type)
	require.Equal(t, 30.0, payouts[0].Value)
	require.Equal(t, 30.0, payouts[0].Amount, "30%% of 99.99 rounded to cents")
}

func TestEnrollment_Enroll_FixedPayout(t *testing.T) {
	s := newShop(t, 100, 1, payoutType(models.PayoutTypeFixed), 25)

	s.enroll(t, s.student(t))

	payouts := s.payouts(t)
	require.Len(t, payouts, 1)
	require.Equal(t, 25.0, payouts[0].Amount)
}

func TestEnrollment_Enroll_FixedPayoutIsNeverMoreThanThePrice(t *testing.T) {
	s := newShop(t, 100, 1, payoutType(models.PayoutTypeFixed), 500)

	s.enroll(t, s.student(t))

	payouts := s.payouts(t)
	require.Len(t, payouts, 1)
	require.Equal(t, 100.0, payouts[0].Amount)
	require.Equal(t, 500.0, payouts[0].Value, "the configured value is kept as it was")
}

func TestEnrollment_Enroll_FreeCourseHasNoPayout(t *testing.T) {
	s := newShop(t, 0, 1, payoutType(models.PayoutTypePercentage), 30)

	s.enroll(t, s.student(t))

	require.Empty(t, s.payouts(t))
}

func TestEnrollment_Enroll_PaymentKeepsThePriceOfThatDay(t *testing.T) {
	s := newShop(t, 20, 1, nil, 0)

	ctx := context.Background()
	enrollment := s.enroll(t, s.student(t))

	price := 80.0

	_, err := s.e.Svc.Course.Update(ctx, testutil.ActorOf(s.owner), s.course.ID, models.UpdateCourseRequest{Price: &price})
	require.NoError(t, err)

	payment, err := s.e.Repo.Payment.GetByEnrollmentID(ctx, enrollment.ID)
	require.NoError(t, err)
	require.Equal(t, 20.0, payment.Amount)
}

func TestEnrollment_Enroll_Twice(t *testing.T) {
	s := newShop(t, 20, 1, nil, 0)

	student := s.student(t)
	s.enroll(t, student)

	_, err := s.e.Svc.Enrollment.Enroll(context.Background(), testutil.ActorOf(student), s.course.ID)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestEnrollment_Enroll_DraftCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()

	err := s.e.Svc.Course.UpdateStatus(ctx, testutil.ActorOf(s.owner), models.UpdateCourseStatus{ID: s.course.ID, Status: models.CourseStatusDraft})
	require.NoError(t, err)

	_, err = s.e.Svc.Enrollment.Enroll(ctx, testutil.ActorOf(s.student(t)), s.course.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestEnrollment_Enroll_UnknownCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Enrollment.Enroll(context.Background(), testutil.ActorOf(s.student(t)), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestEnrollment_Enroll_OwnCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Enrollment.Enroll(context.Background(), testutil.ActorOf(s.owner), s.course.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestEnrollment_Enroll_AfterDroppingComesBackWithoutPayingAgain(t *testing.T) {
	s := newShop(t, 20, 1, payoutType(models.PayoutTypeFixed), 5)

	ctx := context.Background()
	student := s.student(t)
	first := s.enroll(t, student)

	err := s.e.Svc.Enrollment.UpdateStatus(ctx, testutil.ActorOf(student), models.UpdateEnrollmentStatus{ID: first.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	again, err := s.e.Svc.Enrollment.Enroll(ctx, testutil.ActorOf(student), s.course.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID)
	require.Equal(t, models.EnrollmentStatusActive, again.Status)

	payments, total, err := s.e.Repo.Payment.GetList(ctx, models.PaymentFilter{CourseID: &s.course.ID})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, payments, 1)
	require.Len(t, s.payouts(t), 1)
}

func TestEnrollment_GetByID(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	enrollment := s.enroll(t, student)

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(s.owner), testutil.AdminActor()} {
		found, err := s.e.Svc.Enrollment.GetByID(ctx, actor, enrollment.ID)
		require.NoError(t, err)
		require.Equal(t, enrollment.ID, found.ID)
	}
}

func TestEnrollment_GetByID_OtherPeopleCannotSee(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	enrollment := s.enroll(t, s.student(t))

	for _, other := range []*models.User{s.student(t), s.e.NewUser(t, models.RoleInstructor)} {
		_, err := s.e.Svc.Enrollment.GetByID(ctx, testutil.ActorOf(other), enrollment.ID)
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestEnrollment_GetByID_NotFound(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Enrollment.GetByID(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestEnrollment_GetListByCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	active, leaving := s.student(t), s.student(t)
	s.enroll(t, active)
	dropped := s.enroll(t, leaving)

	err := s.e.Svc.Enrollment.UpdateStatus(ctx, testutil.ActorOf(leaving), models.UpdateEnrollmentStatus{ID: dropped.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	list, err := s.e.Svc.Enrollment.GetListByCourse(ctx, testutil.ActorOf(s.owner), s.course.ID, models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)
	require.NotEmpty(t, list.Items[0].StudentName)

	status := models.EnrollmentStatusDropped

	list, err = s.e.Svc.Enrollment.GetListByCourse(ctx, testutil.AdminActor(), s.course.ID, models.EnrollmentFilter{Status: &status})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, dropped.ID, list.Items[0].ID)
}

func TestEnrollment_GetListByCourse_OnlyTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()

	for _, other := range []*models.User{s.student(t), s.e.NewUser(t, models.RoleInstructor)} {
		_, err := s.e.Svc.Enrollment.GetListByCourse(ctx, testutil.ActorOf(other), s.course.ID, models.EnrollmentFilter{})
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestEnrollment_GetMyList(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	enrollment := s.enroll(t, student)

	_, err := s.e.Svc.Progress.SetLessonProgress(ctx, testutil.ActorOf(student), s.lessons[0], true)
	require.NoError(t, err)

	list, err := s.e.Svc.Enrollment.GetMyList(ctx, testutil.ActorOf(student), models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, enrollment.ID, list.Items[0].ID)
	require.Equal(t, 2, list.Items[0].TotalLessons)
	require.Equal(t, 1, list.Items[0].CompletedLessons)
	require.Equal(t, 50.0, list.Items[0].ProgressPercent)

	other, err := s.e.Svc.Enrollment.GetMyList(ctx, testutil.ActorOf(s.student(t)), models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Zero(t, other.Total)
}

func TestEnrollment_UpdateStatus_StudentLeaves(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	enrollment := s.enroll(t, student)

	err := s.e.Svc.Enrollment.UpdateStatus(ctx, testutil.ActorOf(student), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	found, err := s.e.Svc.Enrollment.GetByID(ctx, testutil.ActorOf(student), enrollment.ID)
	require.NoError(t, err)
	require.Equal(t, models.EnrollmentStatusDropped, found.Status)
}

func TestEnrollment_UpdateStatus_OwnerRemovesAStudent(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	enrollment := s.enroll(t, s.student(t))

	err := s.e.Svc.Enrollment.UpdateStatus(context.Background(), testutil.ActorOf(s.owner), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)
}

func TestEnrollment_UpdateStatus_OtherStudentCannot(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	enrollment := s.enroll(t, s.student(t))

	err := s.e.Svc.Enrollment.UpdateStatus(context.Background(), testutil.ActorOf(s.student(t)), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusDropped})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestEnrollment_UpdateStatus_OnlyDroppedCanBeSetByHand(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.student(t)
	enrollment := s.enroll(t, student)

	for _, status := range []models.EnrollmentStatus{models.EnrollmentStatusCompleted, models.EnrollmentStatusActive} {
		err := s.e.Svc.Enrollment.UpdateStatus(context.Background(), testutil.AdminActor(), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: status})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestEnrollment_UpdateStatus_NotFound(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	err := s.e.Svc.Enrollment.UpdateStatus(context.Background(), testutil.AdminActor(), models.UpdateEnrollmentStatus{ID: uuid.New(), Status: models.EnrollmentStatusDropped})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestEnrollment_GetMyList_CourseCoverHasALink(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	cover := "covers/" + testutil.Uniq() + ".png"
	s.e.Storage.Upload(cover, 1024, "image/png")

	_, err := s.e.Svc.Course.Update(ctx, testutil.ActorOf(s.owner), s.course.ID, models.UpdateCourseRequest{Cover: &cover})
	require.NoError(t, err)

	student := s.student(t)
	s.enroll(t, student)

	list, err := s.e.Svc.Enrollment.GetMyList(ctx, testutil.ActorOf(student), models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, "http://files.test/"+cover, *list.Items[0].CourseCoverURL)
}
