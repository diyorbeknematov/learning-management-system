package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreateCategory struct {
	Name        string  `db:"name" json:"name" validate:"required,max=100"`
	Description *string `db:"description" json:"description"`
}

type UpdateCategory struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        *string   `db:"name" json:"name" validate:"omitempty,min=1,max=100"`
	Description *string   `db:"description" json:"description"`
}

type CategoryFilter struct {
	Name  *string `form:"name" json:"name"`
	Limit int     `form:"limit" json:"limit"`
	Page  int     `form:"page" json:"page"`
}
