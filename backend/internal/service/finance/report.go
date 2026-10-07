package finance

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
)

// Report builds the reports of the platform for the SuperAdmin. Every report
// takes the same filter: a period, a course and, for the trends, how to group.
// The handler turns the main table (Rows) into JSON or CSV.
type Report struct {
	repo *repo.Repository
}

func NewReport(deps core.Dependencies) *Report {
	return &Report{
		repo: deps.Repo,
	}
}

// check lets the SuperAdmin through and checks the dates. Without dates a
// report covers all the time.
func check(actor models.Actor, filter models.ReportFilter, op string) error {
	if err := core.RequireSuperAdmin(actor, op); err != nil {
		return err
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return apperror.InvalidInput("service", op, "from must not be after to", apperror.ErrInvalidInput)
	}

	return nil
}

func (s *Report) Enrollments(ctx context.Context, actor models.Actor, filter models.ReportFilter) (*models.EnrollmentReport, error) {
	if err := check(actor, filter, "EnrollmentReport"); err != nil {
		return nil, err
	}

	rows, err := s.repo.Report.Enrollments(ctx, filter)
	if err != nil {
		return nil, err
	}

	trend, err := s.repo.Report.EnrollmentTrend(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &models.EnrollmentReport{Rows: rows, Trend: trend}, nil
}

func (s *Report) Revenue(ctx context.Context, actor models.Actor, filter models.ReportFilter) ([]models.RevenueReportRow, error) {
	if err := check(actor, filter, "RevenueReport"); err != nil {
		return nil, err
	}

	return s.repo.Report.Revenue(ctx, filter)
}

func (s *Report) Students(ctx context.Context, actor models.Actor, filter models.ReportFilter) (*models.StudentReport, error) {
	if err := check(actor, filter, "StudentReport"); err != nil {
		return nil, err
	}

	rows, err := s.repo.Report.Students(ctx, filter)
	if err != nil {
		return nil, err
	}

	growth, err := s.repo.Report.StudentGrowth(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &models.StudentReport{Rows: rows, Growth: growth}, nil
}

// Progress adds the funnel (how many students did each lesson) when the report
// is for one course.
func (s *Report) Progress(ctx context.Context, actor models.Actor, filter models.ReportFilter) (*models.ProgressReport, error) {
	if err := check(actor, filter, "ProgressReport"); err != nil {
		return nil, err
	}

	rows, err := s.repo.Report.Progress(ctx, filter)
	if err != nil {
		return nil, err
	}

	report := &models.ProgressReport{Rows: rows, Funnel: []models.ProgressFunnelRow{}}

	if filter.CourseID != nil {
		report.Funnel, err = s.repo.Report.ProgressFunnel(ctx, *filter.CourseID)
		if err != nil {
			return nil, err
		}
	}

	return report, nil
}

func (s *Report) Quizzes(ctx context.Context, actor models.Actor, filter models.ReportFilter) (*models.QuizReport, error) {
	if err := check(actor, filter, "QuizReport"); err != nil {
		return nil, err
	}

	rows, err := s.repo.Report.Quizzes(ctx, filter)
	if err != nil {
		return nil, err
	}

	failed, err := s.repo.Report.MostFailedQuestions(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &models.QuizReport{Rows: rows, MostFailedQuestions: failed}, nil
}

func (s *Report) Certificates(ctx context.Context, actor models.Actor, filter models.ReportFilter) (*models.CertificateReport, error) {
	if err := check(actor, filter, "CertificateReport"); err != nil {
		return nil, err
	}

	rows, err := s.repo.Report.Certificates(ctx, filter)
	if err != nil {
		return nil, err
	}

	trend, err := s.repo.Report.CertificateTrend(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &models.CertificateReport{Rows: rows, Trend: trend}, nil
}

func (s *Report) Instructors(ctx context.Context, actor models.Actor, filter models.ReportFilter) ([]models.InstructorReportRow, error) {
	if err := check(actor, filter, "InstructorReport"); err != nil {
		return nil, err
	}

	return s.repo.Report.Instructors(ctx, filter)
}

func (s *Report) Reviews(ctx context.Context, actor models.Actor, filter models.ReportFilter) ([]models.ReviewReportRow, error) {
	if err := check(actor, filter, "ReviewReport"); err != nil {
		return nil, err
	}

	return s.repo.Report.Reviews(ctx, filter)
}
