package postgres

import (
	"context"
	"errors"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type moduleRepo struct {
	db DBTX
}

func NewModuleRepository(db DBTX) *moduleRepo {
	return &moduleRepo{
		db: db,
	}
}

// scanModule scans a row selected with the module columns in the order used
// below. It works for both a single row and rows of a list.
func scanModule(row pgx.Row, module *models.Module) error {
	return row.Scan(
		&module.ID,
		&module.CourseID,
		&module.Title,
		&module.Description,
		&module.OrderNumber,
		&module.CreatedAt,
		&module.UpdatedAt,
	)
}

func (r *moduleRepo) Create(
	ctx context.Context,
	module models.CreateModule,
) (uuid.UUID, error) {
	query := `
		INSERT INTO modules (
			id,
			course_id,
			title,
			description,
			order_number
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		module.CourseID,
		module.Title,
		module.Description,
		module.OrderNumber,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateModule",
				"module order number already exists in this course",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateModule",
				"course not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateModule",
			"failed to create module",
			err,
		)
	}

	return id, nil
}

func (r *moduleRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Module, error) {
	query := `
		SELECT
			id,
			course_id,
			title,
			description,
			order_number,
			created_at,
			updated_at
		FROM modules
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var module models.Module

	err := scanModule(r.db.QueryRow(ctx, query, id), &module)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetModuleByID",
				"module not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetModuleByID",
			"failed to get module",
			err,
		)
	}

	return &module, nil
}

func (r *moduleRepo) GetListByCourseID(
	ctx context.Context,
	courseID uuid.UUID,
) ([]models.Module, error) {
	query := `
		SELECT
			id,
			course_id,
			title,
			description,
			order_number,
			created_at,
			updated_at
		FROM modules
		WHERE course_id = $1
			AND deleted_at IS NULL
		ORDER BY order_number;
	`

	rows, err := r.db.Query(ctx, query, courseID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetModuleList",
			"failed to get modules",
			err,
		)
	}
	defer rows.Close()

	modules := make([]models.Module, 0)

	for rows.Next() {
		var module models.Module

		if err := scanModule(rows, &module); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetModuleList",
				"failed to scan modules",
				err,
			)
		}

		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetModuleList",
			"failed to read modules",
			err,
		)
	}

	return modules, nil
}

func (r *moduleRepo) Update(
	ctx context.Context,
	module models.UpdateModule,
) (*models.Module, error) {
	query := `
		UPDATE modules
		SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			course_id,
			title,
			description,
			order_number,
			created_at,
			updated_at;
	`

	var updatedModule models.Module

	err := scanModule(
		r.db.QueryRow(
			ctx,
			query,
			module.ID,
			module.Title,
			module.Description,
		),
		&updatedModule,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateModule",
				"module not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateModule",
			"failed to update module",
			err,
		)
	}

	return &updatedModule, nil
}

func (r *moduleRepo) UpdateOrder(
	ctx context.Context,
	module models.UpdateModuleOrder,
) error {
	query := `
		UPDATE modules
		SET
			order_number = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		module.ID,
		module.OrderNumber,
	)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return apperror.Conflict(
				"repository",
				"UpdateModuleOrder",
				"module order number already exists in this course",
				apperror.ErrAlreadyExists,
			)
		}

		return apperror.Internal(
			"repository",
			"UpdateModuleOrder",
			"failed to update module order",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"UpdateModuleOrder",
			"module not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *moduleRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE modules
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
			"DeleteModule",
			"failed to delete module",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteModule",
			"module not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
