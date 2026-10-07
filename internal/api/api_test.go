package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/api"
	"github.com/diyorbeknematov/lms/internal/api/authz"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// client calls the real router, with the real services on a real database (the
// storage, Redis and mail are fakes).
type client struct {
	t      *testing.T
	e      *testutil.Env
	router *gin.Engine
	// remote is the address the requests come from; empty is the default one of
	// httptest (192.0.2.1).
	remote string
}

func newClient(t *testing.T) *client {
	t.Helper()

	return newClientWith(t, nil)
}

// newClientWith builds the router the same way and lets the test change its
// dependencies first.
func newClientWith(t *testing.T, change func(*api.Dependencies)) *client {
	t.Helper()

	e := testutil.Setup(t)

	enforcer, err := authz.New()
	require.NoError(t, err)

	deps := api.Dependencies{
		Service:     e.Svc,
		Tokens:      e.Tokens,
		Enforcer:    enforcer,
		Logger:      slog.New(slog.DiscardHandler),
		CORSOrigins: []string{"http://localhost:3000"},
		Limiter:     e.Redis,
		Revocations: core.NewRevocations(e.Redis, time.Minute),
	}

	if change != nil {
		change(&deps)
	}

	return &client{t: t, e: e, router: api.NewRouter(deps)}
}

// from returns the same client, but its requests come from another address.
func (c *client) from(address string) *client {
	other := *c
	other.remote = address

	return &other
}

type answer struct {
	Status  int
	Body    map[string]any
	Raw     string
	Headers http.Header
}

func (a answer) data() map[string]any {
	data, _ := a.Body["data"].(map[string]any)

	return data
}

func (a answer) errorCode() string {
	body, _ := a.Body["error"].(map[string]any)
	code, _ := body["code"].(string)

	return code
}

func (a answer) errorMessage() string {
	body, _ := a.Body["error"].(map[string]any)
	message, _ := body["message"].(string)

	return message
}

func (a answer) fields() map[string]any {
	body, _ := a.Body["error"].(map[string]any)
	fields, _ := body["fields"].(map[string]any)

	return fields
}

// send sends a request with a JSON body (a string is sent as it is).
func (c *client) send(method, path string, body any, authorization string) answer {
	c.t.Helper()

	return c.sendHeaders(method, path, body, authorization, nil)
}

// sendHeaders is send with more headers.
func (c *client) sendHeaders(method, path string, body any, authorization string, headers map[string]string) answer {
	c.t.Helper()

	var reader *bytes.Reader

	switch value := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(value))
	default:
		raw, err := json.Marshal(value)
		require.NoError(c.t, err)

		reader = bytes.NewReader(raw)
	}

	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")

	if c.remote != "" {
		request.RemoteAddr = c.remote + ":4000"
	}

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	recorder := httptest.NewRecorder()
	c.router.ServeHTTP(recorder, request)

	parsed := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &parsed)

	return answer{Status: recorder.Code, Body: parsed, Raw: recorder.Body.String(), Headers: recorder.Header()}
}

func (c *client) post(path string, body any) answer {
	return c.send(http.MethodPost, path, body, "")
}

func uniq() string {
	return testutil.Uniq()
}

func registerBody() map[string]any {
	name := uniq()

	return map[string]any{
		"first_name": "Ali",
		"last_name":  "Valiyev",
		"username":   "u" + name,
		"email":      name + "@example.com",
		"password":   testutil.Password,
	}
}

func TestHealth(t *testing.T) {
	c := newClient(t)

	got := c.send(http.MethodGet, "/health", nil, "")
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, "ok", got.Body["status"])
}

func TestUnknownRouteAndMethod(t *testing.T) {
	c := newClient(t)

	got := c.send(http.MethodGet, "/api/v1/nothing-here", nil, "")
	require.Equal(t, http.StatusNotFound, got.Status)
	require.Equal(t, "NOT_FOUND", got.errorCode())
	require.Equal(t, false, got.Body["success"])

	got = c.send(http.MethodGet, "/api/v1/auth/login", nil, "")
	require.Equal(t, http.StatusMethodNotAllowed, got.Status)
	require.Equal(t, "INVALID_INPUT", got.errorCode())
}

func TestEveryAnswerHasARequestID(t *testing.T) {
	c := newClient(t)

	require.NotEmpty(t, c.send(http.MethodGet, "/health", nil, "").Headers.Get("X-Request-ID"))
	require.NotEmpty(t, c.send(http.MethodGet, "/api/v1/nothing-here", nil, "").Headers.Get("X-Request-ID"))
}

