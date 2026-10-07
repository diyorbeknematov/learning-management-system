package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/api/authz"
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var tokens = token.NewManager("test-secret", time.Minute)

func init() {
	gin.SetMode(gin.TestMode)
}

type answer struct {
	Status  int
	Body    map[string]any
	Headers http.Header
}

func (a answer) errorCode() string {
	errBody, _ := a.Body["error"].(map[string]any)
	code, _ := errBody["code"].(string)

	return code
}

func call(router *gin.Engine, method, path, authorization string, headers ...string) answer {
	request := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)

	return answer{Status: recorder.Code, Body: body, Headers: recorder.Header()}
}

func bearer(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()

	raw, err := tokens.GenerateAccessToken(userID, uuid.New(), role)
	require.NoError(t, err)

	return "Bearer " + raw
}

// secured is a router with the same two groups as the real one and a few
// routes of the policy.
func secured(t *testing.T) *gin.Engine {
	t.Helper()

	enforcer, err := authz.New()
	require.NoError(t, err)

	router := gin.New()
	router.Use(middleware.RequestID())

	whoami := func(c *gin.Context) {
		actor := middleware.ActorFrom(c)
		response.OK(c, gin.H{"user_id": actor.UserID.String(), "role": actor.RoleName})
	}

	public := router.Group("/api/v1", middleware.Authenticate(tokens, nil, false))
	public.GET("/courses", whoami)

	private := router.Group("/api/v1", middleware.Authenticate(tokens, nil, true), middleware.Authorize(enforcer))
	private.GET("/users/me", whoami)
	private.GET("/users", whoami)
	private.POST("/courses/:courseId/enrollments", whoami)
	private.POST("/courses", whoami)
	private.GET("/closed", whoami)

	return router
}

func TestAuthenticate_PrivateRouteNeedsAToken(t *testing.T) {
	router := secured(t)

	got := call(router, http.MethodGet, "/api/v1/users/me", "")
	require.Equal(t, http.StatusUnauthorized, got.Status)
	require.Equal(t, "UNAUTHORIZED", got.errorCode())
	require.Equal(t, false, got.Body["success"])
}

func TestAuthenticate_BadHeaders(t *testing.T) {
	router := secured(t)

	expired, err := token.NewManager("test-secret", -time.Minute).GenerateAccessToken(uuid.New(), uuid.New(), models.RoleStudent)
	require.NoError(t, err)

	otherSecret, err := token.NewManager("other-secret", time.Minute).GenerateAccessToken(uuid.New(), uuid.New(), models.RoleStudent)
	require.NoError(t, err)

	for name, header := range map[string]string{
		"garbage":           "Bearer not-a-token",
		"no scheme":         "justatoken",
		"empty token":       "Bearer ",
		"basic scheme":      "Basic dXNlcjpwYXNz",
		"expired":           "Bearer " + expired,
		"another secret":    "Bearer " + otherSecret,
		"two spaces inside": "Bearer a b",
	} {
		got := call(router, http.MethodGet, "/api/v1/users/me", header)
		require.Equal(t, http.StatusUnauthorized, got.Status, name)
		require.Equal(t, "UNAUTHORIZED", got.errorCode(), name)
	}
}

func TestAuthenticate_SchemeIsNotCaseSensitive(t *testing.T) {
	router := secured(t)

	userID := uuid.New()
	header := "bearer " + bearer(t, userID, models.RoleStudent)[len("Bearer "):]

	got := call(router, http.MethodGet, "/api/v1/users/me", header)
	require.Equal(t, http.StatusOK, got.Status)
}

func TestAuthenticate_TheActorComesFromTheToken(t *testing.T) {
	router := secured(t)

	userID := uuid.New()

	got := call(router, http.MethodGet, "/api/v1/users/me", bearer(t, userID, models.RoleInstructor))
	require.Equal(t, http.StatusOK, got.Status)

	data := got.Body["data"].(map[string]any)
	require.Equal(t, userID.String(), data["user_id"])
	require.Equal(t, models.RoleInstructor, data["role"])
}

func TestAuthenticate_PublicRouteWorksWithoutAToken(t *testing.T) {
	router := secured(t)

	got := call(router, http.MethodGet, "/api/v1/courses", "")
	require.Equal(t, http.StatusOK, got.Status)

	data := got.Body["data"].(map[string]any)
	require.Equal(t, uuid.Nil.String(), data["user_id"], "a visitor is an empty actor")
	require.Equal(t, "", data["role"])
}

