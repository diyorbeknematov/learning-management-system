package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// school is a published course with lessons, made with the services; the tests
// then use the API on top of it.
type school struct {
	*client
	owner   *models.User
	id      uuid.UUID
	title   string
	lessons []uuid.UUID
}

func (c *client) newSchool(price float64, lessons int) school {
	c.t.Helper()

	ctx := c.t.Context()
	owner := c.e.NewUser(c.t, models.RoleInstructor)

	admin := models.Actor{UserID: uuid.New(), RoleName: models.RoleSuperAdmin}
	title := "Course " + uniq()

	course, err := c.e.Svc.Course.Create(ctx, admin, models.CreateCourseRequest{
		InstructorID: &owner.ID,
		CategoryID:   c.e.NewCategory(c.t).ID,
		Title:        title,
		Price:        price,
	})
	require.NoError(c.t, err)

	c.e.CleanupCourse(c.t, course.ID)

	actor := models.Actor{UserID: owner.ID, RoleName: models.RoleInstructor}

	module, err := c.e.Svc.Module.Create(ctx, actor, course.ID, models.CreateModuleRequest{Title: "Module"})
	require.NoError(c.t, err)

	ids := make([]uuid.UUID, lessons)

	for i := range ids {
		lesson, err := c.e.Svc.Lesson.Create(ctx, actor, module.ID, models.CreateLessonRequest{Title: fmt.Sprintf("Lesson %d", i+1)})
		require.NoError(c.t, err)

		ids[i] = lesson.ID
	}

	err = c.e.Svc.Course.UpdateStatus(ctx, actor, models.UpdateCourseStatus{ID: course.ID, Status: models.CourseStatusPublished})
	require.NoError(c.t, err)

	return school{client: c, owner: owner, id: course.ID, title: title, lessons: ids}
}

func (s school) path(suffix string) string {
	return "/api/v1/courses/" + s.id.String() + suffix
}

// enrolled creates a student and enrolls them through the API.
func (s school) enrolled() *models.User {
	s.t.Helper()

	student := s.e.NewUser(s.t, models.RoleStudent)

	got := s.as(student, http.MethodPost, s.path("/enrollments"), nil)
	require.Equal(s.t, http.StatusCreated, got.Status, got.Raw)

	return student
}

func (s school) mark(student *models.User, lessonID uuid.UUID, completed bool) answer {
	s.t.Helper()

	return s.as(student, http.MethodPost, "/api/v1/lessons/"+lessonID.String()+"/progress", map[string]any{"completed": completed})
}

func TestEnroll(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := c.e.NewUser(t, models.RoleStudent)

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPost, s.path("/enrollments"), nil, "").Status)
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodPost, s.path("/enrollments"), nil).Status, "an instructor does not enroll")

	got := c.as(student, http.MethodPost, s.path("/enrollments"), nil)
	require.Equal(t, http.StatusCreated, got.Status, got.Raw)
	require.Equal(t, "active", got.data()["status"])
	require.Equal(t, student.ID.String(), got.data()["student_id"])

	again := c.as(student, http.MethodPost, s.path("/enrollments"), nil)
	require.Equal(t, http.StatusConflict, again.Status)

	unknown := c.as(student, http.MethodPost, "/api/v1/courses/"+uuid.NewString()+"/enrollments", nil)
	require.Equal(t, http.StatusNotFound, unknown.Status)

	bad := c.as(student, http.MethodPost, "/api/v1/courses/nope/enrollments", nil)
	require.Equal(t, http.StatusBadRequest, bad.Status)
}

func TestEnroll_PaidCourseKeepsThePayment(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(75.5, 1)

	student := c.e.NewUser(t, models.RoleStudent)

	enrolled := c.as(student, http.MethodPost, s.path("/enrollments"), nil)
	require.Equal(t, http.StatusCreated, enrolled.Status)

	payment := c.as(student, http.MethodGet, "/api/v1/enrollments/"+enrolled.data()["id"].(string)+"/payment", nil)
	require.Equal(t, http.StatusOK, payment.Status, payment.Raw)
	require.EqualValues(t, 75.5, payment.data()["amount"])
	require.Equal(t, "paid", payment.data()["status"])

	// the instructor of the course has no access to the money
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodGet, "/api/v1/enrollments/"+enrolled.data()["id"].(string)+"/payment", nil).Status)
}

