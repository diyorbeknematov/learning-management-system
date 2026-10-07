package repo

import (
	"context"
	"fmt"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool

	User         User
	RefreshToken RefreshToken
	Category     Category
	Course       Course
	Module       Module
	Lesson       Lesson
	Quiz         Quiz
	Question     Question
	Attempt      Attempt
	Enrollment   Enrollment
	Progress     Progress
	Payment      Payment
	Certificate  Certificate
	Review       Review
	Finance      Finance
	Report       Report
}

// newRepository is the only place where repositories are listed. q is either
// the pool or a transaction.
func newRepository(db *pgxpool.Pool, q postgres.DBTX) *Repository {
	return &Repository{
		db:           db,
		User:         postgres.NewUserRepository(q),
		RefreshToken: postgres.NewRefreshTokenRepository(q),
		Category:     postgres.NewCategoryRepository(q),
		Course:       postgres.NewCourseRepository(q),
		Module:       postgres.NewModuleRepository(q),
		Lesson:       postgres.NewLessonRepository(q),
		Quiz:         postgres.NewQuizRepository(q),
		Question:     postgres.NewQuestionRepository(q),
		Attempt:      postgres.NewAttemptRepository(q),
		Enrollment:   postgres.NewEnrollmentRepository(q),
		Progress:     postgres.NewProgressRepository(q),
		Payment:      postgres.NewPaymentRepository(q),
		Certificate:  postgres.NewCertificateRepository(q),
		Review:       postgres.NewReviewRepository(q),
		Finance:      postgres.NewFinanceRepository(q),
		Report:       postgres.NewReportRepository(q),
	}
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return newRepository(db, db)
}

// WithTx runs fn with repositories bound to a single transaction. The
// transaction is committed if fn returns nil and rolled back otherwise
// (including when fn panics).
func (r *Repository) WithTx(ctx context.Context, fn func(txRepo *Repository) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tx boshlashda xato: %w", err)
	}

	// no-op after a successful Commit
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(newRepository(nil, tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type User interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	GetByUsername(ctx context.Context, username string) (*models.GetByUsername, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetPasswordHash(ctx context.Context, id string) (string, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	Create(context.Context, models.CreateUser) (uuid.UUID, error)
	Update(context.Context, models.UpdateUser) (*models.User, error)
	UpdateStatus(context.Context, string, string,
	) error
	Delete(context.Context, string) error
	GetByID(context.Context, string) (*models.User, error)
	GetList(context.Context, models.UserFilter) ([]models.User, int, error)
}

type RefreshToken interface {
	Create(ctx context.Context, token models.RefreshToken) (uuid.UUID, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type Category interface {
	Create(ctx context.Context, category models.CreateCategory) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Category, error)
	GetList(ctx context.Context, filter models.CategoryFilter) ([]models.Category, int, error)
	Update(ctx context.Context, category models.UpdateCategory) (*models.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Course interface {
	Create(ctx context.Context, course models.CreateCourse) (uuid.UUID, error)
	CreateLearningOutcome(ctx context.Context, outcome models.CourseLearningOutcome) error
	CreateRequirement(ctx context.Context, requirement models.CourseRequirement) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Course, error)
	GetLearningOutcomes(ctx context.Context, courseID uuid.UUID) ([]models.CourseLearningOutcome, error)
	GetRequirements(ctx context.Context, courseID uuid.UUID) ([]models.CourseRequirement, error)
	GetList(ctx context.Context, filter models.CourseFilter) ([]models.CourseListItem, int, error)
	Update(ctx context.Context, course models.UpdateCourse) (*models.Course, error)
	UpdateStatus(ctx context.Context, status models.UpdateCourseStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteLearningOutcomes(ctx context.Context, courseID uuid.UUID) error
	DeleteRequirements(ctx context.Context, courseID uuid.UUID) error
}

type Module interface {
	Create(ctx context.Context, module models.CreateModule) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Module, error)
	GetListByCourseID(ctx context.Context, courseID uuid.UUID) ([]models.Module, error)
	Update(ctx context.Context, module models.UpdateModule) (*models.Module, error)
	UpdateOrder(ctx context.Context, module models.UpdateModuleOrder) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Lesson interface {
	Create(ctx context.Context, lesson models.CreateLesson) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Lesson, error)
	GetListByModuleID(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error)
	Update(ctx context.Context, lesson models.UpdateLesson) (*models.Lesson, error)
	UpdateOrder(ctx context.Context, lesson models.UpdateLessonOrder) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByModuleID(ctx context.Context, moduleID uuid.UUID) error

	CreateMaterial(ctx context.Context, material models.CreateLessonMaterial) (uuid.UUID, error)
	GetMaterialByID(ctx context.Context, id uuid.UUID) (*models.LessonMaterial, error)
	GetMaterials(ctx context.Context, lessonID uuid.UUID) ([]models.LessonMaterial, error)
	UpdateMaterial(ctx context.Context, material models.UpdateLessonMaterial) (*models.LessonMaterial, error)
	DeleteMaterial(ctx context.Context, id uuid.UUID) error
}

type Quiz interface {
	Create(ctx context.Context, quiz models.CreateQuiz) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Quiz, error)
	GetListByCourseID(ctx context.Context, courseID uuid.UUID) ([]models.Quiz, error)
	GetListByModuleID(ctx context.Context, moduleID uuid.UUID) ([]models.Quiz, error)
	Update(ctx context.Context, quiz models.UpdateQuiz) (*models.Quiz, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Question interface {
	Create(ctx context.Context, question models.CreateQuestion) (uuid.UUID, error)
	CreateOption(ctx context.Context, option models.CreateQuestionOption) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Question, error)
	GetListByQuizID(ctx context.Context, quizID uuid.UUID) ([]models.Question, error)
	GetOptions(ctx context.Context, questionID uuid.UUID) ([]models.QuestionOption, error)
	GetOptionsByQuizID(ctx context.Context, quizID uuid.UUID) ([]models.QuestionOption, error)
	Update(ctx context.Context, question models.UpdateQuestion) (*models.Question, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteOptions(ctx context.Context, questionID uuid.UUID) error
}

type Attempt interface {
	Create(ctx context.Context, attempt models.CreateQuizAttempt) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.QuizAttempt, error)
	GetList(ctx context.Context, quizID uuid.UUID, studentID *uuid.UUID) ([]models.QuizAttempt, error)
	CountByStudent(ctx context.Context, quizID uuid.UUID, studentID uuid.UUID) (int, error)
	Complete(ctx context.Context, attempt models.CompleteAttempt) error
	CreateAnswer(ctx context.Context, answer models.AttemptAnswer) error
	GetAnswers(ctx context.Context, attemptID uuid.UUID) ([]models.AttemptAnswer, error)
}

type Enrollment interface {
	Create(ctx context.Context, enrollment models.CreateEnrollment) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Enrollment, error)
	GetByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) (*models.Enrollment, error)
	GetListByCourseID(ctx context.Context, courseID uuid.UUID, filter models.EnrollmentFilter) ([]models.EnrollmentRosterItem, int, error)
	GetMyList(ctx context.Context, studentID uuid.UUID, filter models.EnrollmentFilter) ([]models.MyEnrollment, int, error)
	UpdateStatus(ctx context.Context, enrollment models.UpdateEnrollmentStatus) error
}

type Progress interface {
	Set(ctx context.Context, progress models.SetLessonProgress) (*models.LessonProgress, error)
	GetByLesson(ctx context.Context, studentID uuid.UUID, lessonID uuid.UUID) (*models.LessonProgress, error)
	GetCourseProgress(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) (*models.CourseProgress, error)
	GetCompletedLessonIDs(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) ([]uuid.UUID, error)
	GetStudentList(ctx context.Context, courseID uuid.UUID, filter models.EnrollmentFilter) ([]models.StudentProgress, int, error)
}

type Payment interface {
	Create(ctx context.Context, payment models.CreatePayment) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Payment, error)
	GetByEnrollmentID(ctx context.Context, enrollmentID uuid.UUID) (*models.Payment, error)
	GetList(ctx context.Context, filter models.PaymentFilter) ([]models.Payment, int, error)
	CreatePayout(ctx context.Context, payout models.CreateInstructorPayout) (uuid.UUID, error)
}

type Certificate interface {
	Create(ctx context.Context, certificate models.CreateCertificate) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.CertificateDetail, error)
	GetByUniqueID(ctx context.Context, uniqueID string) (*models.CertificateDetail, error)
	GetByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) (*models.Certificate, error)
	GetListByStudentID(ctx context.Context, studentID uuid.UUID) ([]models.CertificateDetail, error)
}

