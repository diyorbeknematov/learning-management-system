package api_test

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/stretchr/testify/require"
)

// moneyRoutes are the routes only the SuperAdmin may call.
func moneyRoutes(courseID string) []string {
	return []string{
		"/api/v1/payments",
		"/api/v1/finance",
		"/api/v1/finance/revenue",
		"/api/v1/finance/expenses",
		"/api/v1/reports/enrollments?course_id=" + courseID,
		"/api/v1/reports/revenue",
		"/api/v1/reports/students",
		"/api/v1/reports/progress",
		"/api/v1/reports/quizzes",
		"/api/v1/reports/certificates",
		"/api/v1/reports/instructors",
		"/api/v1/reports/reviews",
	}
}

func TestMoney_OnlyTheSuperAdmin(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(10, 1)

	student := s.enrolled()
	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	for _, path := range moneyRoutes(s.id.String()) {
		require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, path, nil, "").Status, path)
		require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, path, nil).Status, path)
		require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodGet, path, nil).Status, path)
		require.Equal(t, http.StatusOK, c.as(admin, http.MethodGet, path, nil).Status, path)
	}
}

func TestFinance(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(10, 1)

	s.enrolled()

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	summary := c.as(admin, http.MethodGet, "/api/v1/finance?group_by=day", nil)
	require.Equal(t, http.StatusOK, summary.Status, summary.Raw)
	require.GreaterOrEqual(t, summary.data()["revenue"].(float64), 10.0, "the payment of today is in the last 30 days")
	require.Contains(t, []any{"profit", "loss"}, summary.data()["status"])
	require.NotNil(t, summary.data()["points"])

	revenue := c.as(admin, http.MethodGet, "/api/v1/finance/revenue", nil)
	require.Equal(t, http.StatusOK, revenue.Status)

	expenses := c.as(admin, http.MethodGet, "/api/v1/finance/expenses", nil)
	require.Equal(t, http.StatusOK, expenses.Status)
	require.NotNil(t, expenses.data()["payouts"])
}

func TestFinance_Dates(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	for _, query := range []string{
		"from=1995-05-01&to=1995-05-31",
		"from=1995-05-01T00:00:00Z&to=1995-05-31T23:59:59Z",
		"from=1995-05-01",
		"to=1995-05-31",
		"group_by=month&from=1995-01-01&to=1995-12-31",
	} {
		got := c.as(admin, http.MethodGet, "/api/v1/finance?"+query, nil)
		require.Equal(t, http.StatusOK, got.Status, query)
	}

	// the whole of the last day counts: 1995-05-31 means until its last moment. 1995 is a year
	// no test writes to, so the answer is always empty
	empty := c.as(admin, http.MethodGet, "/api/v1/finance?from=1995-05-01&to=1995-05-31", nil)
	require.EqualValues(t, 0, empty.data()["revenue"])
}

func TestFinance_BadQuery(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	notADate := c.as(admin, http.MethodGet, "/api/v1/finance?from=yesterday&to=05/31/1995", nil)
	require.Equal(t, http.StatusBadRequest, notADate.Status)
	require.Contains(t, notADate.fields()["from"], "must be a date")
	require.Contains(t, notADate.fields()["to"], "must be a date")

	backwards := c.as(admin, http.MethodGet, "/api/v1/finance?from=1995-06-01&to=1995-05-01", nil)
	require.Equal(t, http.StatusBadRequest, backwards.Status)
	require.Contains(t, backwards.errorMessage(), "from must not be after to")

	badGroup := c.as(admin, http.MethodGet, "/api/v1/finance?group_by=year", nil)
	require.Equal(t, http.StatusBadRequest, badGroup.Status)
	require.Contains(t, badGroup.fields()["group_by"], "one of")
}

