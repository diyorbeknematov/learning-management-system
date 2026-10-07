package models

import (
	"time"

	"github.com/google/uuid"
)

type LessonProgress struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	StudentID   uuid.UUID  `db:"student_id" json:"student_id"`
	LessonID    uuid.UUID  `db:"lesson_id" json:"lesson_id"`
	Completed   bool       `db:"completed" json:"completed"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type SetLessonProgress struct {
	StudentID uuid.UUID `json:"-"`
	LessonID  uuid.UUID `json:"-"`
	Completed bool      `json:"completed"`
}

type CourseProgress struct {
	CourseID           uuid.UUID   `db:"course_id" json:"course_id"`
	TotalLessons       int         `db:"total_lessons" json:"total_lessons"`
	CompletedLessons   int         `db:"completed_lessons" json:"completed_lessons"`
	ProgressPercent    float64     `db:"progress_percent" json:"progress_percent"`
	CompletedLessonIDs []uuid.UUID `db:"-" json:"completed_lesson_ids"`
}

type StudentProgress struct {
	StudentID uuid.UUID `db:"student_id" json:"student_id"`
	FullName  string    `db:"full_name" json:"full_name"`
	Email     string    `db:"email" json:"email"`
	CourseProgress
	LastActivityAt *time.Time `db:"last_activity_at" json:"last_activity_at"`
}
