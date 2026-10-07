package service

import (
	"github.com/diyorbeknematov/lms/internal/service/account"
	"github.com/diyorbeknematov/lms/internal/service/catalog"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/internal/service/files"
	"github.com/diyorbeknematov/lms/internal/service/finance"
	"github.com/diyorbeknematov/lms/internal/service/learning"
)

// Dependencies and Config are defined in core; they are repeated here so that
// main only imports this package.
type (
	Dependencies = core.Dependencies
	Config       = core.Config
)

// Service gathers every service, so the handlers know only this one type.
// Services are grouped in packages by domain and added here as they are
// written.
type Service struct {
	Auth     *account.Auth
	User     *account.User
	Category *catalog.Category
	Course   *catalog.Course
	Module   *catalog.Module
	Lesson   *catalog.Lesson

	Enrollment *learning.Enrollment
	Progress   *learning.Progress
	Quiz       *learning.Quiz
	Question   *learning.Question
	Attempt    *learning.Attempt

	Certificate *learning.Certificate
	Review      *learning.Review

	Upload *files.Upload

	Finance *finance.Finance
	Payment *finance.Payment
	Report  *finance.Report
}

func New(deps Dependencies) *Service {
	return &Service{
		Auth:     account.NewAuth(deps),
		User:     account.NewUser(deps),
		Category: catalog.NewCategory(deps),
		Course:   catalog.NewCourse(deps),
		Module:   catalog.NewModule(deps),
		Lesson:   catalog.NewLesson(deps),

		Enrollment: learning.NewEnrollment(deps),
		Progress:   learning.NewProgress(deps),
		Quiz:       learning.NewQuiz(deps),
		Question:   learning.NewQuestion(deps),
		Attempt:    learning.NewAttempt(deps),

		Certificate: learning.NewCertificate(deps),
		Review:      learning.NewReview(deps),

		Upload: files.NewUpload(deps),

		Finance: finance.NewFinance(deps),
		Payment: finance.NewPayment(deps),
		Report:  finance.NewReport(deps),
	}
}
