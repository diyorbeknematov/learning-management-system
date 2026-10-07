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

func (s shop) enrollmentStatus(t *testing.T, enrollmentID uuid.UUID) models.EnrollmentStatus {
	t.Helper()

	enrollment, err := s.e.Svc.Enrollment.GetByID(context.Background(), testutil.AdminActor(), enrollmentID)
	require.NoError(t, err)

	return enrollment.Status
}

func (s shop) mark(t *testing.T, student *models.User, lessonID uuid.UUID, completed bool) {
	t.Helper()

	_, err := s.e.Svc.Progress.SetLessonProgress(context.Background(), testutil.ActorOf(student), lessonID, completed)
	require.NoError(t, err)
}

func TestProgress_SetLessonProgress(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.student(t)
	s.enroll(t, student)

	progress, err := s.e.Svc.Progress.SetLessonProgress(context.Background(), testutil.ActorOf(student), s.lessons[0], true)
	require.NoError(t, err)
	require.Equal(t, student.ID, progress.StudentID)
	require.Equal(t, s.lessons[0], progress.LessonID)
	require.True(t, progress.Completed)
	require.NotNil(t, progress.CompletedAt)
}

func TestProgress_SetLessonProgress_CanBeTakenBack(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.student(t)
	s.enroll(t, student)

	s.mark(t, student, s.lessons[0], true)

	progress, err := s.e.Svc.Progress.SetLessonProgress(context.Background(), testutil.ActorOf(student), s.lessons[0], false)
	require.NoError(t, err)
	require.False(t, progress.Completed)
	require.Nil(t, progress.CompletedAt)
}

func TestProgress_SetLessonProgress_NotEnrolled(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Progress.SetLessonProgress(context.Background(), testutil.ActorOf(s.student(t)), s.lessons[0], true)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestProgress_SetLessonProgress_DroppedCourse(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	enrollment := s.enroll(t, student)

	err := s.e.Svc.Enrollment.UpdateStatus(ctx, testutil.ActorOf(student), models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusDropped})
	require.NoError(t, err)

	_, err = s.e.Svc.Progress.SetLessonProgress(ctx, testutil.ActorOf(student), s.lessons[0], true)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestProgress_SetLessonProgress_UnknownLesson(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Progress.SetLessonProgress(context.Background(), testutil.ActorOf(s.student(t)), uuid.New(), true)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestProgress_SetLessonProgress_CompletesTheEnrollment(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.student(t)
	enrollment := s.enroll(t, student)

	s.mark(t, student, s.lessons[0], true)
	require.Equal(t, models.EnrollmentStatusActive, s.enrollmentStatus(t, enrollment.ID))

	s.mark(t, student, s.lessons[1], true)
	require.Equal(t, models.EnrollmentStatusCompleted, s.enrollmentStatus(t, enrollment.ID))

	// taking a lesson back makes the course active again
	s.mark(t, student, s.lessons[1], false)
	require.Equal(t, models.EnrollmentStatusActive, s.enrollmentStatus(t, enrollment.ID))
}

func TestProgress_SetLessonProgress_MarkingTwiceIsHarmless(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.student(t)
	enrollment := s.enroll(t, student)

	s.mark(t, student, s.lessons[0], true)
	s.mark(t, student, s.lessons[0], true)

	require.Equal(t, models.EnrollmentStatusCompleted, s.enrollmentStatus(t, enrollment.ID))
}

func TestProgress_GetLessonProgress(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	s.enroll(t, student)

	untouched, err := s.e.Svc.Progress.GetLessonProgress(ctx, testutil.ActorOf(student), s.lessons[0])
	require.NoError(t, err)
	require.False(t, untouched.Completed)
	require.Equal(t, s.lessons[0], untouched.LessonID)

	s.mark(t, student, s.lessons[0], true)

	done, err := s.e.Svc.Progress.GetLessonProgress(ctx, testutil.ActorOf(student), s.lessons[0])
	require.NoError(t, err)
	require.True(t, done.Completed)
}

func TestProgress_GetLessonProgress_UnknownLesson(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Progress.GetLessonProgress(context.Background(), testutil.ActorOf(s.student(t)), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestProgress_GetCourseProgress(t *testing.T) {
	s := newShop(t, 0, 4, nil, 0)

	student := s.student(t)
	s.enroll(t, student)

	s.mark(t, student, s.lessons[0], true)
	s.mark(t, student, s.lessons[2], true)

	progress, err := s.e.Svc.Progress.GetCourseProgress(context.Background(), testutil.ActorOf(student), s.course.ID)
	require.NoError(t, err)
	require.Equal(t, 4, progress.TotalLessons)
	require.Equal(t, 2, progress.CompletedLessons)
	require.Equal(t, 50.0, progress.ProgressPercent)
	require.ElementsMatch(t, []uuid.UUID{s.lessons[0], s.lessons[2]}, progress.CompletedLessonIDs)
}

func TestProgress_GetCourseProgress_NotEnrolled(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Progress.GetCourseProgress(context.Background(), testutil.ActorOf(s.student(t)), s.course.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestProgress_GetStudents(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	ctx := context.Background()
	working, idle := s.student(t), s.student(t)
	s.enroll(t, working)
	s.enroll(t, idle)

	s.mark(t, working, s.lessons[0], true)

	list, err := s.e.Svc.Progress.GetStudents(ctx, testutil.ActorOf(s.owner), s.course.ID, models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Equal(t, 2, list.Total)

	byID := map[uuid.UUID]models.StudentProgress{}
	for _, student := range list.Items {
		byID[student.StudentID] = student
	}

	require.Equal(t, 50.0, byID[working.ID].ProgressPercent)
	require.NotNil(t, byID[working.ID].LastActivityAt)
	require.Zero(t, byID[idle.ID].ProgressPercent)
	require.Nil(t, byID[idle.ID].LastActivityAt)
}

func TestProgress_GetStudents_OnlyTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	for _, other := range []*models.User{s.student(t), s.e.NewUser(t, models.RoleInstructor)} {
		_, err := s.e.Svc.Progress.GetStudents(context.Background(), testutil.ActorOf(other), s.course.ID, models.EnrollmentFilter{})
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}
}

func TestProgress_GetStudentProgress(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	ctx := context.Background()
	student := s.student(t)
	s.enroll(t, student)
	s.mark(t, student, s.lessons[1], true)

	progress, err := s.e.Svc.Progress.GetStudentProgress(ctx, testutil.ActorOf(s.owner), s.course.ID, student.ID)
	require.NoError(t, err)
	require.Equal(t, 1, progress.CompletedLessons)
	require.Equal(t, []uuid.UUID{s.lessons[1]}, progress.CompletedLessonIDs)

	_, err = s.e.Svc.Progress.GetStudentProgress(ctx, testutil.AdminActor(), s.course.ID, student.ID)
	require.NoError(t, err)
}

func TestProgress_GetStudentProgress_NotEnrolledStudent(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	_, err := s.e.Svc.Progress.GetStudentProgress(context.Background(), testutil.ActorOf(s.owner), s.course.ID, s.student(t).ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestProgress_GetStudentProgress_OnlyTheOwner(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.student(t)
	s.enroll(t, student)

	_, err := s.e.Svc.Progress.GetStudentProgress(context.Background(), testutil.ActorOf(s.e.NewUser(t, models.RoleInstructor)), s.course.ID, student.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}
