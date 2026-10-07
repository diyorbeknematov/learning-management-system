package handler

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/finance"
	"github.com/gin-gonic/gin"
)

// Report has the eight reports of the SuperAdmin. Each one answers JSON, or a
// CSV file when the query has format=csv; the file holds the main table of the
// report.
type Report struct {
	svc *finance.Report
}

func NewReport(svc *finance.Report) *Report {
	return &Report{
		svc: svc,
	}
}

func (h *Report) Private(group *gin.RouterGroup) {
	reports := group.Group("/reports")

	reports.GET("/enrollments", h.enrollments)
	reports.GET("/revenue", h.revenue)
	reports.GET("/students", h.students)
	reports.GET("/progress", h.progress)
	reports.GET("/quizzes", h.quizzes)
	reports.GET("/certificates", h.certificates)
	reports.GET("/instructors", h.instructors)
	reports.GET("/reviews", h.reviews)
}

type reportQuery struct {
	periodQuery
	CourseID *string `form:"course_id" validate:"omitempty,uuid"`
	GroupBy  string  `form:"group_by" validate:"omitempty,oneof=day week month"`
	Format   string  `form:"format" validate:"omitempty,oneof=json csv"`
}

// run reads the query, builds the report with build, and writes it. The main
// table of the report is what the CSV file holds.
func run[T any](c *gin.Context, name string, build func(context.Context, models.Actor, models.ReportFilter) (T, error), table func(T) any) {
	var query reportQuery

	if !bindQuery(c, &query) {
		return
	}

	dates, ok := period(c, query.periodQuery)
	if !ok {
		return
	}

	report, err := build(c.Request.Context(), middleware.ActorFrom(c), models.ReportFilter{
		DateRange: dates,
		CourseID:  optionalID(query.CourseID),
		GroupBy:   query.GroupBy,
	})
	if err != nil {
		response.Fail(c, err)

		return
	}

	if query.Format == "csv" {
		writeCSV(c, name, table(report))

		return
	}

	response.OK(c, report)
}

// @Summary Enrollments report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.EnrollmentReport}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/enrollments [get]
func (h *Report) enrollments(c *gin.Context) {
	run(c, "enrollments", h.svc.Enrollments, func(r *models.EnrollmentReport) any { return r.Rows })
}

// @Summary Revenue report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=[]models.RevenueReportRow}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/revenue [get]
func (h *Report) revenue(c *gin.Context) {
	run(c, "revenue", h.svc.Revenue, func(rows []models.RevenueReportRow) any { return rows })
}

// @Summary Students report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.StudentReport}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/students [get]
func (h *Report) students(c *gin.Context) {
	run(c, "students", h.svc.Students, func(r *models.StudentReport) any { return r.Rows })
}

// @Summary Progress report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.ProgressReport}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/progress [get]
func (h *Report) progress(c *gin.Context) {
	run(c, "progress", h.svc.Progress, func(r *models.ProgressReport) any { return r.Rows })
}

// @Summary Quizzes report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.QuizReport}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/quizzes [get]
func (h *Report) quizzes(c *gin.Context) {
	run(c, "quizzes", h.svc.Quizzes, func(r *models.QuizReport) any { return r.Rows })
}

// @Summary Certificates report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=models.CertificateReport}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/certificates [get]
func (h *Report) certificates(c *gin.Context) {
	run(c, "certificates", h.svc.Certificates, func(r *models.CertificateReport) any { return r.Rows })
}

// @Summary Instructors report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=[]models.InstructorReportRow}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/instructors [get]
func (h *Report) instructors(c *gin.Context) {
	run(c, "instructors", h.svc.Instructors, func(rows []models.InstructorReportRow) any { return rows })
}

// @Summary Reviews report (format=csv downloads the main table as a CSV file)
// @Description Access: SuperAdmin.
// @Tags Reports
// @Produce json,text/csv
// @Security BearerAuth
// @Param query query reportQuery false "Query parameters"
// @Success 200 {object} response.Success{data=[]models.ReviewReportRow}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /reports/reviews [get]
func (h *Report) reviews(c *gin.Context) {
	run(c, "reviews", h.svc.Reviews, func(rows []models.ReviewReportRow) any { return rows })
}
