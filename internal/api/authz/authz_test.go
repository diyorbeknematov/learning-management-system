package authz_test

import (
	"net/http"
	"testing"

	"github.com/diyorbeknematov/lms/internal/api/authz"
	"github.com/stretchr/testify/require"
)

type route struct {
	method  string
	pattern string
}

func (r route) String() string { return r.method + " " + r.pattern }

const (
	student    = "Student"
	instructor = "Instructor"
	admin      = "SuperAdmin"
)

func allowed(t *testing.T, role string, r route) bool {
	t.Helper()

	enforcer, err := authz.New()
	require.NoError(t, err)

	ok, err := enforcer.Enforce(role, r.pattern, r.method)
	require.NoError(t, err)

	return ok
}

func TestPolicyLoads(t *testing.T) {
	enforcer, err := authz.New()
	require.NoError(t, err)

	policies, err := enforcer.GetPolicy()
	require.NoError(t, err)
	require.NotEmpty(t, policies)
}

// who lists the roles that may call each route. Every role not listed must be
// turned away.
func TestPolicyMatrix(t *testing.T) {
	everybody := []string{student, instructor, admin}
	staff := []string{instructor, admin}

	cases := []struct {
		route route
		who   []string
	}{
		// everybody who is logged in
		{route{http.MethodGet, "/api/v1/users/me"}, everybody},
		{route{http.MethodPut, "/api/v1/users/me"}, everybody},
		{route{http.MethodPut, "/api/v1/users/me/password"}, everybody},
		{route{http.MethodPost, "/api/v1/uploads/presign"}, everybody},
		{route{http.MethodGet, "/api/v1/enrollments/:enrollmentId"}, everybody},
		{route{http.MethodPatch, "/api/v1/enrollments/:enrollmentId/status"}, everybody},
		{route{http.MethodGet, "/api/v1/quizzes/:quizId"}, everybody},
		{route{http.MethodGet, "/api/v1/attempts/:attemptId"}, everybody},
		{route{http.MethodGet, "/api/v1/certificates/:certificateId"}, everybody},
		{route{http.MethodDelete, "/api/v1/reviews/:reviewId"}, everybody},
		{route{http.MethodGet, "/api/v1/payments/:paymentId"}, everybody},

		// the SuperAdmin only
		{route{http.MethodGet, "/api/v1/users"}, []string{admin}},
		{route{http.MethodPost, "/api/v1/users"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/users/:id"}, []string{admin}},
		{route{http.MethodPut, "/api/v1/users/:id"}, []string{admin}},
		{route{http.MethodDelete, "/api/v1/users/:id"}, []string{admin}},
		{route{http.MethodPatch, "/api/v1/users/:id/status"}, []string{admin}},
		{route{http.MethodPost, "/api/v1/categories"}, []string{admin}},
		{route{http.MethodPut, "/api/v1/categories/:id"}, []string{admin}},
		{route{http.MethodDelete, "/api/v1/categories/:id"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/payments"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/finance"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/finance/revenue"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/finance/expenses"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/enrollments"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/revenue"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/students"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/progress"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/quizzes"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/certificates"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/instructors"}, []string{admin}},
		{route{http.MethodGet, "/api/v1/reports/reviews"}, []string{admin}},

		// the people who make the courses: an instructor, and the SuperAdmin
		{route{http.MethodPost, "/api/v1/courses"}, staff},
		{route{http.MethodPut, "/api/v1/courses/:courseId"}, staff},
		{route{http.MethodDelete, "/api/v1/courses/:courseId"}, staff},
		{route{http.MethodPatch, "/api/v1/courses/:courseId/status"}, staff},
		{route{http.MethodPost, "/api/v1/courses/:courseId/modules"}, staff},
		{route{http.MethodPut, "/api/v1/modules/:moduleId"}, staff},
		{route{http.MethodDelete, "/api/v1/modules/:moduleId"}, staff},
		{route{http.MethodPatch, "/api/v1/modules/:moduleId/order"}, staff},
		{route{http.MethodPost, "/api/v1/modules/:moduleId/lessons"}, staff},
		{route{http.MethodPut, "/api/v1/lessons/:lessonId"}, staff},
		{route{http.MethodDelete, "/api/v1/lessons/:lessonId"}, staff},
		{route{http.MethodPatch, "/api/v1/lessons/:lessonId/order"}, staff},
		{route{http.MethodPost, "/api/v1/lessons/:lessonId/materials"}, staff},
		{route{http.MethodPut, "/api/v1/materials/:materialId"}, staff},
		{route{http.MethodDelete, "/api/v1/materials/:materialId"}, staff},
		{route{http.MethodGet, "/api/v1/courses/:courseId/enrollments"}, staff},
		{route{http.MethodGet, "/api/v1/courses/:courseId/students"}, staff},
		{route{http.MethodGet, "/api/v1/courses/:courseId/students/:studentId/progress"}, staff},
		{route{http.MethodPost, "/api/v1/courses/:courseId/quizzes"}, staff},
		{route{http.MethodPost, "/api/v1/modules/:moduleId/quizzes"}, staff},
		{route{http.MethodPut, "/api/v1/quizzes/:quizId"}, staff},
		{route{http.MethodDelete, "/api/v1/quizzes/:quizId"}, staff},
		{route{http.MethodGet, "/api/v1/quizzes/:quizId/questions"}, staff},
		{route{http.MethodPost, "/api/v1/quizzes/:quizId/questions"}, staff},
		{route{http.MethodGet, "/api/v1/questions/:questionId"}, staff},
		{route{http.MethodPut, "/api/v1/questions/:questionId"}, staff},
		{route{http.MethodDelete, "/api/v1/questions/:questionId"}, staff},

		// the students: the SuperAdmin does not study
		{route{http.MethodPost, "/api/v1/courses/:courseId/enrollments"}, []string{student}},
		{route{http.MethodGet, "/api/v1/enrollments/me"}, []string{student}},
		{route{http.MethodGet, "/api/v1/courses/:courseId/progress"}, []string{student}},
		{route{http.MethodGet, "/api/v1/lessons/:lessonId/progress"}, []string{student}},
		{route{http.MethodPost, "/api/v1/lessons/:lessonId/progress"}, []string{student}},
		{route{http.MethodPost, "/api/v1/quizzes/:quizId/attempts"}, []string{student}},
		{route{http.MethodPost, "/api/v1/attempts/:attemptId/submit"}, []string{student}},
		{route{http.MethodGet, "/api/v1/certificates/me"}, []string{student}},
		{route{http.MethodPost, "/api/v1/courses/:courseId/reviews"}, []string{student}},
		{route{http.MethodPut, "/api/v1/reviews/:reviewId"}, []string{student}},
	}

	for _, tt := range cases {
		for _, role := range []string{student, instructor, admin} {
			want := false

			for _, who := range tt.who {
				if who == role {
					want = true
				}
			}

			require.Equal(t, want, allowed(t, role, tt.route), "%s on %s", role, tt.route)
		}
	}
}

func TestPolicy_UnknownRolesAndRoutesAreClosed(t *testing.T) {
	require.False(t, allowed(t, "", route{http.MethodGet, "/api/v1/users/me"}))
	require.False(t, allowed(t, "Hacker", route{http.MethodGet, "/api/v1/users/me"}))
	require.False(t, allowed(t, "user", route{http.MethodGet, "/api/v1/users/me"}), "role names are exact")
	require.False(t, allowed(t, admin, route{http.MethodGet, "/api/v1/not-in-the-policy"}))
	require.False(t, allowed(t, admin, route{http.MethodTrace, "/api/v1/users/me"}))
}

func TestPolicy_TheMethodMatters(t *testing.T) {
	require.True(t, allowed(t, admin, route{http.MethodGet, "/api/v1/users/:id"}))
	require.False(t, allowed(t, admin, route{http.MethodPost, "/api/v1/users/:id"}))
	require.False(t, allowed(t, instructor, route{http.MethodPatch, "/api/v1/courses/:courseId"}))
	require.False(t, allowed(t, student, route{http.MethodDelete, "/api/v1/courses/:courseId/enrollments"}))
}

func TestPolicy_AParameterNameIsPartOfTheRoute(t *testing.T) {
	// the policy is matched against the pattern of the router, so a route
	// declared with another parameter name is not covered by accident
	require.False(t, allowed(t, admin, route{http.MethodGet, "/api/v1/users/:userId"}))
}
