package models

import (
	"time"

	"github.com/google/uuid"
)

type Module struct {
	ID          uuid.UUID `db:"id" json:"id"`
	CourseID    uuid.UUID `db:"course_id" json:"course_id"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description"`
	OrderNumber int       `db:"order_number" json:"order_number"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type ModuleDetail struct {
	Module
	Lessons []Lesson `json:"lessons"`
}

type CreateModule struct {
	CourseID    uuid.UUID `db:"course_id" json:"-"`
	Title       string    `db:"title" json:"title" validate:"required"`
	Description *string   `db:"description" json:"description"`
	OrderNumber int       `db:"order_number" json:"order_number" validate:"required,min=1"`
}

type UpdateModule struct {
	ID          uuid.UUID `db:"id" json:"-"`
	Title       *string   `db:"title" json:"title"`
	Description *string   `db:"description" json:"description"`
}

type UpdateModuleOrder struct {
	ID          uuid.UUID `json:"-"`
	OrderNumber int       `json:"order_number" validate:"required,min=1"`
}

// CreateModuleRequest adds a module to a course. Without OrderNumber the module
// goes to the end; with it, the module takes that place and the following
// modules move down.
type CreateModuleRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description"`
	OrderNumber *int    `json:"order_number" validate:"omitempty,min=1"`
}
