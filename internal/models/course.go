package models

import (
	"time"

	"github.com/google/uuid"
)

type CourseStatus string

const (
	CourseStatusDraft     CourseStatus = "draft"
	CourseStatusPublished CourseStatus = "published"
)

type Difficulty string

const (
	DifficultyBeginner     Difficulty = "beginner"
	DifficultyIntermediate Difficulty = "intermediate"
	DifficultyAdvanced     Difficulty = "advanced"
)

type PayoutType string

const (
	PayoutTypePercentage PayoutType = "percentage"
	PayoutTypeFixed      PayoutType = "fixed"
)

type CourseRequirement struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CourseID  uuid.UUID `db:"course_id" json:"course_id"`
	Content   string    `db:"content" json:"content"`
	Position  int       `db:"position" json:"position"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CourseLearningOutcome struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CourseID  uuid.UUID `db:"course_id" json:"course_id"`
	Content   string    `db:"content" json:"content"`
	Position  int       `db:"position" json:"position"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Course struct {
	ID               uuid.UUID               `db:"id" json:"id"`
	InstructorID     uuid.UUID               `db:"instructor_id" json:"instructor_id"`
	CategoryID       uuid.UUID               `db:"category_id" json:"category_id"`
	Title            string                  `db:"title" json:"title"`
	Cover            *string                 `db:"cover" json:"cover"`
	Description      *string                 `db:"description" json:"description"`
	Difficulty       *Difficulty             `db:"difficulty" json:"difficulty"`
	TotalDuration    *int                    `db:"total_duration" json:"total_duration"`
	Language         *string                 `db:"language" json:"language"`
	Status           CourseStatus            `db:"status" json:"status"`
	Price            float64                 `db:"price" json:"price"`
	PayoutType       *PayoutType             `db:"payout_type" json:"payout_type"`
	PayoutValue      *float64                `db:"payout_value" json:"payout_value"`
	LearningOutcomes []CourseLearningOutcome `db:"-" json:"learning_outcomes"`
	Requirements     []CourseRequirement     `db:"-" json:"requirements"`
	CreatedAt        time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time               `db:"updated_at" json:"updated_at"`
	DeletedAt        *time.Time              `db:"deleted_at" json:"deleted_at"`
}

type CreateCourse struct {
	InstructorID  uuid.UUID   `db:"instructor_id" json:"instructor_id" validate:"required"`
	CategoryID    uuid.UUID   `db:"category_id" json:"category_id" validate:"required"`
	Title         string      `db:"title" json:"title" validate:"required"`
	Cover         *string     `db:"cover" json:"cover"`
	Description   *string     `db:"description" json:"description"`
	Difficulty    *Difficulty `db:"difficulty" json:"difficulty"`
	TotalDuration *int        `db:"total_duration" json:"total_duration"`
	Language      *string     `db:"language" json:"language"`
	Price         float64     `db:"price" json:"price"`
	PayoutType    *PayoutType `db:"payout_type" json:"payout_type"`
	PayoutValue   *float64    `db:"payout_value" json:"payout_value"`

	LearningOutcomes []string `db:"-" json:"learning_outcomes"`
	Requirements     []string `db:"-" json:"requirements"`
}

type UpdateCourse struct {
	ID            uuid.UUID   `db:"id" json:"id"`
	CategoryID    *uuid.UUID  `db:"category_id" json:"category_id"`
	Title         *string     `db:"title" json:"title"`
	Cover         *string     `db:"cover" json:"cover"`
	Description   *string     `db:"description" json:"description"`
	Difficulty    *Difficulty `db:"difficulty" json:"difficulty"`
	TotalDuration *int        `db:"total_duration" json:"total_duration"`
	Language      *string     `db:"language" json:"language"`
	Price         *float64    `db:"price" json:"price"`
	PayoutType    *PayoutType `db:"payout_type" json:"payout_type"`
	PayoutValue   *float64    `db:"payout_value" json:"payout_value"`

	LearningOutcomes *[]string `db:"-" json:"learning_outcomes"`
	Requirements     *[]string `db:"-" json:"requirements"`
}

type UpdateCourseStatus struct {
	ID     uuid.UUID    `db:"id" json:"id"`
	Status CourseStatus `db:"status" json:"status" validate:"required"`
}

type CourseFilter struct {
	Search       *string       `form:"q" json:"q"`
	CategoryID   *uuid.UUID    `form:"category_id" json:"category_id"`
	InstructorID *uuid.UUID    `form:"instructor_id" json:"instructor_id"`
	Difficulty   *Difficulty   `form:"difficulty" json:"difficulty"`
	Language     *string       `form:"language" json:"language"`
	MinRating    *float64      `form:"min_rating" json:"min_rating"`
	PriceType    *string       `form:"price_type" json:"price_type"`
	Status       *CourseStatus `form:"status" json:"status"`
	Sort         string        `form:"sort" json:"sort"`
	Limit        int           `form:"limit" json:"limit"`
	Page         int           `form:"page" json:"page"`
}

type CourseListItem struct {
	Course
	CategoryName    string  `db:"category_name" json:"category_name"`
	InstructorName  string  `db:"instructor_name" json:"instructor_name"`
	AvgRating       float64 `db:"avg_rating" json:"avg_rating"`
	ReviewCount     int     `db:"review_count" json:"review_count"`
	LessonCount     int     `db:"lesson_count" json:"lesson_count"`
	EnrollmentCount int     `db:"enrollment_count" json:"enrollment_count"`
}

type CourseInstructor struct {
	ID        uuid.UUID `db:"id" json:"id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Avatar    *string   `db:"avatar" json:"avatar"`
	Bio       *string   `db:"bio" json:"bio"`
	AvgRating float64   `db:"avg_rating" json:"avg_rating"`
}

type CourseDetail struct {
	CourseListItem
	Instructor CourseInstructor `json:"instructor"`
	Modules    []ModuleDetail   `json:"modules"`
}
