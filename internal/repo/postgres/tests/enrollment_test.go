package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestEnrollment must be called after the student and the course are
// created: cleanups run in reverse order, so the enrollment is removed before
// the course and the student.
func createTestEnrollment(t *testing.T, tc *TestContext, studentID, courseID uuid.UUID) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	id, err := tc.Repo.Enrollment.Create(ctx, models.CreateEnrollment{
		CourseID:  courseID,
		StudentID: studentID,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM enrollments WHERE id = $1`, id)
	})

	return id
}

func TestEnrollmentRepo_Create(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	id := createTestEnrollment(t, tc, studentID, courseID)
	require.NotEqual(t, uuid.Nil, id)
}

func TestEnrollmentRepo_Create_AlreadyEnrolled(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestEnrollment(t, tc, studentID, courseID)

	_, err := tc.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{
		CourseID:  courseID,
		StudentID: studentID,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestEnrollmentRepo_Create_CourseNotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)

	_, err := tc.Repo.Enrollment.Create(context.Background(), models.CreateEnrollment{
		CourseID:  uuid.New(),
		StudentID: studentID,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestEnrollmentRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestEnrollment(t, tc, studentID, courseID)

	enrollment, err := tc.Repo.Enrollment.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, enrollment.ID)
	require.Equal(t, studentID, enrollment.StudentID)
	require.Equal(t, courseID, enrollment.CourseID)
	require.Equal(t, models.EnrollmentStatusActive, enrollment.Status)
}

func TestEnrollmentRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Enrollment.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestEnrollmentRepo_GetByStudentAndCourse(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestEnrollment(t, tc, studentID, courseID)

	enrollment, err := tc.Repo.Enrollment.GetByStudentAndCourse(context.Background(), studentID, courseID)

	require.NoError(t, err)
	require.Equal(t, id, enrollment.ID)
}

func TestEnrollmentRepo_GetByStudentAndCourse_NotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	_, err := tc.Repo.Enrollment.GetByStudentAndCourse(context.Background(), studentID, courseID)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestEnrollmentRepo_GetListByCourseID(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	first := createTestStudent(t, tc)
	second := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestEnrollment(t, tc, first, courseID)
	dropped := createTestEnrollment(t, tc, second, courseID)

	require.NoError(t, tc.Repo.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{
		ID:     dropped,
		Status: models.EnrollmentStatusDropped,
	}))

	all, total, err := tc.Repo.Enrollment.GetListByCourseID(ctx, courseID, models.EnrollmentFilter{})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, all, 2)
	require.NotEmpty(t, all[0].StudentName)
	require.NotEmpty(t, all[0].Email)

	status := models.EnrollmentStatusDropped
	filtered, total, err := tc.Repo.Enrollment.GetListByCourseID(ctx, courseID, models.EnrollmentFilter{Status: &status})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, dropped, filtered[0].ID)
}

func TestEnrollmentRepo_GetMyList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	doneLesson := createTestLesson(t, tc, moduleID, 1)
	createTestLesson(t, tc, moduleID, 2)
	id := createTestEnrollment(t, tc, studentID, courseID)

	_, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID,
		LessonID:  doneLesson,
		Completed: true,
	})
	require.NoError(t, err)

	enrollments, total, err := tc.Repo.Enrollment.GetMyList(ctx, studentID, models.EnrollmentFilter{})

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, enrollments, 1)
	require.Equal(t, id, enrollments[0].ID)
	require.NotEmpty(t, enrollments[0].CourseTitle)
	require.Equal(t, 2, enrollments[0].TotalLessons)
	require.Equal(t, 1, enrollments[0].CompletedLessons)
	require.Equal(t, 50.0, enrollments[0].ProgressPercent)
}

func TestEnrollmentRepo_GetMyList_CourseWithoutLessons(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestEnrollment(t, tc, studentID, courseID)

	enrollments, _, err := tc.Repo.Enrollment.GetMyList(context.Background(), studentID, models.EnrollmentFilter{})

	require.NoError(t, err)
	require.Len(t, enrollments, 1)
	require.Zero(t, enrollments[0].TotalLessons)
	require.Zero(t, enrollments[0].ProgressPercent)
}

func TestEnrollmentRepo_UpdateStatus(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestEnrollment(t, tc, studentID, courseID)

	require.NoError(t, tc.Repo.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{
		ID:     id,
		Status: models.EnrollmentStatusCompleted,
	}))

	enrollment, err := tc.Repo.Enrollment.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, models.EnrollmentStatusCompleted, enrollment.Status)
}

func TestEnrollmentRepo_UpdateStatus_NotFound(t *testing.T) {
	tc := setupTest(t)

	err := tc.Repo.Enrollment.UpdateStatus(context.Background(), models.UpdateEnrollmentStatus{
		ID:     uuid.New(),
		Status: models.EnrollmentStatusCompleted,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}
