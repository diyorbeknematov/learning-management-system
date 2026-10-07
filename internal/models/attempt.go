package models

import (
	"time"

	"github.com/google/uuid"
)

type QuizAttempt struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	StudentID     uuid.UUID  `db:"student_id" json:"student_id"`
	QuizID        uuid.UUID  `db:"quiz_id" json:"quiz_id"`
	AttemptNumber int        `db:"attempt_number" json:"attempt_number"`
	Score         *int       `db:"score" json:"score"`
	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	CompletedAt   *time.Time `db:"completed_at" json:"completed_at"`
	TimeSpent     *int       `db:"time_spent" json:"time_spent"`
	Passed        *bool      `db:"-" json:"passed"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

type CreateQuizAttempt struct {
	StudentID     uuid.UUID `db:"student_id" json:"-"`
	QuizID        uuid.UUID `db:"quiz_id" json:"-"`
	AttemptNumber int       `db:"attempt_number" json:"-"`
	StartedAt     time.Time `db:"started_at" json:"-"`
}

type AttemptAnswer struct {
	ID         uuid.UUID `db:"id" json:"id"`
	AttemptID  uuid.UUID `db:"attempt_id" json:"attempt_id"`
	QuestionID uuid.UUID `db:"question_id" json:"question_id"`
	OptionID   uuid.UUID `db:"option_id" json:"option_id"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type SubmitAnswer struct {
	QuestionID uuid.UUID   `json:"question_id" validate:"required"`
	OptionIDs  []uuid.UUID `json:"option_ids" validate:"required,min=1"`
}

type SubmitAttempt struct {
	AttemptID uuid.UUID      `json:"-"`
	Answers   []SubmitAnswer `json:"answers" validate:"required,dive"`
}

type CompleteAttempt struct {
	ID          uuid.UUID `db:"id"`
	Score       int       `db:"score"`
	CompletedAt time.Time `db:"completed_at"`
	TimeSpent   int       `db:"time_spent"`
}

type AttemptDetail struct {
	QuizAttempt
	Answers []AttemptAnswer `json:"answers"`
}
