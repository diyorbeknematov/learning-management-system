package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/casbin/casbin/v2"
	_ "github.com/diyorbeknematov/lms/docs/swagger"
	"github.com/diyorbeknematov/lms/internal/api/handler"
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/api/validation"
	"github.com/diyorbeknematov/lms/internal/service"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Dependencies struct {
	Service     *service.Service
	Tokens      *token.Manager
	Enforcer    *casbin.Enforcer
	Logger      *slog.Logger
	CORSOrigins []string
	Production  bool

	// Limiter counts requests for the rate limits; nil turns them off.
	Limiter middleware.Limiter
	// Revocations tells which access tokens were taken back; nil turns it off.
	Revocations middleware.Revocations
	// TrustedProxies are the addresses of the reverse proxies in front of the
	// API. Only they may tell the real address of a client in X-Forwarded-For.
	TrustedProxies []string
}

// NewRouter builds the HTTP router. The routes are in two groups:
//
//   - public: no access token is needed (an optional token only tells the
//     service who is asking, for example the owner of a draft course);
//   - private: a valid access token is needed and Casbin must allow the role
//     of the user to call the route (see authz/policy.csv).
func NewRouter(d Dependencies) *gin.Engine {
	validation.Setup()

	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// the address of the client is the address of the connection, a header
	// like X-Forwarded-For can be written by anybody; only a proxy we trust
	// may tell the real address
	if err := router.SetTrustedProxies(d.TrustedProxies); err != nil {
		panic("api: invalid trusted proxies: " + err.Error())
	}

	router.HandleMethodNotAllowed = true

	router.Use(
		middleware.RequestID(),
		middleware.Recover(d.Logger),
		middleware.Logger(d.Logger),
		middleware.CORS(d.CORSOrigins),
	)

	router.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "route not found")
	})

	router.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, response.CodeInvalidInput, "method not allowed")
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// the interactive documentation is for development; it is not served in
	// production
	if !d.Production {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := router.Group("/api/v1")

	public := v1.Group("", middleware.Authenticate(d.Tokens, d.Revocations, false))
	private := v1.Group("", middleware.Authenticate(d.Tokens, d.Revocations, true), middleware.Authorize(d.Enforcer))

	handler.NewAuth(d.Service.Auth).Public(public, func(name string, max int, window time.Duration) gin.HandlerFunc {
		return middleware.RateLimit(d.Limiter, d.Logger, name, max, window)
	})

	categories := handler.NewCategory(d.Service.Category)
	courses := handler.NewCourse(d.Service.Course)
	modules := handler.NewModule(d.Service.Module)
	lessons := handler.NewLesson(d.Service.Lesson)

	enrollments := handler.NewEnrollment(d.Service.Enrollment, d.Service.Progress)
	quizzes := handler.NewQuiz(d.Service.Quiz, d.Service.Question, d.Service.Attempt)
	reviews := handler.NewReview(d.Service.Certificate, d.Service.Review)

	categories.Public(public)
	courses.Public(public)
	modules.Public(public)
	lessons.Public(public)
	reviews.Public(public)

	handler.NewUser(d.Service.User).Private(private)
	handler.NewUpload(d.Service.Upload).Private(private)
	categories.Private(private)
	courses.Private(private)
	modules.Private(private)
	lessons.Private(private)
	enrollments.Private(private)
	quizzes.Private(private)
	reviews.Private(private)
	handler.NewFinance(d.Service.Finance, d.Service.Payment).Private(private)
	handler.NewReport(d.Service.Report).Private(private)

	return router
}
