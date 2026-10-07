package api_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/api"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/stretchr/testify/require"
)

// loginAs sends a login with a username nobody has; the answer is 401, unless
// the limit stops it first.
func (c *client) strangerLogin(n int) answer {
	return c.post("/api/v1/auth/login", map[string]any{"username": fmt.Sprintf("stranger%s%d", uniq(), n), "password": "Wrong-Passw0rd!"})
}

func TestRateLimit_Login(t *testing.T) {
	c := newClient(t)

	for i := 1; i <= 10; i++ {
		require.Equal(t, http.StatusUnauthorized, c.strangerLogin(i).Status, "request %d", i)
	}

	got := c.strangerLogin(11)
	require.Equal(t, http.StatusTooManyRequests, got.Status)
	require.Equal(t, "TOO_MANY_REQUESTS", got.errorCode())
	require.Equal(t, false, got.Body["success"])

	require.NotEmpty(t, got.Headers.Get("Retry-After"))
	require.Contains(t, got.errorMessage(), "seconds")

	// another address, and another route, are not touched
	require.Equal(t, http.StatusUnauthorized, c.from("203.0.113.9").strangerLogin(1).Status)
	require.Equal(t, http.StatusBadRequest, c.post("/api/v1/auth/refresh", map[string]any{}).Status)

	// when the window ends the address can try again
	c.e.Redis.Reset()
	require.Equal(t, http.StatusUnauthorized, c.strangerLogin(12).Status)
}

func TestRateLimit_RegisterAndForgotPassword(t *testing.T) {
	c := newClient(t)

	for i := 1; i <= 5; i++ {
		require.Equal(t, http.StatusBadRequest, c.post("/api/v1/auth/register", map[string]any{}).Status, "request %d", i)
	}

	require.Equal(t, http.StatusTooManyRequests, c.post("/api/v1/auth/register", map[string]any{}).Status)

	for i := 1; i <= 3; i++ {
		got := c.post("/api/v1/auth/forgot-password", map[string]any{"email": fmt.Sprintf("nobody%s%d@example.com", uniq(), i)})
		require.Equal(t, http.StatusNotFound, got.Status, "request %d", i)
	}

	require.Equal(t, http.StatusTooManyRequests, c.post("/api/v1/auth/forgot-password", map[string]any{"email": "one-more@example.com"}).Status)
}

func TestRateLimit_OnlyTheAuthRoutesAreLimited(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	for i := 0; i < 30; i++ {
		require.Equal(t, http.StatusOK, c.as(user, http.MethodGet, "/api/v1/users/me", nil).Status)
		require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/courses", nil, "").Status)
	}
}

func TestRateLimit_AMadeUpForwardedHeaderDoesNotHelp(t *testing.T) {
	c := newClient(t)

	for i := 1; i <= 10; i++ {
		got := c.sendWith(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "x" + uniq(), "password": "Wrong-Passw0rd!"}, "X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
		require.Equal(t, http.StatusUnauthorized, got.Status)
	}

	got := c.sendWith(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "x" + uniq(), "password": "Wrong-Passw0rd!"}, "X-Forwarded-For", "198.51.100.200")
	require.Equal(t, http.StatusTooManyRequests, got.Status, "the header of an unknown sender is not believed")
}

func TestRateLimit_ATrustedProxyTellsTheRealAddress(t *testing.T) {
	// the test requests come from 192.0.2.1, which is declared as the proxy
	c := newClientWith(t, func(d *api.Dependencies) {
		d.TrustedProxies = []string{"192.0.2.1"}
	})

	for i := 1; i <= 15; i++ {
		got := c.sendWith(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "x" + uniq(), "password": "Wrong-Passw0rd!"}, "X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
		require.Equal(t, http.StatusUnauthorized, got.Status, "every client behind the proxy has their own count (request %d)", i)
	}

	// one client behind the proxy is limited like anybody else
	for i := 1; i <= 10; i++ {
		c.sendWith(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "x" + uniq(), "password": "Wrong-Passw0rd!"}, "X-Forwarded-For", "198.51.100.250")
	}

	got := c.sendWith(http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "x" + uniq(), "password": "Wrong-Passw0rd!"}, "X-Forwarded-For", "198.51.100.250")
	require.Equal(t, http.StatusTooManyRequests, got.Status)
}

func TestRateLimit_CanBeTurnedOff(t *testing.T) {
	c := newClientWith(t, func(d *api.Dependencies) {
		d.Limiter = nil
	})

	for i := 1; i <= 25; i++ {
		require.Equal(t, http.StatusUnauthorized, c.strangerLogin(i).Status)
	}
}