func TestEnrollment_ReadAndLeave(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := c.e.NewUser(t, models.RoleStudent)
	other := c.e.NewUser(t, models.RoleStudent)

	enrolled := c.as(student, http.MethodPost, s.path("/enrollments"), nil)
	path := "/api/v1/enrollments/" + enrolled.data()["id"].(string)

	require.Equal(t, http.StatusOK, c.as(student, http.MethodGet, path, nil).Status)
	require.Equal(t, http.StatusOK, c.as(s.owner, http.MethodGet, path, nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(other, http.MethodGet, path, nil).Status)

	require.Equal(t, http.StatusForbidden, c.as(other, http.MethodPatch, path+"/status", map[string]any{"status": "dropped"}).Status)

	completed := c.as(student, http.MethodPatch, path+"/status", map[string]any{"status": "completed"})
	require.Equal(t, http.StatusBadRequest, completed.Status, "completed cannot be set by hand")

	invalid := c.as(student, http.MethodPatch, path+"/status", map[string]any{"status": "gone"})
	require.Equal(t, http.StatusBadRequest, invalid.Status)
	require.Contains(t, invalid.fields()["status"], "one of")

	left := c.as(student, http.MethodPatch, path+"/status", map[string]any{"status": "dropped"})
	require.Equal(t, http.StatusOK, left.Status, left.Raw)
	require.Equal(t, "dropped", c.as(student, http.MethodGet, path, nil).data()["status"])
}

func TestEnrollment_RosterIsForTheOwner(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := s.enrolled()

	roster := c.as(s.owner, http.MethodGet, s.path("/enrollments"), nil)
	require.Equal(t, http.StatusOK, roster.Status, roster.Raw)
	require.EqualValues(t, 1, roster.data()["total"])

	dropped := c.as(s.owner, http.MethodGet, s.path("/enrollments?status=dropped"), nil)
	require.EqualValues(t, 0, dropped.data()["total"])

	require.Equal(t, http.StatusBadRequest, c.as(s.owner, http.MethodGet, s.path("/enrollments?status=gone"), nil).Status)

	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, s.path("/enrollments"), nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleInstructor), http.MethodGet, s.path("/enrollments"), nil).Status)
}

func TestEnrollment_MyDashboard(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 2)

	student := s.enrolled()
	require.Equal(t, http.StatusOK, s.mark(student, s.lessons[0], true).Status)

	got := c.as(student, http.MethodGet, "/api/v1/enrollments/me", nil)
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.EqualValues(t, 1, got.data()["total"])

	item := got.data()["items"].([]any)[0].(map[string]any)
	require.Equal(t, s.title, item["course_title"])
	require.EqualValues(t, 2, item["total_lessons"])
	require.EqualValues(t, 1, item["completed_lessons"])
	require.EqualValues(t, 50, item["progress_percent"])

	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodGet, "/api/v1/enrollments/me", nil).Status)
}

func TestProgress(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 2)

	student := s.enrolled()
	outsider := c.e.NewUser(t, models.RoleStudent)
	lesson := "/api/v1/lessons/" + s.lessons[0].String() + "/progress"

	missing := c.as(student, http.MethodPost, lesson, map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status, "a missing completed is not the same as false")
	require.Contains(t, missing.fields(), "completed")

	done := s.mark(student, s.lessons[0], true)
	require.Equal(t, http.StatusOK, done.Status, done.Raw)
	require.Equal(t, true, done.data()["completed"])

	got := c.as(student, http.MethodGet, lesson, nil)
	require.Equal(t, true, got.data()["completed"])

	untouched := c.as(student, http.MethodGet, "/api/v1/lessons/"+s.lessons[1].String()+"/progress", nil)
	require.Equal(t, false, untouched.data()["completed"])

	course := c.as(student, http.MethodGet, s.path("/progress"), nil)
	require.Equal(t, http.StatusOK, course.Status, course.Raw)
	require.EqualValues(t, 2, course.data()["total_lessons"])
	require.EqualValues(t, 1, course.data()["completed_lessons"])
	require.EqualValues(t, 50, course.data()["progress_percent"])
	require.Len(t, course.data()["completed_lesson_ids"], 1)

	require.Equal(t, http.StatusForbidden, s.mark(outsider, s.lessons[0], true).Status, "a student who did not enroll")
	require.Equal(t, http.StatusForbidden, c.as(outsider, http.MethodGet, s.path("/progress"), nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodPost, lesson, map[string]any{"completed": true}).Status)

	unknown := c.as(student, http.MethodPost, "/api/v1/lessons/"+uuid.NewString()+"/progress", map[string]any{"completed": true})
	require.Equal(t, http.StatusNotFound, unknown.Status)
}

