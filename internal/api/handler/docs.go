package handler

import "github.com/diyorbeknematov/lms/internal/models"

// The types below exist only for the API documentation: the generator cannot
// read models.ListResponse[T], so each list that the API returns has its own
// plain type with the same JSON.

// CategoryList is models.ListResponse[models.Category].
type CategoryList struct {
	Items []models.Category `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// CourseListItemList is models.ListResponse[models.CourseListItem].
type CourseListItemList struct {
	Items []models.CourseListItem `json:"items"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Limit int                     `json:"limit"`
}

// EnrollmentRosterItemList is models.ListResponse[models.EnrollmentRosterItem].
type EnrollmentRosterItemList struct {
	Items []models.EnrollmentRosterItem `json:"items"`
	Total int                           `json:"total"`
	Page  int                           `json:"page"`
	Limit int                           `json:"limit"`
}

// MyEnrollmentList is models.ListResponse[models.MyEnrollment].
type MyEnrollmentList struct {
	Items []models.MyEnrollment `json:"items"`
	Total int                   `json:"total"`
	Page  int                   `json:"page"`
	Limit int                   `json:"limit"`
}

// PaymentList is models.ListResponse[models.Payment].
type PaymentList struct {
	Items []models.Payment `json:"items"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Limit int              `json:"limit"`
}

// StudentProgressList is models.ListResponse[models.StudentProgress].
type StudentProgressList struct {
	Items []models.StudentProgress `json:"items"`
	Total int                      `json:"total"`
	Page  int                      `json:"page"`
	Limit int                      `json:"limit"`
}

// UserList is models.ListResponse[models.User].
type UserList struct {
	Items []models.User `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
}
