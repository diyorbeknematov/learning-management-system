package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestProgressLesson creates a student, a course, a module and a lesson
// in the order their cleanups need (the module cleanup removes the progress).
func createTestProgressLesson(t *testing.T, tc *TestContext) (studentID, courseID, lessonID uuid.UUID) {
	t.Helper()

	studentID = createTestStudent(t, tc)
	courseID = createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	lessonID = createTestLesson(t, tc, moduleID, 1)

	return studentID, courseID, lessonID
}

func TestProgressRepo_Set(t *testing.T) {
	tc := setupTest(t)

	studentID, _, lessonID := createTestProgressLesson(t, tc)

	progress, err := tc.Repo.Progress.Set(context.Background(), models.SetLessonProgress{
		StudentID: studentID,
		LessonID:  lessonID,
		Completed: true,
	})

	require.NoError(t, err)
	require.Equal(t, studentID, progress.StudentID)
	require.Equal(t, lessonID, progress.LessonID)
	require.True(t, progress.Completed)
	require.NotNil(t, progress.CompletedAt)
}

func TestProgressRepo_Set_Toggle(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID, _, lessonID := createTestProgressLesson(t, tc)

	first, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID, LessonID: lessonID, Completed: true,
	})
	require.NoError(t, err)

	second, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID, LessonID: lessonID, Completed: true,
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.CompletedAt, second.CompletedAt)

	third, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID, LessonID: lessonID, Completed: false,
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, third.ID)
	require.False(t, third.Completed)
	require.Nil(t, third.CompletedAt)
}

func TestProgressRepo_Set_LessonNotFound(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)

	_, err := tc.Repo.Progress.Set(context.Background(), models.SetLessonProgress{
		StudentID: studentID,
		LessonID:  uuid.New(),
		Completed: true,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestProgressRepo_GetByLesson(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID, _, lessonID := createTestProgressLesson(t, tc)

	_, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID, LessonID: lessonID, Completed: true,
	})
	require.NoError(t, err)

	progress, err := tc.Repo.Progress.GetByLesson(ctx, studentID, lessonID)

	require.NoError(t, err)
	require.True(t, progress.Completed)
}

func TestProgressRepo_GetByLesson_NotFound(t *testing.T) {
	tc := setupTest(t)

	studentID, _, lessonID := createTestProgressLesson(t, tc)

	_, err := tc.Repo.Progress.GetByLesson(context.Background(), studentID, lessonID)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestProgressRepo_GetCourseProgress(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	first := createTestLesson(t, tc, moduleID, 1)
	second := createTestLesson(t, tc, moduleID, 2)
	createTestLesson(t, tc, moduleID, 3)
	createTestLesson(t, tc, moduleID, 4)

	for _, lessonID := range []uuid.UUID{first, second} {
		_, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
			StudentID: studentID, LessonID: lessonID, Completed: true,
		})
		require.NoError(t, err)
	}

	progress, err := tc.Repo.Progress.GetCourseProgress(ctx, studentID, courseID)

	require.NoError(t, err)
	require.Equal(t, courseID, progress.CourseID)
	require.Equal(t, 4, progress.TotalLessons)
	require.Equal(t, 2, progress.CompletedLessons)
	require.Equal(t, 50.0, progress.ProgressPercent)
}

func TestProgressRepo_GetCourseProgress_NoLessons(t *testing.T) {
	tc := setupTest(t)

	studentID := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	progress, err := tc.Repo.Progress.GetCourseProgress(context.Background(), studentID, courseID)

	require.NoError(t, err)
	require.Zero(t, progress.TotalLessons)
	require.Zero(t, progress.CompletedLessons)
	require.Zero(t, progress.ProgressPercent)
}

func TestProgressRepo_GetCompletedLessonIDs(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	studentID, courseID, lessonID := createTestProgressLesson(t, tc)

	ids, err := tc.Repo.Progress.GetCompletedLessonIDs(ctx, studentID, courseID)
	require.NoError(t, err)
	require.Empty(t, ids)

	_, err = tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: studentID, LessonID: lessonID, Completed: true,
	})
	require.NoError(t, err)

	ids, err = tc.Repo.Progress.GetCompletedLessonIDs(ctx, studentID, courseID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{lessonID}, ids)
}

func TestProgressRepo_GetStudentList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	active := createTestStudent(t, tc)
	idle := createTestStudent(t, tc)
	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	moduleID := createTestModule(t, tc, courseID, 1)
	lessonID := createTestLesson(t, tc, moduleID, 1)
	createTestLesson(t, tc, moduleID, 2)
	createTestEnrollment(t, tc, active, courseID)
	createTestEnrollment(t, tc, idle, courseID)

	_, err := tc.Repo.Progress.Set(ctx, models.SetLessonProgress{
		StudentID: active, LessonID: lessonID, Completed: true,
	})
	require.NoError(t, err)

	students, total, err := tc.Repo.Progress.GetStudentList(ctx, courseID, models.EnrollmentFilter{})

	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, students, 2)

	byID := map[uuid.UUID]models.StudentProgress{}
	for _, student := range students {
		byID[student.StudentID] = student
	}

	require.Equal(t, 2, byID[active].TotalLessons)
	require.Equal(t, 1, byID[active].CompletedLessons)
	require.Equal(t, 50.0, byID[active].ProgressPercent)
	require.NotNil(t, byID[active].LastActivityAt)
	require.NotEmpty(t, byID[active].FullName)

	require.Zero(t, byID[idle].CompletedLessons)
	require.Zero(t, byID[idle].ProgressPercent)
	require.Nil(t, byID[idle].LastActivityAt)
}