func TestRegister(t *testing.T) {
	c := newClient(t)

	body := registerBody()

	got := c.post("/api/v1/auth/register", body)
	require.Equal(t, http.StatusCreated, got.Status, got.Raw)
	require.Equal(t, true, got.Body["success"])
	require.NotEmpty(t, got.data()["access_token"])
	require.NotEmpty(t, got.data()["refresh_token"])

	claims, err := c.e.Tokens.ParseAccessToken(got.data()["access_token"].(string))
	require.NoError(t, err)
	require.Equal(t, models.RoleStudent, claims.RoleName)

	c.e.CleanupUser(t, claims.UserID)
}

func TestRegister_TheAnswerNeverShowsThePassword(t *testing.T) {
	c := newClient(t)

	body := registerBody()
	got := c.post("/api/v1/auth/register", body)
	require.Equal(t, http.StatusCreated, got.Status)

	claims, err := c.e.Tokens.ParseAccessToken(got.data()["access_token"].(string))
	require.NoError(t, err)

	c.e.CleanupUser(t, claims.UserID)

	login := c.post("/api/v1/auth/login", map[string]any{"username": body["username"], "password": body["password"]})
	require.Equal(t, http.StatusOK, login.Status)
	require.NotContains(t, login.Raw, "password", "no password field, no hash")
	require.NotContains(t, login.Raw, "$2a$")
}

func TestRegister_ValidationFields(t *testing.T) {
	c := newClient(t)

	got := c.post("/api/v1/auth/register", map[string]any{
		"username": "a b",
		"email":    "not-an-email",
		"password": "weak",
	})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Equal(t, "INVALID_INPUT", got.errorCode())
	require.Equal(t, "validation failed", got.errorMessage())

	fields := got.fields()
	require.Contains(t, fields["first_name"], "required")
	require.Contains(t, fields["last_name"], "required")
	require.Contains(t, fields["username"], "3 to 30")
	require.Contains(t, fields["email"], "valid email")
	require.Contains(t, fields["password"], "capital letter")
}

func TestRegister_FieldsAreNamedByTheirJSONName(t *testing.T) {
	c := newClient(t)

	got := c.post("/api/v1/auth/register", map[string]any{})

	for name := range got.fields() {
		require.Equal(t, strings.ToLower(name), name, "json names, not Go names like FirstName")
		require.NotContains(t, name, "RegisterRequest")
	}

	require.Contains(t, got.fields(), "first_name")
}

func TestRegister_BodyProblems(t *testing.T) {
	c := newClient(t)

	cases := map[string]struct {
		body    any
		status  int
		message string
	}{
		"empty body":    {nil, http.StatusBadRequest, "empty"},
		"not json":      {"{this is not json", http.StatusBadRequest, "malformed"},
		"wrong type":    {`{"email": 5}`, http.StatusBadRequest, "malformed"},
		"an array":      {`[1, 2]`, http.StatusBadRequest, "malformed"},
		"far too large": {`{"first_name": "` + strings.Repeat("a", 2<<20) + `"}`, http.StatusRequestEntityTooLarge, "too large"},
	}

	for name, tt := range cases {
		got := c.post("/api/v1/auth/register", tt.body)
		require.Equal(t, tt.status, got.Status, name)
		require.Equal(t, "INVALID_INPUT", got.errorCode(), name)
		require.Contains(t, got.errorMessage(), tt.message, name)
	}
}

func TestRegister_Duplicate(t *testing.T) {
	c := newClient(t)

	body := registerBody()

	first := c.post("/api/v1/auth/register", body)
	require.Equal(t, http.StatusCreated, first.Status)

	claims, err := c.e.Tokens.ParseAccessToken(first.data()["access_token"].(string))
	require.NoError(t, err)

	c.e.CleanupUser(t, claims.UserID)

	again := registerBody()
	again["email"] = strings.ToUpper(body["email"].(string))

	got := c.post("/api/v1/auth/register", again)
	require.Equal(t, http.StatusConflict, got.Status)
	require.Equal(t, "CONFLICT", got.errorCode())
}

func TestLogin(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleInstructor)

	got := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password})
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.NotEmpty(t, got.data()["access_token"])
	require.NotEmpty(t, got.data()["refresh_token"])

	loggedIn, ok := got.data()["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, user.Username, loggedIn["username"])
	require.Equal(t, models.RoleInstructor, loggedIn["role_name"])
}

