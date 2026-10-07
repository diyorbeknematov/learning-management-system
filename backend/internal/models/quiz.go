package models

import (
	"time"

	"github.com/google/uuid"
)

type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "single_choice"
	QuestionTypeMultipleChoice QuestionType = "multiple_choice"
	QuestionTypeTrueFalse      QuestionType = "true_false"
)

type Quiz struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	CourseID      *uuid.UUID `db:"course_id" json:"course_id"`
	ModuleID      *uuid.UUID `db:"module_id" json:"module_id"`
	Title         string     `db:"title" json:"title"`
	Description   *string    `db:"description" json:"description"`
	TimeLimit     int        `db:"time_limit" json:"time_limit"`
	PassThreshold int        `db:"pass_threshold" json:"pass_threshold"`
	MaxAttempts   int        `db:"max_attempts" json:"max_attempts"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type CreateQuiz struct {
	CourseID      *uuid.UUID `db:"course_id" json:"-"`
	ModuleID      *uuid.UUID `db:"module_id" json:"-"`
	Title         string     `db:"title" json:"title" validate:"required"`
	Description   *string    `db:"description" json:"description"`
	TimeLimit     int        `db:"time_limit" json:"time_limit" validate:"required,gt=0"`
	PassThreshold int        `db:"pass_threshold" json:"pass_threshold" validate:"gte=0,lte=100"`
	MaxAttempts   int        `db:"max_attempts" json:"max_attempts" validate:"required,gt=0"`
}

type UpdateQuiz struct {
	ID            uuid.UUID `db:"id" json:"-"`
	Title         *string   `db:"title" json:"title" validate:"omitempty,min=1"`
	Description   *string   `db:"description" json:"description"`
	TimeLimit     *int      `db:"time_limit" json:"time_limit" validate:"omitempty,gt=0"`
	PassThreshold *int      `db:"pass_threshold" json:"pass_threshold" validate:"omitempty,gte=0,lte=100"`
	MaxAttempts   *int      `db:"max_attempts" json:"max_attempts" validate:"omitempty,gt=0"`
}

// IsCorrect is nil for students so the answer key is never exposed.
type QuestionOption struct {
	ID         uuid.UUID `db:"id" json:"id"`
	QuestionID uuid.UUID `db:"question_id" json:"question_id"`
	OptionText string    `db:"option_text" json:"option_text"`
	IsCorrect  *bool     `db:"is_correct" json:"is_correct,omitempty"`
}

type Question struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	QuizID      uuid.UUID        `db:"quiz_id" json:"quiz_id"`
	Text        string           `db:"text" json:"text"`
	Type        QuestionType     `db:"type" json:"type"`
	OrderNumber int              `db:"order_number" json:"order_number"`
	Options     []QuestionOption `db:"-" json:"options"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
}

type CreateQuestionOption struct {
	QuestionID uuid.UUID `json:"-"`
	Position   int       `json:"-"`
	OptionText string    `json:"option_text" validate:"required"`
	IsCorrect  bool      `json:"is_correct"`
}

type CreateQuestion struct {
	QuizID      uuid.UUID              `db:"quiz_id" json:"-"`
	Text        string                 `db:"text" json:"text" validate:"required"`
	Type        QuestionType           `db:"type" json:"type" validate:"required,oneof=single_choice multiple_choice true_false"`
	OrderNumber int                    `db:"order_number" json:"order_number" validate:"required,min=1"`
	Options     []CreateQuestionOption `db:"-" json:"options" validate:"required,min=2,dive"`
}

type UpdateQuestion struct {
	ID          uuid.UUID               `db:"id" json:"-"`
	Text        *string                 `db:"text" json:"text" validate:"omitempty,min=1"`
	Type        *QuestionType           `db:"type" json:"type" validate:"omitempty,oneof=single_choice multiple_choice true_false"`
	OrderNumber *int                    `db:"order_number" json:"order_number" validate:"omitempty,min=1"`
	Options     *[]CreateQuestionOption `db:"-" json:"options" validate:"omitempty,min=2,dive"`
}

// CreateQuestionRequest adds a question to a quiz. Without OrderNumber the
// question goes to the end; with it, the question takes that place and the
// following ones move down. The order matters only to the author: a student
// sees the questions in a random order.
type CreateQuestionRequest struct {
	Text        string                 `json:"text" validate:"required"`
	Type        QuestionType           `json:"type" validate:"required,oneof=single_choice multiple_choice true_false"`
	OrderNumber *int                   `json:"order_number" validate:"omitempty,min=1"`
	Options     []CreateQuestionOption `json:"options" validate:"required,min=2,dive"`
}
