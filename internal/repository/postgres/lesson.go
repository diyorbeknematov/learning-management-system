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

type lessonRepo struct {
	db DBTX
}

func NewLessonRepository(db DBTX) *lessonRepo {
	return &lessonRepo{
		db: db,
	}
}

// scanLesson scans a row selected with the lesson columns in the order used
// below. It works for both a single row and rows of a list.
func scanLesson(row pgx.Row, lesson *models.Lesson) error {
	return row.Scan(
		&lesson.ID,
		&lesson.ModuleID,
		&lesson.Title,
		&lesson.Duration,
		&lesson.OrderNumber,
		&lesson.IsPreview,
		&lesson.CreatedAt,
		&lesson.UpdatedAt,
	)
}

// scanMaterial scans a row selected with the material columns in the order
// used below. It works for both a single row and rows of a list.
func scanMaterial(row pgx.Row, material *models.LessonMaterial) error {
	return row.Scan(
		&material.ID,
		&material.LessonID,
		&material.Type,
		&material.Content,
		&material.ObjectKey,
		&material.FileName,
		&material.MimeType,
		&material.FileSize,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
}

func (r *lessonRepo) Create(
	ctx context.Context,
	lesson models.CreateLesson,
) (uuid.UUID, error) {
	query := `
		INSERT INTO lessons (
			id,
			module_id,
			title,
			duration,
			order_number,
			is_preview
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		lesson.ModuleID,
		lesson.Title,
		lesson.Duration,
		lesson.OrderNumber,
		lesson.IsPreview,
	).Scan(&id)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateLesson",
				"lesson order number already exists in this module",
				apperror.ErrAlreadyExists,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateLesson",
				"module not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateLesson",
			"failed to create lesson",
			err,
		)
	}

	return id, nil
}

func (r *lessonRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.Lesson, error) {
	query := `
		SELECT
			id,
			module_id,
			title,
			duration,
			order_number,
			is_preview,
			created_at,
			updated_at
		FROM lessons
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var lesson models.Lesson

	err := scanLesson(r.db.QueryRow(ctx, query, id), &lesson)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetLessonByID",
				"lesson not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetLessonByID",
			"failed to get lesson",
			err,
		)
	}

	return &lesson, nil
}

func (r *lessonRepo) GetListByModuleID(
	ctx context.Context,
	moduleID uuid.UUID,
) ([]models.Lesson, error) {
	query := `
		SELECT
			id,
			module_id,
			title,
			duration,
			order_number,
			is_preview,
			created_at,
			updated_at
		FROM lessons
		WHERE module_id = $1
			AND deleted_at IS NULL
		ORDER BY order_number;
	`

	rows, err := r.db.Query(ctx, query, moduleID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetLessonList",
			"failed to get lessons",
			err,
		)
	}
	defer rows.Close()

	lessons := make([]models.Lesson, 0)

	for rows.Next() {
		var lesson models.Lesson

		if err := scanLesson(rows, &lesson); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetLessonList",
				"failed to scan lessons",
				err,
			)
		}

		lessons = append(lessons, lesson)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetLessonList",
			"failed to read lessons",
			err,
		)
	}

	return lessons, nil
}

func (r *lessonRepo) Update(
	ctx context.Context,
	lesson models.UpdateLesson,
) (*models.Lesson, error) {
	query := `
		UPDATE lessons
		SET
			title = COALESCE($2, title),
			duration = COALESCE($3, duration),
			is_preview = COALESCE($4, is_preview),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			module_id,
			title,
			duration,
			order_number,
			is_preview,
			created_at,
			updated_at;
	`

	var updatedLesson models.Lesson

	err := scanLesson(
		r.db.QueryRow(
			ctx,
			query,
			lesson.ID,
			lesson.Title,
			lesson.Duration,
			lesson.IsPreview,
		),
		&updatedLesson,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateLesson",
				"lesson not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateLesson",
			"failed to update lesson",
			err,
		)
	}

	return &updatedLesson, nil
}

