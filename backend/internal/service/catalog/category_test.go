package catalog_test

import (
	"context"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCategory_Create(t *testing.T) {
	e := testutil.Setup(t)

	description := "All about Go"
	name := "cat-" + testutil.Uniq()

	category, err := e.Svc.Category.Create(context.Background(), models.CreateCategory{Name: name, Description: &description})
	require.NoError(t, err)

	t.Cleanup(func() { e.Exec(t, `DELETE FROM categories WHERE id = $1`, category.ID) })

	require.Equal(t, name, category.Name)
	require.Equal(t, description, *category.Description)
}

func TestCategory_Create_DuplicateName(t *testing.T) {
	e := testutil.Setup(t)

	existing := e.NewCategory(t)

	_, err := e.Svc.Category.Create(context.Background(), models.CreateCategory{Name: existing.Name})
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestCategory_GetByID(t *testing.T) {
	e := testutil.Setup(t)

	category := e.NewCategory(t)

	found, err := e.Svc.Category.GetByID(context.Background(), category.ID)
	require.NoError(t, err)
	require.Equal(t, category.ID, found.ID)
}

func TestCategory_GetByID_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Category.GetByID(context.Background(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCategory_GetList(t *testing.T) {
	e := testutil.Setup(t)

	category := e.NewCategory(t)

	list, err := e.Svc.Category.GetList(context.Background(), models.CategoryFilter{Name: &category.Name})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, category.ID, list.Items[0].ID)
	require.Equal(t, 1, list.Page)
}

func TestCategory_Update(t *testing.T) {
	e := testutil.Setup(t)

	category := e.NewCategory(t)
	name := "renamed-" + testutil.Uniq()

	updated, err := e.Svc.Category.Update(context.Background(), models.UpdateCategory{ID: category.ID, Name: &name})
	require.NoError(t, err)
	require.Equal(t, name, updated.Name)
}

func TestCategory_Update_DuplicateName(t *testing.T) {
	e := testutil.Setup(t)

	first := e.NewCategory(t)
	second := e.NewCategory(t)

	_, err := e.Svc.Category.Update(context.Background(), models.UpdateCategory{ID: second.ID, Name: &first.Name})
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestCategory_Update_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	name := "x-" + testutil.Uniq()

	_, err := e.Svc.Category.Update(context.Background(), models.UpdateCategory{ID: uuid.New(), Name: &name})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCategory_Delete(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	category := e.NewCategory(t)

	require.NoError(t, e.Svc.Category.Delete(ctx, category.ID))

	_, err := e.Svc.Category.GetByID(ctx, category.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	testutil.RequireCode(t, e.Svc.Category.Delete(ctx, category.ID), apperror.CodeNotFound)
}