func TestLogin_WrongCredentials(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	wrongPassword := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": "Wrong-Passw0rd!"})
	require.Equal(t, http.StatusUnauthorized, wrongPassword.Status)
	require.Equal(t, "UNAUTHORIZED", wrongPassword.errorCode())

	unknown := c.post("/api/v1/auth/login", map[string]any{"username": "nobody" + uniq(), "password": "Wrong-Passw0rd!"})
	require.Equal(t, http.StatusUnauthorized, unknown.Status)
	require.Equal(t, wrongPassword.errorMessage(), unknown.errorMessage(), "the answers must not tell a wrong username from a wrong password")
}

func TestLogin_MissingFields(t *testing.T) {
	c := newClient(t)

	got := c.post("/api/v1/auth/login", map[string]any{})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Contains(t, got.fields(), "username")
	require.Contains(t, got.fields(), "password")
}

func TestLogin_BlockedUser(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)
	c.e.Exec(t, `UPDATE users SET status = 'blocked' WHERE id = $1`, user.ID)

	got := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password})
	require.Equal(t, http.StatusForbidden, got.Status)
	require.Equal(t, "FORBIDDEN", got.errorCode())
}

func TestRefreshAndLogout(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)
	login := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password})
	refresh := login.data()["refresh_token"].(string)

	renewed := c.post("/api/v1/auth/refresh", map[string]any{"refresh_token": refresh})
	require.Equal(t, http.StatusOK, renewed.Status, renewed.Raw)
	require.NotEqual(t, refresh, renewed.data()["refresh_token"])

	reused := c.post("/api/v1/auth/refresh", map[string]any{"refresh_token": refresh})
	require.Equal(t, http.StatusUnauthorized, reused.Status, "a refresh token works once")

	next := renewed.data()["refresh_token"].(string)

	logout := c.post("/api/v1/auth/logout", map[string]any{"refresh_token": next})
	require.Equal(t, http.StatusNoContent, logout.Status)
	require.Empty(t, logout.Raw)

	after := c.post("/api/v1/auth/refresh", map[string]any{"refresh_token": next})
	require.Equal(t, http.StatusUnauthorized, after.Status)

	again := c.post("/api/v1/auth/logout", map[string]any{"refresh_token": next})
	require.Equal(t, http.StatusNoContent, again.Status, "logging out twice is fine")
}

func TestRefresh_MissingToken(t *testing.T) {
	c := newClient(t)

	got := c.post("/api/v1/auth/refresh", map[string]any{})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Contains(t, got.fields(), "refresh_token")
}

func TestForgotAndResetPassword(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	forgot := c.post("/api/v1/auth/forgot-password", map[string]any{"email": user.Email})
	require.Equal(t, http.StatusOK, forgot.Status, forgot.Raw)
	require.NotEmpty(t, forgot.data()["message"])

	match := regexp.MustCompile(`\?token=(\S+)`).FindStringSubmatch(c.e.Mailer.Body)
	require.Len(t, match, 2)

	raw, err := url.QueryUnescape(match[1])
	require.NoError(t, err)

	weak := c.post("/api/v1/auth/reset-password", map[string]any{"token": raw, "new_password": "weak"})
	require.Equal(t, http.StatusBadRequest, weak.Status)
	require.Contains(t, weak.fields()["new_password"], "capital letter")

	reset := c.post("/api/v1/auth/reset-password", map[string]any{"token": raw, "new_password": "Brand-New-Pass1!"})
	require.Equal(t, http.StatusOK, reset.Status, reset.Raw)

	again := c.post("/api/v1/auth/reset-password", map[string]any{"token": raw, "new_password": "Another-Pass12!"})
	require.Equal(t, http.StatusBadRequest, again.Status, "the link works once")

	login := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": "Brand-New-Pass1!"})
	require.Equal(t, http.StatusOK, login.Status)
}

func TestForgotPassword_UnknownEmailAndBadEmail(t *testing.T) {
	c := newClient(t)

	unknown := c.post("/api/v1/auth/forgot-password", map[string]any{"email": "nobody" + uniq() + "@example.com"})
	require.Equal(t, http.StatusNotFound, unknown.Status)
	require.Equal(t, "NOT_FOUND", unknown.errorCode())
	require.Zero(t, c.e.Mailer.Sent)

	bad := c.post("/api/v1/auth/forgot-password", map[string]any{"email": "nope"})
	require.Equal(t, http.StatusBadRequest, bad.Status)
	require.Contains(t, bad.fields()["email"], "valid email")
}