func (r *lessonRepo) UpdateOrder(
	ctx context.Context,
	lesson models.UpdateLessonOrder,
) error {
	query := `
		UPDATE lessons
		SET
			order_number = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		lesson.ID,
		lesson.OrderNumber,
	)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return apperror.Conflict(
				"repository",
				"UpdateLessonOrder",
				"lesson order number already exists in this module",
				apperror.ErrAlreadyExists,
			)
		}

		return apperror.Internal(
			"repository",
			"UpdateLessonOrder",
			"failed to update lesson order",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"UpdateLessonOrder",
			"lesson not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

func (r *lessonRepo) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE lessons
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
			"DeleteLesson",
			"failed to delete lesson",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteLesson",
			"lesson not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}

// DeleteByModuleID soft-deletes all lessons of a module, used when the module
// itself is deleted.
func (r *lessonRepo) DeleteByModuleID(
	ctx context.Context,
	moduleID uuid.UUID,
) error {
	query := `
		UPDATE lessons
		SET
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE module_id = $1
			AND deleted_at IS NULL;
	`

	_, err := r.db.Exec(
		ctx,
		query,
		moduleID,
	)

	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteLessonsByModuleID",
			"failed to delete lessons of module",
			err,
		)
	}

	return nil
}

func (r *lessonRepo) CreateMaterial(
	ctx context.Context,
	material models.CreateLessonMaterial,
) (uuid.UUID, error) {
	query := `
		INSERT INTO lesson_materials (
			id,
			lesson_id,
			type,
			content,
			object_key,
			file_name,
			mime_type,
			file_size
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id;
	`

	id := uuid.New()

	err := r.db.QueryRow(
		ctx,
		query,
		id,
		material.LessonID,
		material.Type,
		material.Content,
		material.ObjectKey,
		material.FileName,
		material.MimeType,
		material.FileSize,
	).Scan(&id)

	if err != nil {
		if pgerr.IsCheckViolation(err) {
			return uuid.Nil, apperror.InvalidInput(
				"repository",
				"CreateMaterial",
				"text and video need content, file needs object key",
				apperror.ErrInvalidInput,
			)
		}

		if pgerr.IsForeignKeyViolation(err) {
			return uuid.Nil, apperror.NotFound(
				"repository",
				"CreateMaterial",
				"lesson not found",
				apperror.ErrNotFound,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateMaterial",
			"failed to create material",
			err,
		)
	}

	return id, nil
}

func (r *lessonRepo) GetMaterialByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.LessonMaterial, error) {
	query := `
		SELECT
			id,
			lesson_id,
			type,
			content,
			object_key,
			file_name,
			mime_type,
			file_size,
			created_at,
			updated_at
		FROM lesson_materials
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	var material models.LessonMaterial

	err := scanMaterial(r.db.QueryRow(ctx, query, id), &material)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetMaterialByID",
				"material not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetMaterialByID",
			"failed to get material",
			err,
		)
	}

	return &material, nil
}

func (r *lessonRepo) GetMaterials(
	ctx context.Context,
	lessonID uuid.UUID,
) ([]models.LessonMaterial, error) {
	query := `
		SELECT
			id,
			lesson_id,
			type,
			content,
			object_key,
			file_name,
			mime_type,
			file_size,
			created_at,
			updated_at
		FROM lesson_materials
		WHERE lesson_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at;
	`

	rows, err := r.db.Query(ctx, query, lessonID)
	if err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetMaterials",
			"failed to get materials",
			err,
		)
	}
	defer rows.Close()

	materials := make([]models.LessonMaterial, 0)

	for rows.Next() {
		var material models.LessonMaterial

		if err := scanMaterial(rows, &material); err != nil {
			return nil, apperror.Internal(
				"repository",
				"GetMaterials",
				"failed to scan materials",
				err,
			)
		}

		materials = append(materials, material)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(
			"repository",
			"GetMaterials",
			"failed to read materials",
			err,
		)
	}

	return materials, nil
}

func (r *lessonRepo) UpdateMaterial(
	ctx context.Context,
	material models.UpdateLessonMaterial,
) (*models.LessonMaterial, error) {
	query := `
		UPDATE lesson_materials
		SET
			content = COALESCE($2, content),
			object_key = COALESCE($3, object_key),
			file_name = COALESCE($4, file_name),
			mime_type = COALESCE($5, mime_type),
			file_size = COALESCE($6, file_size),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND deleted_at IS NULL
		RETURNING
			id,
			lesson_id,
			type,
			content,
			object_key,
			file_name,
			mime_type,
			file_size,
			created_at,
			updated_at;
	`

	var updatedMaterial models.LessonMaterial

	err := scanMaterial(
		r.db.QueryRow(
			ctx,
			query,
			material.ID,
			material.Content,
			material.ObjectKey,
			material.FileName,
			material.MimeType,
			material.FileSize,
		),
		&updatedMaterial,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"UpdateMaterial",
				"material not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateMaterial",
			"failed to update material",
			err,
		)
	}

	return &updatedMaterial, nil
}

func (r *lessonRepo) DeleteMaterial(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE lesson_materials
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
			"DeleteMaterial",
			"failed to delete material",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteMaterial",
			"material not found",
			apperror.ErrNotFound,
		)
	}

	return nil
}
