package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestModule(t *testing.T, tc *TestContext, courseID uuid.UUID, orderNumber int) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	id, err := tc.Repo.Module.Create(ctx, models.CreateModule{
		CourseID:    courseID,
		Title:       "module_" + uuid.NewString(),
		OrderNumber: orderNumber,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `
			DELETE FROM lesson_progress
			WHERE lesson_id IN (SELECT id FROM lessons WHERE module_id = $1)`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `
			DELETE FROM lesson_materials
			WHERE lesson_id IN (SELECT id FROM lessons WHERE module_id = $1)`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM lessons WHERE module_id = $1`, id)
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM modules WHERE id = $1`, id)
	})

	return id
}

func TestModuleRepo_Create(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	id := createTestModule(t, tc, courseID, 1)
	require.NotEqual(t, uuid.Nil, id)
}

func TestModuleRepo_Create_DuplicateOrder(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestModule(t, tc, courseID, 1)

	_, err := tc.Repo.Module.Create(context.Background(), models.CreateModule{
		CourseID:    courseID,
		Title:       "duplicate order",
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestModuleRepo_Create_CourseNotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Module.Create(context.Background(), models.CreateModule{
		CourseID:    uuid.New(),
		Title:       "orphan",
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestModuleRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestModule(t, tc, courseID, 1)

	module, err := tc.Repo.Module.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, module.ID)
	require.Equal(t, courseID, module.CourseID)
	require.Equal(t, 1, module.OrderNumber)
}

func TestModuleRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Module.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestModuleRepo_GetListByCourseID(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	second := createTestModule(t, tc, courseID, 2)
	first := createTestModule(t, tc, courseID, 1)

	modules, err := tc.Repo.Module.GetListByCourseID(context.Background(), courseID)

	require.NoError(t, err)
	require.Len(t, modules, 2)
	require.Equal(t, first, modules[0].ID)
	require.Equal(t, second, modules[1].ID)
}

func TestModuleRepo_Update(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestModule(t, tc, courseID, 1)

	title := "renamed_" + uuid.NewString()

	module, err := tc.Repo.Module.Update(context.Background(), models.UpdateModule{
		ID:    id,
		Title: &title,
	})

	require.NoError(t, err)
	require.Equal(t, title, module.Title)
	require.Equal(t, 1, module.OrderNumber)
}

func TestModuleRepo_UpdateOrder(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestModule(t, tc, courseID, 1)

	require.NoError(t, tc.Repo.Module.UpdateOrder(ctx, models.UpdateModuleOrder{
		ID:          id,
		OrderNumber: 5,
	}))

	module, err := tc.Repo.Module.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 5, module.OrderNumber)
}

func TestModuleRepo_UpdateOrder_Conflict(t *testing.T) {
	tc := setupTest(t)

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	createTestModule(t, tc, courseID, 1)
	id := createTestModule(t, tc, courseID, 2)

	err := tc.Repo.Module.UpdateOrder(context.Background(), models.UpdateModuleOrder{
		ID:          id,
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestModuleRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())
	id := createTestModule(t, tc, courseID, 1)

	require.NoError(t, tc.Repo.Module.Delete(ctx, id))

	_, err := tc.Repo.Module.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)

	// the order number can be reused once the module is soft-deleted
	createTestModule(t, tc, courseID, 1)
}