func TestAuthenticate_PublicRouteKnowsTheUserWhenATokenIsSent(t *testing.T) {
	router := secured(t)

	userID := uuid.New()

	got := call(router, http.MethodGet, "/api/v1/courses", bearer(t, userID, models.RoleStudent))
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, userID.String(), got.Body["data"].(map[string]any)["user_id"])
}

func TestAuthenticate_PublicRouteRejectsABadToken(t *testing.T) {
	router := secured(t)

	got := call(router, http.MethodGet, "/api/v1/courses", "Bearer not-a-token")
	require.Equal(t, http.StatusUnauthorized, got.Status, "a wrong token is 401 even where no token is needed, so the client refreshes it")
}

func TestAuthorize_ByRole(t *testing.T) {
	router := secured(t)

	cases := []struct {
		role   string
		method string
		path   string
		status int
	}{
		{models.RoleStudent, http.MethodGet, "/api/v1/users/me", http.StatusOK},
		{models.RoleInstructor, http.MethodGet, "/api/v1/users/me", http.StatusOK},
		{models.RoleSuperAdmin, http.MethodGet, "/api/v1/users/me", http.StatusOK},

		{models.RoleStudent, http.MethodGet, "/api/v1/users", http.StatusForbidden},
		{models.RoleInstructor, http.MethodGet, "/api/v1/users", http.StatusForbidden},
		{models.RoleSuperAdmin, http.MethodGet, "/api/v1/users", http.StatusOK},

		{models.RoleStudent, http.MethodPost, "/api/v1/courses", http.StatusForbidden},
		{models.RoleInstructor, http.MethodPost, "/api/v1/courses", http.StatusOK},
		{models.RoleSuperAdmin, http.MethodPost, "/api/v1/courses", http.StatusOK},

		{models.RoleStudent, http.MethodPost, "/api/v1/courses/" + uuid.NewString() + "/enrollments", http.StatusOK},
		{models.RoleInstructor, http.MethodPost, "/api/v1/courses/" + uuid.NewString() + "/enrollments", http.StatusForbidden},
		{models.RoleSuperAdmin, http.MethodPost, "/api/v1/courses/" + uuid.NewString() + "/enrollments", http.StatusForbidden},

		// a route that is not in the policy is closed to everybody
		{models.RoleStudent, http.MethodGet, "/api/v1/closed", http.StatusForbidden},
		{models.RoleInstructor, http.MethodGet, "/api/v1/closed", http.StatusForbidden},
		{models.RoleSuperAdmin, http.MethodGet, "/api/v1/closed", http.StatusForbidden},

		// a role the policy does not know has no rights
		{"Hacker", http.MethodGet, "/api/v1/users/me", http.StatusForbidden},
		{"", http.MethodGet, "/api/v1/users/me", http.StatusForbidden},
	}

	for _, tt := range cases {
		got := call(router, tt.method, tt.path, bearer(t, uuid.New(), tt.role))
		require.Equal(t, tt.status, got.Status, "%s %s %s", tt.role, tt.method, tt.path)

		if tt.status == http.StatusForbidden {
			require.Equal(t, "FORBIDDEN", got.errorCode())
		}
	}
}

func TestRequestID(t *testing.T) {
	router := secured(t)

	generated := call(router, http.MethodGet, "/api/v1/courses", "")
	_, err := uuid.Parse(generated.Headers.Get("X-Request-ID"))
	require.NoError(t, err)

	mine := uuid.NewString()

	kept := call(router, http.MethodGet, "/api/v1/courses", "", "X-Request-ID", mine)
	require.Equal(t, mine, kept.Headers.Get("X-Request-ID"))

	replaced := call(router, http.MethodGet, "/api/v1/courses", "", "X-Request-ID", "<script>alert(1)</script>")
	require.NotEqual(t, "<script>alert(1)</script>", replaced.Headers.Get("X-Request-ID"))
	_, err = uuid.Parse(replaced.Headers.Get("X-Request-ID"))
	require.NoError(t, err)
}

func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS([]string{"http://localhost:3000"}))
	router.GET("/x", func(c *gin.Context) { response.OK(c, nil) })

	allowed := call(router, http.MethodGet, "/x", "", "Origin", "http://localhost:3000")
	require.Equal(t, "http://localhost:3000", allowed.Headers.Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Origin", allowed.Headers.Get("Vary"))

	other := call(router, http.MethodGet, "/x", "", "Origin", "http://evil.example")
	require.Empty(t, other.Headers.Get("Access-Control-Allow-Origin"))
	require.Equal(t, http.StatusOK, other.Status, "the server answers; the browser is the one that blocks it")

	none := call(router, http.MethodGet, "/x", "")
	require.Empty(t, none.Headers.Get("Access-Control-Allow-Origin"))
}

