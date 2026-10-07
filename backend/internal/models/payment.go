package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const PaymentStatusPaid PaymentStatus = "paid"

type Payment struct {
	ID           uuid.UUID     `db:"id" json:"id"`
	EnrollmentID uuid.UUID     `db:"enrollment_id" json:"enrollment_id"`
	Amount       float64       `db:"amount" json:"amount"`
	Status       PaymentStatus `db:"status" json:"status"`
	PaidAt       *time.Time    `db:"paid_at" json:"paid_at"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at" json:"updated_at"`
}

type CreatePayment struct {
	EnrollmentID uuid.UUID `db:"enrollment_id"`
	Amount       float64   `db:"amount"`
	PaidAt       time.Time `db:"paid_at"`
}

type PaymentFilter struct {
	DateRange
	CourseID *uuid.UUID `form:"course_id" json:"course_id"`
	Limit    int        `form:"limit" json:"limit"`
	Page     int        `form:"page" json:"page"`
}

type InstructorPayout struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	InstructorID uuid.UUID  `db:"instructor_id" json:"instructor_id"`
	CourseID     uuid.UUID  `db:"course_id" json:"course_id"`
	EnrollmentID uuid.UUID  `db:"enrollment_id" json:"enrollment_id"`
	Type         PayoutType `db:"type" json:"type"`
	Value        float64    `db:"value" json:"value"`
	Amount       float64    `db:"amount" json:"amount"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

type CreateInstructorPayout struct {
	InstructorID uuid.UUID  `db:"instructor_id"`
	CourseID     uuid.UUID  `db:"course_id"`
	EnrollmentID uuid.UUID  `db:"enrollment_id"`
	Type         PayoutType `db:"type"`
	Value        float64    `db:"value"`
	Amount       float64    `db:"amount"`
}
