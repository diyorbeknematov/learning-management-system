package models

import (
	"time"

	"github.com/google/uuid"
)

type MaterialType string

const (
	MaterialTypeText  MaterialType = "text"
	MaterialTypeVideo MaterialType = "video"
	MaterialTypeFile  MaterialType = "file"
)

type Lesson struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ModuleID    uuid.UUID `db:"module_id" json:"module_id"`
	Title       string    `db:"title" json:"title"`
	Duration    *int      `db:"duration" json:"duration"`
	OrderNumber int       `db:"order_number" json:"order_number"`
	IsPreview   bool      `db:"is_preview" json:"is_preview"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreateLesson struct {
	ModuleID    uuid.UUID `db:"module_id" json:"-"`
	Title       string    `db:"title" json:"title" validate:"required"`
	Duration    *int      `db:"duration" json:"duration"`
	OrderNumber int       `db:"order_number" json:"order_number" validate:"required,min=1"`
	IsPreview   bool      `db:"is_preview" json:"is_preview"`
}

type UpdateLesson struct {
	ID        uuid.UUID `db:"id" json:"-"`
	Title     *string   `db:"title" json:"title"`
	Duration  *int      `db:"duration" json:"duration"`
	IsPreview *bool     `db:"is_preview" json:"is_preview"`
}

type UpdateLessonOrder struct {
	ID          uuid.UUID `json:"-"`
	OrderNumber int       `json:"order_number" validate:"required,min=1"`
}

type LessonMaterial struct {
	ID        uuid.UUID    `db:"id" json:"id"`
	LessonID  uuid.UUID    `db:"lesson_id" json:"lesson_id"`
	Type      MaterialType `db:"type" json:"type"`
	Content   *string      `db:"content" json:"content"`
	ObjectKey *string      `db:"object_key" json:"object_key"`
	FileName  *string      `db:"file_name" json:"file_name"`
	MimeType  *string      `db:"mime_type" json:"mime_type"`
	FileSize  *int64       `db:"file_size" json:"file_size"`
	FileURL   *string      `db:"-" json:"file_url"`
	CreatedAt time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt time.Time    `db:"updated_at" json:"updated_at"`
}

type CreateLessonMaterial struct {
	LessonID  uuid.UUID    `db:"lesson_id" json:"-"`
	Type      MaterialType `db:"type" json:"type" validate:"required,oneof=text video file"`
	Content   *string      `db:"content" json:"content"`
	ObjectKey *string      `db:"object_key" json:"object_key"`
	FileName  *string      `db:"file_name" json:"file_name"`
	MimeType  *string      `db:"mime_type" json:"mime_type"`
	FileSize  *int64       `db:"file_size" json:"file_size"`
}

type UpdateLessonMaterial struct {
	ID        uuid.UUID `db:"id" json:"-"`
	Content   *string   `db:"content" json:"content"`
	ObjectKey *string   `db:"object_key" json:"object_key"`
	FileName  *string   `db:"file_name" json:"file_name"`
	MimeType  *string   `db:"mime_type" json:"mime_type"`
	FileSize  *int64    `db:"file_size" json:"file_size"`
}

// CreateLessonRequest adds a lesson to a module, with the same OrderNumber
// rules as CreateModuleRequest.
type CreateLessonRequest struct {
	Title       string `json:"title" validate:"required"`
	Duration    *int   `json:"duration" validate:"omitempty,min=0"`
	OrderNumber *int   `json:"order_number" validate:"omitempty,min=1"`
	IsPreview   bool   `json:"is_preview"`
}