func TestPayments(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(40, 1)

	student := c.e.NewUser(t, models.RoleStudent)
	other := c.e.NewUser(t, models.RoleStudent)
	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	enrolled := c.as(student, http.MethodPost, s.path("/enrollments"), nil)
	require.Equal(t, http.StatusCreated, enrolled.Status)

	list := c.as(admin, http.MethodGet, "/api/v1/payments?course_id="+s.id.String(), nil)
	require.Equal(t, http.StatusOK, list.Status, list.Raw)
	require.EqualValues(t, 1, list.data()["total"])

	paymentID := list.data()["items"].([]any)[0].(map[string]any)["id"].(string)

	require.Equal(t, http.StatusOK, c.as(admin, http.MethodGet, "/api/v1/payments/"+paymentID, nil).Status)
	require.Equal(t, http.StatusOK, c.as(student, http.MethodGet, "/api/v1/payments/"+paymentID, nil).Status, "the student sees their own payment")
	require.Equal(t, http.StatusForbidden, c.as(other, http.MethodGet, "/api/v1/payments/"+paymentID, nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodGet, "/api/v1/payments/"+paymentID, nil).Status)

	require.Equal(t, http.StatusBadRequest, c.as(admin, http.MethodGet, "/api/v1/payments?course_id=nope", nil).Status)
	require.Equal(t, http.StatusBadRequest, c.as(admin, http.MethodGet, "/api/v1/payments?from=x", nil).Status)
}

func TestReports_JSON(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(25, 2)

	student := s.enrolled()
	require.Equal(t, http.StatusOK, s.mark(student, s.lessons[0], true).Status)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	course := "?course_id=" + s.id.String()

	enrollments := c.as(admin, http.MethodGet, "/api/v1/reports/enrollments"+course+"&group_by=month", nil)
	require.Equal(t, http.StatusOK, enrollments.Status, enrollments.Raw)
	require.Len(t, enrollments.data()["rows"], 1)
	require.Len(t, enrollments.data()["trend"], 1)

	revenue := c.as(admin, http.MethodGet, "/api/v1/reports/revenue"+course, nil)
	require.Equal(t, http.StatusOK, revenue.Status)
	rows := c.dataList(revenue)
	require.Len(t, rows, 1)
	require.EqualValues(t, 25, rows[0]["revenue"])

	progress := c.as(admin, http.MethodGet, "/api/v1/reports/progress"+course, nil)
	require.Equal(t, http.StatusOK, progress.Status)
	require.Len(t, progress.data()["funnel"], 2, "a report of one course has the funnel of its lessons")

	all := c.as(admin, http.MethodGet, "/api/v1/reports/progress", nil)
	require.Empty(t, all.data()["funnel"])
}

func TestReports_CSV(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(25, 1)

	s.enrolled()

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	got := c.as(admin, http.MethodGet, "/api/v1/reports/revenue?course_id="+s.id.String()+"&format=csv", nil)
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, "text/csv; charset=utf-8", got.Headers.Get("Content-Type"))
	require.Contains(t, got.Headers.Get("Content-Disposition"), `attachment; filename="revenue-`)
	require.Contains(t, got.Headers.Get("Content-Disposition"), `.csv"`)

	require.True(t, strings.HasPrefix(got.Raw, "\xEF\xBB\xBF"), "a byte order mark, so Excel reads the letters right")

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(got.Raw, "\xEF\xBB\xBF"))).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2, "the header and the course")
	require.Equal(t, []string{"course_id", "course_title", "paid_enrollments", "free_enrollments", "revenue", "avg_enrollment_value"}, rows[0])
	require.Equal(t, s.id.String(), rows[1][0])
	require.Equal(t, s.title, rows[1][1])
	require.Equal(t, "25", rows[1][4])
}

func TestReports_CSVHasOnlyTheMainTable(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	s.enrolled()

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	got := c.as(admin, http.MethodGet, "/api/v1/reports/enrollments?course_id="+s.id.String()+"&format=csv", nil)
	require.Equal(t, http.StatusOK, got.Status)

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(got.Raw, "\xEF\xBB\xBF"))).ReadAll()
	require.NoError(t, err)
	require.Equal(t, "course_id", rows[0][0])
	require.Len(t, rows, 2)
}

func TestReports_BadQuery(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	for _, query := range []string{"format=xml", "course_id=nope", "group_by=year", "from=x", "from=1995-06-01&to=1995-05-01"} {
		got := c.as(admin, http.MethodGet, "/api/v1/reports/revenue?"+query, nil)
		require.Equal(t, http.StatusBadRequest, got.Status, query)
		require.Equal(t, "INVALID_INPUT", got.errorCode(), query)
	}
}