func TestProgress_InstructorSeesTheStudents(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 2)

	student := s.enrolled()
	require.Equal(t, http.StatusOK, s.mark(student, s.lessons[0], true).Status)

	list := c.as(s.owner, http.MethodGet, s.path("/students"), nil)
	require.Equal(t, http.StatusOK, list.Status, list.Raw)
	require.EqualValues(t, 1, list.data()["total"])

	item := list.data()["items"].([]any)[0].(map[string]any)
	require.Equal(t, student.ID.String(), item["student_id"])
	require.EqualValues(t, 50, item["progress_percent"])

	one := c.as(s.owner, http.MethodGet, s.path("/students/"+student.ID.String()+"/progress"), nil)
	require.Equal(t, http.StatusOK, one.Status, one.Raw)
	require.EqualValues(t, 1, one.data()["completed_lessons"])

	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, s.path("/students"), nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleInstructor), http.MethodGet, s.path("/students"), nil).Status)
	require.Equal(t, http.StatusNotFound, c.as(s.owner, http.MethodGet, s.path("/students/"+uuid.NewString()+"/progress"), nil).Status)
	require.Equal(t, http.StatusBadRequest, c.as(s.owner, http.MethodGet, s.path("/students/nope/progress"), nil).Status)
}

func TestCompletionGivesACertificateAndAllowsAReview(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 2)

	student := s.enrolled()
	review := map[string]any{"rating": 5, "comment": "Great course"}

	early := c.as(student, http.MethodPost, s.path("/reviews"), review)
	require.Equal(t, http.StatusForbidden, early.Status, "a review needs a completed course")

	require.Empty(t, c.dataList(c.as(student, http.MethodGet, "/api/v1/certificates/me", nil)))

	s.mark(student, s.lessons[0], true)
	require.Equal(t, http.StatusOK, s.mark(student, s.lessons[1], true).Status)

	certificates := c.dataList(c.as(student, http.MethodGet, "/api/v1/certificates/me", nil))
	require.Len(t, certificates, 1)

	uniqueID := certificates[0]["unique_id"].(string)
	require.Regexp(t, `^LMS(-[A-Z2-9]{4}){3}$`, uniqueID)
	require.Equal(t, s.title, certificates[0]["course_title"])

	byID := c.as(student, http.MethodGet, "/api/v1/certificates/"+certificates[0]["id"].(string), nil)
	require.Equal(t, http.StatusOK, byID.Status)

	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleStudent), http.MethodGet, "/api/v1/certificates/"+certificates[0]["id"].(string), nil).Status)

	// anybody can check it, with no token: that is what the QR code opens
	verified := c.send(http.MethodGet, "/api/v1/certificates/verify/"+uniqueID, nil, "")
	require.Equal(t, http.StatusOK, verified.Status, verified.Raw)
	require.Equal(t, true, verified.data()["valid"])
	require.Equal(t, s.title, verified.data()["course_title"])

	unknown := c.send(http.MethodGet, "/api/v1/certificates/verify/LMS-AAAA-BBBB-CCCC", nil, "")
	require.Equal(t, http.StatusOK, unknown.Status, "an unknown id is an answer, not an error")
	require.Equal(t, false, unknown.data()["valid"])

	created := c.as(student, http.MethodPost, s.path("/reviews"), review)
	require.Equal(t, http.StatusCreated, created.Status, created.Raw)
	require.Equal(t, 5.0, created.data()["rating"])

	require.Equal(t, http.StatusConflict, c.as(student, http.MethodPost, s.path("/reviews"), review).Status)
}

