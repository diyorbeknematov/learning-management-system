package models

import (
	"time"

	"github.com/google/uuid"
)

type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatCSV  ReportFormat = "csv"
)

type ReportFilter struct {
	DateRange
	CourseID *uuid.UUID   `form:"course_id" json:"course_id"`
	GroupBy  string       `form:"group_by" json:"group_by"`
	Format   ReportFormat `form:"format" json:"format"`
}

type EnrollmentReportRow struct {
	CourseID          uuid.UUID `db:"course_id" json:"course_id" csv:"course_id"`
	CourseTitle       string    `db:"course_title" json:"course_title" csv:"course_title"`
	Total             int       `db:"total" json:"total" csv:"total"`
	Active            int       `db:"active" json:"active" csv:"active"`
	Completed         int       `db:"completed" json:"completed" csv:"completed"`
	Dropped           int       `db:"dropped" json:"dropped" csv:"dropped"`
	NewStudents       int       `db:"new_students" json:"new_students" csv:"new_students"`
	ReturningStudents int       `db:"returning_students" json:"returning_students" csv:"returning_students"`
	DropoutRate       float64   `db:"dropout_rate" json:"dropout_rate" csv:"dropout_rate"`
}

type TrendRow struct {
	Period time.Time `db:"period" json:"period" csv:"period"`
	Count  int       `db:"count" json:"count" csv:"count"`
}

type RevenueReportRow struct {
	CourseID           uuid.UUID `db:"course_id" json:"course_id" csv:"course_id"`
	CourseTitle        string    `db:"course_title" json:"course_title" csv:"course_title"`
	PaidEnrollments    int       `db:"paid_enrollments" json:"paid_enrollments" csv:"paid_enrollments"`
	FreeEnrollments    int       `db:"free_enrollments" json:"free_enrollments" csv:"free_enrollments"`
	Revenue            float64   `db:"revenue" json:"revenue" csv:"revenue"`
	AvgEnrollmentValue float64   `db:"avg_enrollment_value" json:"avg_enrollment_value" csv:"avg_enrollment_value"`
}

type StudentReportRow struct {
	StudentID        uuid.UUID  `db:"student_id" json:"student_id" csv:"student_id"`
	FullName         string     `db:"full_name" json:"full_name" csv:"full_name"`
	Email            string     `db:"email" json:"email" csv:"email"`
	RegisteredAt     time.Time  `db:"registered_at" json:"registered_at" csv:"registered_at"`
	EnrolledCourses  int        `db:"enrolled_courses" json:"enrolled_courses" csv:"enrolled_courses"`
	CompletedCourses int        `db:"completed_courses" json:"completed_courses" csv:"completed_courses"`
	LastActivityAt   *time.Time `db:"last_activity_at" json:"last_activity_at" csv:"last_activity_at"`
	Active           bool       `db:"active" json:"active" csv:"active"`
}

type ProgressReportRow struct {
	CourseID          uuid.UUID `db:"course_id" json:"course_id" csv:"course_id"`
	CourseTitle       string    `db:"course_title" json:"course_title" csv:"course_title"`
	Students          int       `db:"students" json:"students" csv:"students"`
	AvgCompletionRate float64   `db:"avg_completion_rate" json:"avg_completion_rate" csv:"avg_completion_rate"`
	InactiveStudents  int       `db:"inactive_students" json:"inactive_students" csv:"inactive_students"`
}

type ProgressFunnelRow struct {
	LessonID          uuid.UUID `db:"lesson_id" json:"lesson_id" csv:"lesson_id"`
	LessonTitle       string    `db:"lesson_title" json:"lesson_title" csv:"lesson_title"`
	ModuleOrder       int       `db:"module_order" json:"module_order" csv:"module_order"`
	LessonOrder       int       `db:"lesson_order" json:"lesson_order" csv:"lesson_order"`
	StudentsCompleted int       `db:"students_completed" json:"students_completed" csv:"students_completed"`
}

type QuizReportRow struct {
	QuizID             uuid.UUID `db:"quiz_id" json:"quiz_id" csv:"quiz_id"`
	QuizTitle          string    `db:"quiz_title" json:"quiz_title" csv:"quiz_title"`
	Attempts           int       `db:"attempts" json:"attempts" csv:"attempts"`
	AvgScore           float64   `db:"avg_score" json:"avg_score" csv:"avg_score"`
	PassRate           float64   `db:"pass_rate" json:"pass_rate" csv:"pass_rate"`
	FailRate           float64   `db:"fail_rate" json:"fail_rate" csv:"fail_rate"`
	AvgAttemptsPerUser float64   `db:"avg_attempts_per_user" json:"avg_attempts_per_user" csv:"avg_attempts_per_user"`
}

type MostFailedQuestionRow struct {
	QuestionID   uuid.UUID `db:"question_id" json:"question_id" csv:"question_id"`
	QuizTitle    string    `db:"quiz_title" json:"quiz_title" csv:"quiz_title"`
	QuestionText string    `db:"question_text" json:"question_text" csv:"question_text"`
	FailCount    int       `db:"fail_count" json:"fail_count" csv:"fail_count"`
}

type CertificateReportRow struct {
	CourseID    uuid.UUID `db:"course_id" json:"course_id" csv:"course_id"`
	CourseTitle string    `db:"course_title" json:"course_title" csv:"course_title"`
	Issued      int       `db:"issued" json:"issued" csv:"issued"`
}

type InstructorReportRow struct {
	InstructorID uuid.UUID `db:"instructor_id" json:"instructor_id" csv:"instructor_id"`
	FullName     string    `db:"full_name" json:"full_name" csv:"full_name"`
	Courses      int       `db:"courses" json:"courses" csv:"courses"`
	Students     int       `db:"students" json:"students" csv:"students"`
	Revenue      float64   `db:"revenue" json:"revenue" csv:"revenue"`
	AvgRating    float64   `db:"avg_rating" json:"avg_rating" csv:"avg_rating"`
}

type ReviewReportRow struct {
	CourseID       uuid.UUID `db:"course_id" json:"course_id" csv:"course_id"`
	CourseTitle    string    `db:"course_title" json:"course_title" csv:"course_title"`
	AvgRating      float64   `db:"avg_rating" json:"avg_rating" csv:"avg_rating"`
	ReviewCount    int       `db:"review_count" json:"review_count" csv:"review_count"`
	CompletedCount int       `db:"completed_count" json:"completed_count" csv:"completed_count"`
	ReviewGap      int       `db:"review_gap" json:"review_gap" csv:"review_gap"`
	LowRating      bool      `db:"low_rating" json:"low_rating" csv:"low_rating"`
}
