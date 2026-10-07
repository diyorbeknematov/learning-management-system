package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestLesson(t *testing.T, tc *TestContext, moduleID uuid.UUID, orderNumber int) uuid.UUID {
	t.Helper()

	id, err := tc.Repo.Lesson.Create(context.Background(), models.CreateLesson{
		ModuleID:    moduleID,
		Title:       "lesson_" + uuid.NewString(),
		OrderNumber: orderNumber,
	})
	require.NoError(t, err)

	return id
}

func createTestLessonModule(t *testing.T, tc *TestContext) uuid.UUID {
	t.Helper()

	courseID := createTestCourse(t, tc, "course_"+uuid.NewString())

	return createTestModule(t, tc, courseID, 1)
}

func TestLessonRepo_Create(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)

	id := createTestLesson(t, tc, moduleID, 1)
	require.NotEqual(t, uuid.Nil, id)
}

func TestLessonRepo_Create_DuplicateOrder(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)
	createTestLesson(t, tc, moduleID, 1)

	_, err := tc.Repo.Lesson.Create(context.Background(), models.CreateLesson{
		ModuleID:    moduleID,
		Title:       "duplicate order",
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestLessonRepo_Create_ModuleNotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Lesson.Create(context.Background(), models.CreateLesson{
		ModuleID:    uuid.New(),
		Title:       "orphan",
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)
	id := createTestLesson(t, tc, moduleID, 1)

	lesson, err := tc.Repo.Lesson.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, lesson.ID)
	require.Equal(t, moduleID, lesson.ModuleID)
	require.False(t, lesson.IsPreview)
}

func TestLessonRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Lesson.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_GetListByModuleID(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)
	second := createTestLesson(t, tc, moduleID, 2)
	first := createTestLesson(t, tc, moduleID, 1)

	lessons, err := tc.Repo.Lesson.GetListByModuleID(context.Background(), moduleID)

	require.NoError(t, err)
	require.Len(t, lessons, 2)
	require.Equal(t, first, lessons[0].ID)
	require.Equal(t, second, lessons[1].ID)
}

func TestLessonRepo_Update(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)
	id := createTestLesson(t, tc, moduleID, 1)

	title := "renamed_" + uuid.NewString()
	duration := 15
	isPreview := true

	lesson, err := tc.Repo.Lesson.Update(context.Background(), models.UpdateLesson{
		ID:        id,
		Title:     &title,
		Duration:  &duration,
		IsPreview: &isPreview,
	})

	require.NoError(t, err)
	require.Equal(t, title, lesson.Title)
	require.Equal(t, duration, *lesson.Duration)
	require.True(t, lesson.IsPreview)
	require.Equal(t, 1, lesson.OrderNumber)
}

func TestLessonRepo_UpdateOrder(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	moduleID := createTestLessonModule(t, tc)
	id := createTestLesson(t, tc, moduleID, 1)

	require.NoError(t, tc.Repo.Lesson.UpdateOrder(ctx, models.UpdateLessonOrder{
		ID:          id,
		OrderNumber: 5,
	}))

	lesson, err := tc.Repo.Lesson.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 5, lesson.OrderNumber)
}