func TestCORS_Preflight(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS([]string{"http://localhost:3000"}))
	router.POST("/x", func(c *gin.Context) { response.OK(c, nil) })

	got := call(router, http.MethodOptions, "/x", "",
		"Origin", "http://localhost:3000",
		"Access-Control-Request-Method", "POST",
	)
	require.Equal(t, http.StatusNoContent, got.Status)
	require.Contains(t, got.Headers.Get("Access-Control-Allow-Methods"), "POST")
	require.Contains(t, got.Headers.Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestRecover_APanicBecomesA500(t *testing.T) {
	router := gin.New()
	router.Use(middleware.Recover(slog.New(slog.DiscardHandler)))
	router.GET("/boom", func(c *gin.Context) { panic("something secret broke") })

	got := call(router, http.MethodGet, "/boom", "")
	require.Equal(t, http.StatusInternalServerError, got.Status)
	require.Equal(t, "INTERNAL", got.errorCode())
	require.NotContains(t, got.Body["error"].(map[string]any)["message"], "secret")
}

func TestFail_MapsServiceErrors(t *testing.T) {
	router := gin.New()

	router.GET("/:kind", func(c *gin.Context) {
		errs := map[string]error{
			"input":        apperror.InvalidInput("service", "op", "name is too short", apperror.ErrInvalidInput),
			"unauthorized": apperror.Unauthorized("service", "op", "invalid username or password", apperror.ErrUnauthorized),
			"forbidden":    apperror.Forbidden("service", "op", "not yours", apperror.ErrForbidden),
			"notfound":     apperror.NotFound("service", "op", "course not found", apperror.ErrNotFound),
			"conflict":     apperror.Conflict("service", "op", "already exists", apperror.ErrAlreadyExists),
			"internal":     apperror.Internal("repository", "op", "failed to get course", http.ErrAbortHandler),
			"plain":        http.ErrAbortHandler,
		}

		response.Fail(c, errs[c.Param("kind")])
	})

	cases := map[string]struct {
		status  int
		code    string
		message string
	}{
		"input":        {http.StatusBadRequest, "INVALID_INPUT", "name is too short"},
		"unauthorized": {http.StatusUnauthorized, "UNAUTHORIZED", "invalid username or password"},
		"forbidden":    {http.StatusForbidden, "FORBIDDEN", "not yours"},
		"notfound":     {http.StatusNotFound, "NOT_FOUND", "course not found"},
		"conflict":     {http.StatusConflict, "CONFLICT", "already exists"},
		"internal":     {http.StatusInternalServerError, "INTERNAL", "internal server error"},
		"plain":        {http.StatusInternalServerError, "INTERNAL", "internal server error"},
	}

	for kind, want := range cases {
		got := call(router, http.MethodGet, "/"+kind, "")
		require.Equal(t, want.status, got.Status, kind)
		require.Equal(t, want.code, got.errorCode(), kind)
		require.Equal(t, want.message, got.Body["error"].(map[string]any)["message"], kind)
		require.Equal(t, false, got.Body["success"], kind)
	}
}

// counter is a Limiter in memory: it counts the hits of every key and says the
// window has the time given in left.
type counter struct {
	hits map[string]int64
	left time.Duration
	err  error
}

func newCounter(left time.Duration) *counter {
	return &counter{hits: map[string]int64{}, left: left}
}

func (c *counter) Incr(_ context.Context, key string, _ time.Duration) (int64, time.Duration, error) {
	if c.err != nil {
		return 0, 0, c.err
	}

	c.hits[key]++

	return c.hits[key], c.left, nil
}

func limited(limiter middleware.Limiter, max int) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)

	router.GET("/x", middleware.RateLimit(limiter, slog.New(slog.DiscardHandler), "x", max, time.Minute), func(c *gin.Context) {
		response.OK(c, nil)
	})
	router.GET("/y", middleware.RateLimit(limiter, slog.New(slog.DiscardHandler), "y", max, time.Minute), func(c *gin.Context) {
		response.OK(c, nil)
	})

	return router
}

func callFrom(router *gin.Engine, address, path string) answer {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = address + ":4000"

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)

	return answer{Status: recorder.Code, Body: body, Headers: recorder.Header()}
}