func TestReviews(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	first, second := s.enrolled(), s.enrolled()
	require.Equal(t, http.StatusOK, s.mark(first, s.lessons[0], true).Status)
	require.Equal(t, http.StatusOK, s.mark(second, s.lessons[0], true).Status)

	bad := c.as(first, http.MethodPost, s.path("/reviews"), map[string]any{"rating": 9})
	require.Equal(t, http.StatusBadRequest, bad.Status)
	require.Contains(t, bad.fields()["rating"], "at most 5")

	missing := c.as(first, http.MethodPost, s.path("/reviews"), map[string]any{})
	require.Contains(t, missing.fields(), "rating")

	one := c.as(first, http.MethodPost, s.path("/reviews"), map[string]any{"rating": 4})
	require.Equal(t, http.StatusCreated, one.Status)
	c.as(second, http.MethodPost, s.path("/reviews"), map[string]any{"rating": 5})

	// the reviews of a published course are public
	list := c.send(http.MethodGet, s.path("/reviews"), nil, "")
	require.Equal(t, http.StatusOK, list.Status, list.Raw)
	require.EqualValues(t, 2, list.data()["total"])
	require.EqualValues(t, 4.5, list.data()["avg_rating"])

	path := "/api/v1/reviews/" + one.data()["id"].(string)

	updated := c.as(first, http.MethodPut, path, map[string]any{"rating": 3})
	require.Equal(t, http.StatusOK, updated.Status)
	require.EqualValues(t, 3, updated.data()["rating"])

	require.Equal(t, http.StatusForbidden, c.as(second, http.MethodPut, path, map[string]any{"rating": 1}).Status)
	require.Equal(t, http.StatusBadRequest, c.as(first, http.MethodPut, path, map[string]any{"rating": 0}).Status)

	require.Equal(t, http.StatusForbidden, c.as(second, http.MethodDelete, path, nil).Status)
	require.Equal(t, http.StatusForbidden, c.as(s.owner, http.MethodDelete, path, nil).Status, "the instructor does not delete reviews of their course")

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	require.Equal(t, http.StatusNoContent, c.as(admin, http.MethodDelete, path, nil).Status, "the SuperAdmin moderates")
	require.Equal(t, http.StatusNotFound, c.as(first, http.MethodDelete, path, nil).Status)
}

func TestCertificate_Download(t *testing.T) {
	c := newClient(t)
	s := c.newSchool(0, 1)

	student := s.enrolled()
	require.Equal(t, http.StatusOK, s.mark(student, s.lessons[0], true).Status)

	certificates := c.dataList(c.as(student, http.MethodGet, "/api/v1/certificates/me", nil))
	require.Len(t, certificates, 1)

	path := "/api/v1/certificates/" + certificates[0]["id"].(string) + "/download"

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodGet, path, nil, "").Status)
	require.Equal(t, http.StatusForbidden, c.as(c.e.NewUser(t, models.RoleStudent), http.MethodGet, path, nil).Status)

	got := c.as(student, http.MethodGet, path, nil)
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, "application/pdf", got.Headers.Get("Content-Type"))
	require.Equal(t, `attachment; filename="certificate-`+certificates[0]["unique_id"].(string)+`.pdf"`, got.Headers.Get("Content-Disposition"))
	require.True(t, strings.HasPrefix(got.Raw, "%PDF-"))

	require.Equal(t, http.StatusOK, c.as(s.owner, http.MethodGet, path, nil).Status)

	unknown := c.as(student, http.MethodGet, "/api/v1/certificates/"+uuid.NewString()+"/download", nil)
	require.Equal(t, http.StatusNotFound, unknown.Status)
	require.Equal(t, "NOT_FOUND", unknown.errorCode())
}