func TestRateLimit_AccountLockoutOverHTTP(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	for i := 0; i < 5; i++ {
		got := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": "Wrong-Passw0rd!"})
		require.Equal(t, http.StatusUnauthorized, got.Status)
	}

	locked := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password})
	require.Equal(t, http.StatusTooManyRequests, locked.Status, "even the right password waits")
	require.Equal(t, "TOO_MANY_REQUESTS", locked.errorCode())
}

func TestToken_Revoked_WhenTheUserIsBlockedAndUnblocked(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status)

	blocked := c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusOK, blocked.Status, blocked.Raw)

	got := c.send(http.MethodGet, "/api/v1/users/me", nil, old)
	require.Equal(t, http.StatusUnauthorized, got.Status, "the token still has minutes to live, but the user is out at once")
	require.Contains(t, got.errorMessage(), "log in again")

	require.Equal(t, http.StatusForbidden, c.post("/api/v1/auth/login", map[string]any{"username": student.Username, "password": testutil.Password}).Status)

	unblocked := c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{"status": "active"})
	require.Equal(t, http.StatusOK, unblocked.Status)

	login := c.post("/api/v1/auth/login", map[string]any{"username": student.Username, "password": testutil.Password})
	require.Equal(t, http.StatusOK, login.Status)

	again := c.send(http.MethodGet, "/api/v1/users/me", nil, "Bearer "+login.data()["access_token"].(string))
	require.Equal(t, http.StatusOK, again.Status, "an unblocked user works with a new token")
}

func TestToken_Revoked_WhenTheUserIsDeleted(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	require.Equal(t, http.StatusNoContent, c.as(admin, http.MethodDelete, "/api/v1/users/"+student.ID.String(), nil).Status)

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status)
}

func TestToken_Revoked_WhenThePasswordChanges(t *testing.T) {
	c := newClient(t)

	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	changed := c.send(http.MethodPut, "/api/v1/users/me/password", map[string]any{"old_password": testutil.Password, "new_password": "Brand-New-Pass1!"}, old)
	require.Equal(t, http.StatusOK, changed.Status, changed.Raw)

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status, "a stolen token dies with the old password")

	login := c.post("/api/v1/auth/login", map[string]any{"username": student.Username, "password": "Brand-New-Pass1!"})
	require.Equal(t, http.StatusOK, login.Status)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/users/me", nil, "Bearer "+login.data()["access_token"].(string)).Status)
}

func TestToken_Revoked_WhenTheRoleChanges(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	changed := c.as(admin, http.MethodPut, "/api/v1/users/"+student.ID.String(), map[string]any{"role": "Instructor"})
	require.Equal(t, http.StatusOK, changed.Status, changed.Raw)

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status, "the role is in the token, so it must be a new token")

	login := c.post("/api/v1/auth/login", map[string]any{"username": student.Username, "password": testutil.Password})
	require.Equal(t, http.StatusOK, login.Status)

	token := "Bearer " + login.data()["access_token"].(string)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/users/me", nil, token).Status)

	claims, err := c.e.Tokens.ParseAccessToken(login.data()["access_token"].(string))
	require.NoError(t, err)
	require.Equal(t, models.RoleInstructor, claims.RoleName, "the new token has the new role")
}

func TestToken_NotRevoked_ByAnOrdinaryProfileChange(t *testing.T) {
	c := newClient(t)

	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	changed := c.send(http.MethodPut, "/api/v1/users/me", map[string]any{"bio": "hello"}, old)
	require.Equal(t, http.StatusOK, changed.Status)

	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status)
}

func TestToken_RevocationIsOffWithoutIt(t *testing.T) {
	c := newClientWith(t, func(d *api.Dependencies) {
		d.Revocations = nil
	})

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)
	old := c.bearerAfterASecond(student)

	require.Equal(t, http.StatusOK, c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{"status": "blocked"}).Status)

	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/users/me", nil, old).Status, "without revocations the token lives until it expires")
}

// bearerAfterASecond makes an access token and waits a little more than a
// second, so the token is older than the changes the test makes afterwards: a
// token carries its time in whole seconds.
func (c *client) bearerAfterASecond(user *models.User) string {
	c.t.Helper()

	header := c.bearer(user)

	time.Sleep(1100 * time.Millisecond)

	return header
}

// sendWith is send with one more header.
func (c *client) sendWith(method, path string, body any, name, value string) answer {
	c.t.Helper()

	return c.sendHeaders(method, path, body, "", map[string]string{name: value})
}
