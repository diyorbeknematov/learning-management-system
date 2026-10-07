package tests

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestCategory(t *testing.T, tc *TestContext) uuid.UUID {
	t.Helper()

	id, err := tc.Repo.Category.Create(context.Background(), models.CreateCategory{
		Name: "cat_" + uuid.NewString(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := tc.DB.Pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, id)
		require.NoError(t, err)
	})

	return id
}

func TestCategoryRepo_Create(t *testing.T) {
	tc := setupTest(t)

	id := createTestCategory(t, tc)
	require.NotEqual(t, uuid.Nil, id)
}

func TestCategoryRepo_Create_DuplicateName(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	name := "dup_" + uuid.NewString()

	_, err := tc.Repo.Category.Create(ctx, models.CreateCategory{Name: name})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM categories WHERE name = $1`, name)
	})

	_, err = tc.Repo.Category.Create(ctx, models.CreateCategory{Name: name})

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeConflict, appErr.Code)
}

func TestCategoryRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	id := createTestCategory(t, tc)

	category, err := tc.Repo.Category.GetByID(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, id, category.ID)
}

func TestCategoryRepo_GetByID_NotFound(t *testing.T) {
	tc := setupTest(t)

	_, err := tc.Repo.Category.GetByID(context.Background(), uuid.New())

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestCategoryRepo_GetList(t *testing.T) {
	tc := setupTest(t)

	id := createTestCategory(t, tc)

	category, err := tc.Repo.Category.GetByID(context.Background(), id)
	require.NoError(t, err)

	categories, total, err := tc.Repo.Category.GetList(
		context.Background(),
		models.CategoryFilter{Name: &category.Name},
	)

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, categories, 1)
	require.Equal(t, id, categories[0].ID)
}

func TestCategoryRepo_Update(t *testing.T) {
	tc := setupTest(t)

	id := createTestCategory(t, tc)

	name := "renamed_" + uuid.NewString()

	category, err := tc.Repo.Category.Update(
		context.Background(),
		models.UpdateCategory{ID: id, Name: &name},
	)

	require.NoError(t, err)
	require.Equal(t, name, category.Name)
}

func TestCategoryRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	name := "del_" + uuid.NewString()

	id, err := tc.Repo.Category.Create(ctx, models.CreateCategory{Name: name})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = tc.DB.Pool.Exec(ctx, `DELETE FROM categories WHERE name = $1`, name)
	})

	require.NoError(t, tc.Repo.Category.Delete(ctx, id))

	_, err = tc.Repo.Category.GetByID(ctx, id)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)

	// the name can be reused once the category is soft-deleted
	_, err = tc.Repo.Category.Create(ctx, models.CreateCategory{Name: name})
	require.NoError(t, err)
}
