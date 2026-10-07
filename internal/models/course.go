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
	CoverURL         *string                 `db:"-" json:"cover_url"`
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
	Status CourseStatus `db:"status" json:"status" validate:"required,oneof=draft published"`
}

type CourseFilter struct {
	Search       *string       `form:"q" json:"q"`
	CategoryID   *uuid.UUID    `form:"category_id" json:"category_id"`
	InstructorID *uuid.UUID    `form:"instructor_id" json:"instructor_id"`
	Difficulty   *Difficulty   `form:"difficulty" json:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	Language     *string       `form:"language" json:"language"`
	MinRating    *float64      `form:"min_rating" json:"min_rating" validate:"omitempty,gte=0,lte=5"`
	PriceType    *string       `form:"price_type" json:"price_type" validate:"omitempty,oneof=free paid"`
	Status       *CourseStatus `form:"status" json:"status" validate:"omitempty,oneof=draft published"`
	Sort         string        `form:"sort" json:"sort" validate:"omitempty,oneof=popular rating newest price"`
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
	AvatarURL *string   `db:"-" json:"avatar_url"`
	Bio       *string   `db:"bio" json:"bio"`
	AvgRating float64   `db:"avg_rating" json:"avg_rating"`
}

type CourseDetail struct {
	CourseListItem
	Instructor CourseInstructor `json:"instructor"`
	Modules    []ModuleDetail   `json:"modules"`
}

// CreateCourseRequest is the body of a new course. InstructorID, PayoutType
// and PayoutValue are used only when a SuperAdmin sends them.
type CreateCourseRequest struct {
	InstructorID     *uuid.UUID  `json:"instructor_id"`
	CategoryID       uuid.UUID   `json:"category_id" validate:"required"`
	Title            string      `json:"title" validate:"required"`
	Cover            *string     `json:"cover"`
	Description      *string     `json:"description"`
	Difficulty       *Difficulty `json:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	TotalDuration    *int        `json:"total_duration" validate:"omitempty,min=0"`
	Language         *string     `json:"language"`
	Price            float64     `json:"price" validate:"min=0"`
	PayoutType       *PayoutType `json:"payout_type" validate:"omitempty,oneof=percentage fixed"`
	PayoutValue      *float64    `json:"payout_value" validate:"omitempty,min=0"`
	LearningOutcomes []string    `json:"learning_outcomes"`
	Requirements     []string    `json:"requirements"`
}

// UpdateCourseRequest changes only the fields that are sent. A sent list of
// learning outcomes or requirements replaces the whole old list.
type UpdateCourseRequest struct {
	CategoryID       *uuid.UUID  `json:"category_id"`
	Title            *string     `json:"title" validate:"omitempty,min=1"`
	Cover            *string     `json:"cover"`
	Description      *string     `json:"description"`
	Difficulty       *Difficulty `json:"difficulty" validate:"omitempty,oneof=beginner intermediate advanced"`
	TotalDuration    *int        `json:"total_duration" validate:"omitempty,min=0"`
	Language         *string     `json:"language"`
	Price            *float64    `json:"price" validate:"omitempty,min=0"`
	PayoutType       *PayoutType `json:"payout_type" validate:"omitempty,oneof=percentage fixed"`
	PayoutValue      *float64    `json:"payout_value" validate:"omitempty,min=0"`
	LearningOutcomes *[]string   `json:"learning_outcomes"`
	Requirements     *[]string   `json:"requirements"`
}