func TestLessonRepo_UpdateOrder_Conflict(t *testing.T) {
	tc := setupTest(t)

	moduleID := createTestLessonModule(t, tc)
	createTestLesson(t, tc, moduleID, 1)
	id := createTestLesson(t, tc, moduleID, 2)

	err := tc.Repo.Lesson.UpdateOrder(context.Background(), models.UpdateLessonOrder{
		ID:          id,
		OrderNumber: 1,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestLessonRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	moduleID := createTestLessonModule(t, tc)
	id := createTestLesson(t, tc, moduleID, 1)

	require.NoError(t, tc.Repo.Lesson.Delete(ctx, id))

	_, err := tc.Repo.Lesson.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_DeleteByModuleID(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	moduleID := createTestLessonModule(t, tc)
	createTestLesson(t, tc, moduleID, 1)
	createTestLesson(t, tc, moduleID, 2)

	require.NoError(t, tc.Repo.Lesson.DeleteByModuleID(ctx, moduleID))

	lessons, err := tc.Repo.Lesson.GetListByModuleID(ctx, moduleID)
	require.NoError(t, err)
	require.Empty(t, lessons)
}

func createTestMaterial(t *testing.T, tc *TestContext, lessonID uuid.UUID) uuid.UUID {
	t.Helper()

	content := "Lesson text"

	id, err := tc.Repo.Lesson.CreateMaterial(context.Background(), models.CreateLessonMaterial{
		LessonID: lessonID,
		Type:     models.MaterialTypeText,
		Content:  &content,
	})
	require.NoError(t, err)

	return id
}

func TestLessonRepo_CreateMaterial(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)

	objectKey := "materials/slides.pdf"
	fileName := "slides.pdf"
	mimeType := "application/pdf"
	fileSize := int64(2048)

	id, err := tc.Repo.Lesson.CreateMaterial(ctx, models.CreateLessonMaterial{
		LessonID:  lessonID,
		Type:      models.MaterialTypeFile,
		ObjectKey: &objectKey,
		FileName:  &fileName,
		MimeType:  &mimeType,
		FileSize:  &fileSize,
	})
	require.NoError(t, err)

	material, err := tc.Repo.Lesson.GetMaterialByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, models.MaterialTypeFile, material.Type)
	require.Equal(t, objectKey, *material.ObjectKey)
	require.Equal(t, fileSize, *material.FileSize)
}

func TestLessonRepo_CreateMaterial_MissingContent(t *testing.T) {
	tc := setupTest(t)

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)

	_, err := tc.Repo.Lesson.CreateMaterial(context.Background(), models.CreateLessonMaterial{
		LessonID: lessonID,
		Type:     models.MaterialTypeVideo,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeInvalidInput, appErr.Code)
}

func TestLessonRepo_CreateMaterial_LessonNotFound(t *testing.T) {
	tc := setupTest(t)

	content := "text"

	_, err := tc.Repo.Lesson.CreateMaterial(context.Background(), models.CreateLessonMaterial{
		LessonID: uuid.New(),
		Type:     models.MaterialTypeText,
		Content:  &content,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_GetMaterialByID(t *testing.T) {
	tc := setupTest(t)

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)
	id := createTestMaterial(t, tc, lessonID)

	material, err := tc.Repo.Lesson.GetMaterialByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, material.ID)
	require.Equal(t, lessonID, material.LessonID)
	require.Equal(t, "Lesson text", *material.Content)
}

func TestLessonRepo_GetMaterialByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Lesson.GetMaterialByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_GetMaterials(t *testing.T) {
	tc := setupTest(t)

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)
	createTestMaterial(t, tc, lessonID)
	createTestMaterial(t, tc, lessonID)

	materials, err := tc.Repo.Lesson.GetMaterials(context.Background(), lessonID)

	require.NoError(t, err)
	require.Len(t, materials, 2)
}

func TestLessonRepo_UpdateMaterial(t *testing.T) {
	tc := setupTest(t)

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)
	id := createTestMaterial(t, tc, lessonID)

	content := "Updated text"

	material, err := tc.Repo.Lesson.UpdateMaterial(context.Background(), models.UpdateLessonMaterial{
		ID:      id,
		Content: &content,
	})

	require.NoError(t, err)
	require.Equal(t, content, *material.Content)
	require.Equal(t, models.MaterialTypeText, material.Type)
}

func TestLessonRepo_UpdateMaterial_NotFound(t *testing.T) {
	tc := setupTest(t)

	content := "x"

	_, err := tc.Repo.Lesson.UpdateMaterial(context.Background(), models.UpdateLessonMaterial{
		ID:      uuid.New(),
		Content: &content,
	})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestLessonRepo_DeleteMaterial(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	lessonID := createTestLesson(t, tc, createTestLessonModule(t, tc), 1)
	id := createTestMaterial(t, tc, lessonID)

	require.NoError(t, tc.Repo.Lesson.DeleteMaterial(ctx, id))

	_, err := tc.Repo.Lesson.GetMaterialByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}