type Review interface {
	Create(ctx context.Context, review models.CreateReview) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Review, error)
	GetListByCourseID(ctx context.Context, courseID uuid.UUID, filter models.ReviewFilter) ([]models.Review, int, error)
	GetAverageRating(ctx context.Context, courseID uuid.UUID) (float64, error)
	GetInstructorAverageRating(ctx context.Context, instructorID uuid.UUID) (float64, error)
	Update(ctx context.Context, review models.UpdateReview) (*models.Review, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Finance interface {
	GetSummary(ctx context.Context, filter models.FinanceFilter) (*models.FinanceSummary, error)
	GetRevenue(ctx context.Context, filter models.FinanceFilter) (*models.RevenueSummary, error)
	GetExpenses(ctx context.Context, filter models.FinanceFilter) (*models.ExpenseSummary, error)
	GetPayouts(ctx context.Context, filter models.PaymentFilter) ([]models.InstructorPayout, int, error)
}

type Report interface {
	Enrollments(ctx context.Context, filter models.ReportFilter) ([]models.EnrollmentReportRow, error)
	EnrollmentTrend(ctx context.Context, filter models.ReportFilter) ([]models.TrendRow, error)
	Revenue(ctx context.Context, filter models.ReportFilter) ([]models.RevenueReportRow, error)
	Students(ctx context.Context, filter models.ReportFilter) ([]models.StudentReportRow, error)
	StudentGrowth(ctx context.Context, filter models.ReportFilter) ([]models.TrendRow, error)
	Progress(ctx context.Context, filter models.ReportFilter) ([]models.ProgressReportRow, error)
	ProgressFunnel(ctx context.Context, courseID uuid.UUID) ([]models.ProgressFunnelRow, error)
	Quizzes(ctx context.Context, filter models.ReportFilter) ([]models.QuizReportRow, error)
	MostFailedQuestions(ctx context.Context, filter models.ReportFilter) ([]models.MostFailedQuestionRow, error)
	Certificates(ctx context.Context, filter models.ReportFilter) ([]models.CertificateReportRow, error)
	CertificateTrend(ctx context.Context, filter models.ReportFilter) ([]models.TrendRow, error)
	Instructors(ctx context.Context, filter models.ReportFilter) ([]models.InstructorReportRow, error)
	Reviews(ctx context.Context, filter models.ReportFilter) ([]models.ReviewReportRow, error)
}
