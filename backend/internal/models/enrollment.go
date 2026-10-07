package models

import (
	"time"

	"github.com/google/uuid"
)

type EnrollmentStatus string

const (
	EnrollmentStatusActive    EnrollmentStatus = "active"
	EnrollmentStatusCompleted EnrollmentStatus = "completed"
	EnrollmentStatusDropped   EnrollmentStatus = "dropped"
)

type Enrollment struct {
	ID        uuid.UUID        `db:"id" json:"id"`
	CourseID  uuid.UUID        `db:"course_id" json:"course_id"`
	StudentID uuid.UUID        `db:"student_id" json:"student_id"`
	Status    EnrollmentStatus `db:"status" json:"status"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

type CreateEnrollment struct {
	CourseID  uuid.UUID `db:"course_id" json:"-"`
	StudentID uuid.UUID `db:"student_id" json:"-"`
}

type UpdateEnrollmentStatus struct {
	ID     uuid.UUID        `json:"-"`
	Status EnrollmentStatus `json:"status" validate:"required,oneof=active completed dropped"`
}

type EnrollmentFilter struct {
	Status *EnrollmentStatus `form:"status" json:"status" validate:"omitempty,oneof=active completed dropped"`
	Limit  int               `form:"limit" json:"limit"`
	Page   int               `form:"page" json:"page"`
}

type EnrollmentRosterItem struct {
	Enrollment
	StudentName string `db:"student_name" json:"student_name"`
	Email       string `db:"email" json:"email"`
}

type MyEnrollment struct {
	Enrollment
	CourseTitle      string  `db:"course_title" json:"course_title"`
	CourseCover      *string `db:"course_cover" json:"course_cover"`
	CourseCoverURL   *string `db:"-" json:"course_cover_url"`
	TotalLessons     int     `db:"total_lessons" json:"total_lessons"`
	CompletedLessons int     `db:"completed_lessons" json:"completed_lessons"`
	ProgressPercent  float64 `db:"progress_percent" json:"progress_percent"`
}
