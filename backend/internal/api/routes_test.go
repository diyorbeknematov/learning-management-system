package api_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"testing"

	"github.com/diyorbeknematov/lms/internal/api"
	"github.com/diyorbeknematov/lms/internal/api/authz"
	"github.com/diyorbeknematov/lms/internal/service"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/stretchr/testify/require"
)

// publicRoutes are the routes that need no access token (a token is optional).
// Everything else must be in the Casbin policy; a route that is in neither
// place cannot be called by anybody, which is almost always a mistake.
var publicRoutes = map[string]bool{
	"GET /health": true,

	// the interactive documentation (not served in production)
	"GET /swagger/*any": true,

	"POST /api/v1/auth/register":        true,
	"POST /api/v1/auth/login":           true,
	"POST /api/v1/auth/refresh":         true,
	"POST /api/v1/auth/logout":          true,
	"POST /api/v1/auth/forgot-password": true,
	"POST /api/v1/auth/reset-password":  true,

	"GET /api/v1/categories":     true,
	"GET /api/v1/categories/:id": true,

	"GET /api/v1/courses":                     true,
	"GET /api/v1/courses/:courseId":           true,
	"GET /api/v1/courses/:courseId/modules":   true,
	"GET /api/v1/modules/:moduleId":           true,
	"GET /api/v1/modules/:moduleId/lessons":   true,
	"GET /api/v1/lessons/:lessonId":           true,
	"GET /api/v1/lessons/:lessonId/materials": true,
	"GET /api/v1/materials/:materialId":       true,

	"GET /api/v1/courses/:courseId/reviews":     true,
	"GET /api/v1/certificates/verify/:uniqueId": true,
}

func routes(t *testing.T) []string {
	t.Helper()

	enforcer, err := authz.New()
	require.NoError(t, err)

	router := api.NewRouter(api.Dependencies{
		Service:  service.New(service.Dependencies{}),
		Tokens:   token.NewManager("test-secret", 1),
		Enforcer: enforcer,
		Logger:   slog.New(slog.DiscardHandler),
	})

	var all []string

	for _, route := range router.Routes() {
		all = append(all, route.Method+" "+route.Path)
	}

	sort.Strings(all)

	return all
}

func TestEveryRouteIsPublicOrInThePolicy(t *testing.T) {
	enforcer, err := authz.New()
	require.NoError(t, err)

	for _, route := range routes(t) {
		if publicRoutes[route] {
			continue
		}

		var method, path string

		_, err := fmt.Sscanf(route, "%s %s", &method, &path)
		require.NoError(t, err)

		covered := false

		for _, role := range []string{"Student", "Instructor", "SuperAdmin"} {
			ok, err := enforcer.Enforce(role, path, method)
			require.NoError(t, err)

			covered = covered || ok
		}

		require.True(t, covered, "%s is neither public nor in authz/policy.csv: nobody can call it", route)
	}
}

func TestPublicRoutesAreNotInThePolicy(t *testing.T) {
	enforcer, err := authz.New()
	require.NoError(t, err)

	for _, route := range routes(t) {
		if !publicRoutes[route] {
			continue
		}

		var method, path string

		_, err := fmt.Sscanf(route, "%s %s", &method, &path)
		require.NoError(t, err)

		for _, role := range []string{"Student", "Instructor", "SuperAdmin"} {
			ok, err := enforcer.Enforce(role, path, method)
			require.NoError(t, err)
			require.False(t, ok, "%s is public, so it must not be in the policy as well (%s)", route, role)
		}
	}
}

func TestMeAndIdRoutesLiveTogether(t *testing.T) {
	// "me" is a fixed word and ":id" a parameter on the same place of the
	// address; Gin must keep both
	all := routes(t)

	require.Contains(t, all, http.MethodGet+" /api/v1/users/me")
	require.Contains(t, all, http.MethodGet+" /api/v1/users/:id")
}

// notWrittenYet are routes the policy already lists but the API does not have
// yet. There are none now; a new idea can be put here until its handler is
// written.
var notWrittenYet = map[string]bool{}

func TestEveryPolicyLineHasARoute(t *testing.T) {
	enforcer, err := authz.New()
	require.NoError(t, err)

	registered := map[string]bool{}
	for _, route := range routes(t) {
		registered[route] = true
	}

	policy, err := enforcer.GetPolicy()
	require.NoError(t, err)

	for _, line := range policy {
		// a policy line is "<role>, <path>, <method>"; the router lists "<method> <path>"
		route := line[2] + " " + line[1]

		if notWrittenYet[route] {
			continue
		}

		require.True(t, registered[route], "the policy has %s for %s, but the router has no such route (a typo in a parameter name?)", route, line[0])
	}
}
