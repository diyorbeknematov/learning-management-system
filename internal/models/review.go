package models

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID          uuid.UUID `db:"id" json:"id"`
	StudentID   uuid.UUID `db:"student_id" json:"student_id"`
	CourseID    uuid.UUID `db:"course_id" json:"course_id"`
	StudentName string    `db:"student_name" json:"student_name"`
	Rating      int       `db:"rating" json:"rating"`
	Comment     *string   `db:"comment" json:"comment"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreateReview struct {
	StudentID uuid.UUID `db:"student_id" json:"-"`
	CourseID  uuid.UUID `db:"course_id" json:"-"`
	Rating    int       `db:"rating" json:"rating" validate:"required,min=1,max=5"`
	Comment   *string   `db:"comment" json:"comment"`
}

type UpdateReview struct {
	ID      uuid.UUID `db:"id" json:"-"`
	Rating  *int      `db:"rating" json:"rating" validate:"omitempty,min=1,max=5"`
	Comment *string   `db:"comment" json:"comment"`
}

type ReviewFilter struct {
	Limit int `form:"limit" json:"limit"`
	Page  int `form:"page" json:"page"`
}

type ReviewList struct {
	ListResponse[Review]
	AvgRating float64 `json:"avg_rating"`
}
