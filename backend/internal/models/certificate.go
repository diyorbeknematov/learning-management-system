package models

import (
	"time"

	"github.com/google/uuid"
)

type Certificate struct {
	ID             uuid.UUID `db:"id" json:"id"`
	StudentID      uuid.UUID `db:"student_id" json:"student_id"`
	CourseID       uuid.UUID `db:"course_id" json:"course_id"`
	CompletionDate time.Time `db:"completion_date" json:"completion_date"`
	UniqueID       string    `db:"unique_id" json:"unique_id"`
	QRCode         *string   `db:"qr_code" json:"qr_code"`
	ObjectKey      *string   `db:"object_key" json:"object_key"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type CertificateDetail struct {
	Certificate
	StudentName    string `db:"student_name" json:"student_name"`
	CourseTitle    string `db:"course_title" json:"course_title"`
	InstructorName string `db:"instructor_name" json:"instructor_name"`
}

type CreateCertificate struct {
	StudentID      uuid.UUID `db:"student_id"`
	CourseID       uuid.UUID `db:"course_id"`
	CompletionDate time.Time `db:"completion_date"`
	UniqueID       string    `db:"unique_id"`
	QRCode         *string   `db:"qr_code"`
	ObjectKey      *string   `db:"object_key"`
}

type CertificateVerification struct {
	Valid          bool       `json:"valid"`
	UniqueID       string     `json:"unique_id"`
	StudentName    string     `json:"student_name,omitempty"`
	CourseTitle    string     `json:"course_title,omitempty"`
	InstructorName string     `json:"instructor_name,omitempty"`
	CompletionDate *time.Time `json:"completion_date,omitempty"`
}