func TestRateLimit_LetsTheFirstRequestsThroughAndStopsTheRest(t *testing.T) {
	router := limited(newCounter(42*time.Second), 3)

	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.5", "/x").Status)
	}

	got := callFrom(router, "203.0.113.5", "/x")
	require.Equal(t, http.StatusTooManyRequests, got.Status)
	require.Equal(t, "TOO_MANY_REQUESTS", got.errorCode())
	require.Equal(t, "42", got.Headers.Get("Retry-After"))
	require.Contains(t, got.Body["error"].(map[string]any)["message"], "42 seconds")
	require.Equal(t, false, got.Body["success"])
}

func TestRateLimit_RetryAfterRoundsUp(t *testing.T) {
	router := limited(newCounter(1500*time.Millisecond), 0)

	got := callFrom(router, "203.0.113.5", "/x")
	require.Equal(t, http.StatusTooManyRequests, got.Status)
	require.Equal(t, "2", got.Headers.Get("Retry-After"), "never tell the client to come back too early")
}

func TestRateLimit_EachAddressAndEachRouteHasItsOwnCount(t *testing.T) {
	router := limited(newCounter(time.Minute), 1)

	require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.5", "/x").Status)
	require.Equal(t, http.StatusTooManyRequests, callFrom(router, "203.0.113.5", "/x").Status)

	require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.6", "/x").Status, "another address")
	require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.5", "/y").Status, "another route")
}

func TestRateLimit_AHeaderCannotChangeTheAddress(t *testing.T) {
	router := limited(newCounter(time.Minute), 1)

	first := httptest.NewRequest(http.MethodGet, "/x", nil)
	first.RemoteAddr = "203.0.113.5:4000"
	first.Header.Set("X-Forwarded-For", "198.51.100.1")

	second := httptest.NewRequest(http.MethodGet, "/x", nil)
	second.RemoteAddr = "203.0.113.5:4001"
	second.Header.Set("X-Forwarded-For", "198.51.100.2")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, first)
	require.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, second)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code, "a made-up X-Forwarded-For does not give a new count")
}

func TestRateLimit_NilLimiterTurnsItOff(t *testing.T) {
	router := limited(nil, 1)

	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.5", "/x").Status)
	}
}

func TestRateLimit_AnUnreachableLimiterDoesNotStopTheAPI(t *testing.T) {
	limiter := newCounter(time.Minute)
	limiter.err = errors.New("redis is down")

	router := limited(limiter, 1)

	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusOK, callFrom(router, "203.0.113.5", "/x").Status)
	}
}

// revocationList is a Revocations that knows which users it threw out.
type revocationList struct {
	revoked map[uuid.UUID]bool
	err     error
}

func (r revocationList) IsRevoked(_ context.Context, userID uuid.UUID, _ time.Time) (bool, error) {
	return r.revoked[userID], r.err
}

func withRevocations(t *testing.T, revocations middleware.Revocations) *gin.Engine {
	t.Helper()

	router := gin.New()

	router.GET("/me", middleware.Authenticate(tokens, revocations, true), func(c *gin.Context) {
		response.OK(c, nil)
	})
	router.GET("/open", middleware.Authenticate(tokens, revocations, false), func(c *gin.Context) {
		response.OK(c, nil)
	})

	return router
}

func TestAuthenticate_ARevokedTokenIsRefused(t *testing.T) {
	blocked, fine := uuid.New(), uuid.New()
	router := withRevocations(t, revocationList{revoked: map[uuid.UUID]bool{blocked: true}})

	got := call(router, http.MethodGet, "/me", bearer(t, blocked, models.RoleStudent))
	require.Equal(t, http.StatusUnauthorized, got.Status)
	require.Equal(t, "UNAUTHORIZED", got.errorCode())
	require.Contains(t, got.Body["error"].(map[string]any)["message"], "log in again")

	require.Equal(t, http.StatusOK, call(router, http.MethodGet, "/me", bearer(t, fine, models.RoleStudent)).Status)

	// it holds on the routes where a token is optional, too
	require.Equal(t, http.StatusUnauthorized, call(router, http.MethodGet, "/open", bearer(t, blocked, models.RoleStudent)).Status)
	require.Equal(t, http.StatusOK, call(router, http.MethodGet, "/open", "").Status, "no token, nothing to revoke")
}

func TestAuthenticate_WhenTheRevocationsCannotBeReadTheRequestIsRefused(t *testing.T) {
	router := withRevocations(t, revocationList{err: errors.New("redis is down")})

	got := call(router, http.MethodGet, "/me", bearer(t, uuid.New(), models.RoleStudent))
	require.Equal(t, http.StatusInternalServerError, got.Status, "not letting a thrown-out user in is more important")
}
