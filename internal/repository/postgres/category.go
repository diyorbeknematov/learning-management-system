package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type categoryRepo struct {
	db DBTX
}

func NewCategoryRepository(db DBTX) *categoryRepo {
	return &categoryRepo{
		db: db,
	}
}

func (r *categoryRepo) Create(ctx context.Context, category models.CreateCategory) (uuid.UUID, error) {
	query := `
		INSERT INTO categories (
			id,
			name,
			description
		)
		VALUES($1, $2, $3)
		RETURNING id;
	`

	var id uuid.UUID = uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		category.Name,
		category.Description,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateCategory",
				"category name already exists",
				apperror.ErrAlreadyExists,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateCategory",
			"failed to create category",
			err,
		)
	}

	return id, nil
}

func (r *categoryRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Category, error) {
	query := `
		SELECT
			id,
			name,
			description,
			created_at,
			updated_at
		FROM categories
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var category models.Category

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetCategoryByID",
				"category not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetCategoryByID",
			"failed to get category",
			err,
		)
	}

	return &category, nil
}

func (r *categoryRepo) GetList(
	ctx context.Context,
	filter models.CategoryFilter,
) ([]models.Category, int, error) {
	baseQuery := `
		SELECT
			id,
			name,
			description,
			created_at,
			updated_at
		FROM categories
		WHERE deleted_at IS NULL
	`
	countQuery := `
		SELECT COUNT(*)
		FROM categories
		WHERE deleted_at IS NULL
	`

	conditions := make([]string, 0)
	args := make([]any, 0)
	argIndex := 1

	if filter.Name != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("name ILIKE $%d", argIndex),
		)

		args = append(args, "%"+*filter.Name+"%")
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCategoriesList",
			"failed to get categories",
			err,
		)
	}
	defer rows.Close()

	categories := make([]models.Category, 0)

	for rows.Next() {
		var category models.Category

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetCategoriesList",
				"failed to scan categories",
				err,
			)
		}

		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCategoriesList",
			"failed to read categories",
			err,
		)
	}

	var total int

	err = r.db.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetCategoriesList",
			"failed to get total count",
			err,
		)
	}

	return categories, total, nil
}

func (r *categoryRepo) Update(
	ctx context.Context,
	category models.UpdateCategory,
) (*models.Category, error) {
	query := `
		UPDATE categories
		SET
			name = COALESCE($1, name),
			description = COALESCE($2, description),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
			AND deleted_at IS NULL
		RETURNING
			id,
			name,
			description,
			created_at,
			updated_at;
	`

	var updatedCategory models.Category

	err := r.db.QueryRow(
		ctx,
		query,
		category.Name,
		category.Description,
		category.ID,
	).Scan(
		&updatedCategory.ID,
		&updatedCategory.Name,
		&updatedCategory.Description,
		&updatedCategory.CreatedAt,
		&updatedCategory.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateCategory",
				"category not found",
				apperror.ErrNotFound,
			)
		}

		if pgerr.IsUniqueViolation(err) {
			return nil, apperror.Conflict(
				"repository",
				"UpdateCategory",
				"category name already exists",
				apperror.ErrAlreadyExists,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateCategory",
			"failed to update category",
			err,
		)
	}

	return &updatedCategory, nil
}

func (r *categoryRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE categories
		SET
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		id,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteCategory",
			"failed to delete category",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteCategory",
			"category not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
