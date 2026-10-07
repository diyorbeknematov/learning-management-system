package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func idOf(t *testing.T, got answer) uuid.UUID {
	t.Helper()

	id, err := uuid.Parse(got.data()["id"].(string))
	require.NoError(t, err, got.Raw)

	return id
}

// createCourse makes a course through the API as the instructor.
func (c *client) createCourse(owner *models.User, categoryID uuid.UUID) uuid.UUID {
	c.t.Helper()

	got := c.as(owner, http.MethodPost, "/api/v1/courses", map[string]any{
		"category_id": categoryID,
		"title":       "Course " + uniq(),
		"price":       20,
	})
	require.Equal(c.t, http.StatusCreated, got.Status, got.Raw)

	id := idOf(c.t, got)
	c.e.CleanupCourse(c.t, id)

	return id
}

func TestCategory_PublicReadsAdminWrites(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)
	name := "cat-" + uniq()

	require.Equal(t, http.StatusUnauthorized, c.post("/api/v1/categories", map[string]any{"name": name}).Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodPost, "/api/v1/categories", map[string]any{"name": name}).Status)

	created := c.as(admin, http.MethodPost, "/api/v1/categories", map[string]any{"name": name, "description": "About Go"})
	require.Equal(t, http.StatusCreated, created.Status, created.Raw)

	id := idOf(t, created)
	t.Cleanup(func() { c.e.Exec(t, `DELETE FROM categories WHERE id = $1`, id) })

	// everybody can read, with no token
	one := c.send(http.MethodGet, "/api/v1/categories/"+id.String(), nil, "")
	require.Equal(t, http.StatusOK, one.Status)
	require.Equal(t, name, one.data()["name"])

	list := c.send(http.MethodGet, "/api/v1/categories?name="+name, nil, "")
	require.Equal(t, http.StatusOK, list.Status)
	require.EqualValues(t, 1, list.data()["total"])

	renamed := c.as(admin, http.MethodPut, "/api/v1/categories/"+id.String(), map[string]any{"name": name + "-2"})
	require.Equal(t, http.StatusOK, renamed.Status)
	require.Equal(t, name+"-2", renamed.data()["name"])

	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodDelete, "/api/v1/categories/"+id.String(), nil).Status)
	require.Equal(t, http.StatusNoContent, c.as(admin, http.MethodDelete, "/api/v1/categories/"+id.String(), nil).Status)
	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, "/api/v1/categories/"+id.String(), nil, "").Status)
}

func TestCategory_Validation(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	existing := c.e.NewCategory(t)

	missing := c.as(admin, http.MethodPost, "/api/v1/categories", map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status)
	require.Contains(t, missing.fields(), "name")

	long := c.as(admin, http.MethodPost, "/api/v1/categories", map[string]any{"name": strings.Repeat("a", 101)})
	require.Equal(t, http.StatusBadRequest, long.Status)
	require.Contains(t, long.fields()["name"], "at most 100")

	duplicate := c.as(admin, http.MethodPost, "/api/v1/categories", map[string]any{"name": existing.Name})
	require.Equal(t, http.StatusConflict, duplicate.Status)

	bad := c.send(http.MethodGet, "/api/v1/categories/not-an-id", nil, "")
	require.Equal(t, http.StatusBadRequest, bad.Status)
}

func TestCourse_CreateNeedsAnInstructor(t *testing.T) {
	c := newClient(t)

	category := c.e.NewCategory(t)
	student := c.e.NewUser(t, models.RoleStudent)
	body := map[string]any{"category_id": category.ID, "title": "Go"}

	require.Equal(t, http.StatusUnauthorized, c.post("/api/v1/courses", body).Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodPost, "/api/v1/courses", body).Status)
}

func TestCourse_Create(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	category := c.e.NewCategory(t)

	got := c.as(owner, http.MethodPost, "/api/v1/courses", map[string]any{
		"category_id":       category.ID,
		"title":             "Go basics",
		"difficulty":        "beginner",
		"price":             49.5,
		"learning_outcomes": []string{"Write Go", "Test Go"},
		"requirements":      []string{"A laptop"},
		"payout_type":       "percentage",
		"payout_value":      30,
	})
	require.Equal(t, http.StatusCreated, got.Status, got.Raw)
	c.e.CleanupCourse(t, idOf(t, got))

	require.Equal(t, owner.ID.String(), got.data()["instructor_id"], "the owner is the one who is logged in")
	require.Equal(t, "draft", got.data()["status"])
	require.Len(t, got.data()["learning_outcomes"], 2)
	require.Nil(t, got.data()["payout_type"], "an instructor does not handle the payout")
	require.NotContains(t, got.Raw, "payout_value\":30")
}

func TestCourse_CreateValidation(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)

	got := c.as(owner, http.MethodPost, "/api/v1/courses", map[string]any{
		"category_id": "not-a-uuid",
		"difficulty":  "expert",
		"price":       -5,
	})
	require.Equal(t, http.StatusBadRequest, got.Status)

	empty := c.as(owner, http.MethodPost, "/api/v1/courses", map[string]any{"difficulty": "expert", "price": -5})
	require.Equal(t, http.StatusBadRequest, empty.Status)
	require.Contains(t, empty.fields(), "category_id")
	require.Contains(t, empty.fields(), "title")
	require.Contains(t, empty.fields()["difficulty"], "one of")
	require.Contains(t, empty.fields()["price"], "at least 0")

	unknown := c.as(owner, http.MethodPost, "/api/v1/courses", map[string]any{"category_id": uuid.New(), "title": "x"})
	require.Equal(t, http.StatusNotFound, unknown.Status, "a category that does not exist")
}

func TestCourse_PublicCatalogShowsOnlyPublished(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	category := c.e.NewCategory(t)

	published := c.e.NewCourse(t, owner, category)
	c.e.Publish(t, owner, published.ID)
	draft := c.e.NewCourse(t, owner, category)

	query := "?category_id=" + category.ID.String()

	visitor := c.send(http.MethodGet, "/api/v1/courses"+query, nil, "")
	require.Equal(t, http.StatusOK, visitor.Status, visitor.Raw)
	require.EqualValues(t, 1, visitor.data()["total"])

	own := c.as(owner, http.MethodGet, "/api/v1/courses"+query+"&instructor_id="+owner.ID.String(), nil)
	require.Equal(t, http.StatusOK, own.Status)
	require.EqualValues(t, 2, own.data()["total"], "the owner sees the draft %s too", draft.ID)

	someone := c.as(c.e.NewUser(t, models.RoleStudent), http.MethodGet, "/api/v1/courses"+query+"&instructor_id="+owner.ID.String(), nil)
	require.EqualValues(t, 1, someone.data()["total"])
}

func TestCourse_CatalogFilters(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	category := c.e.NewCategory(t)
	course := c.e.NewCourse(t, owner, category)
	c.e.Publish(t, owner, course.ID)

	query := "?category_id=" + category.ID.String()

	for _, extra := range []string{"&price_type=paid", "&sort=newest", "&sort=price", "&sort=popular", "&sort=rating", "&min_rating=0", "&limit=5&page=1", "&q=" + course.Title[:6]} {
		got := c.send(http.MethodGet, "/api/v1/courses"+query+extra, nil, "")
		require.Equal(t, http.StatusOK, got.Status, extra)
		require.EqualValues(t, 1, got.data()["total"], extra)
	}

	free := c.send(http.MethodGet, "/api/v1/courses"+query+"&price_type=free", nil, "")
	require.EqualValues(t, 0, free.data()["total"], "the course costs 20")
}

func TestCourse_CatalogBadQuery(t *testing.T) {
	c := newClient(t)

	for _, query := range []string{
		"sort=cheapest", "price_type=cheap", "difficulty=expert", "min_rating=6", "min_rating=-1",
		"category_id=not-a-uuid", "instructor_id=1", "status=archived", "limit=ten",
	} {
		got := c.send(http.MethodGet, "/api/v1/courses?"+query, nil, "")
		require.Equal(t, http.StatusBadRequest, got.Status, query)
		require.Equal(t, "INVALID_INPUT", got.errorCode(), query)
	}
}

func TestCourse_PageHidesDrafts(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	stranger := c.e.NewUser(t, models.RoleInstructor)
	course := c.e.NewCourse(t, owner, c.e.NewCategory(t))

	path := "/api/v1/courses/" + course.ID.String()

	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, path, nil, "").Status)
	require.Equal(t, http.StatusNotFound, c.as(stranger, http.MethodGet, path, nil).Status)
	require.Equal(t, http.StatusOK, c.as(owner, http.MethodGet, path, nil).Status)

	c.e.Publish(t, owner, course.ID)

	got := c.send(http.MethodGet, path, nil, "")
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, course.Title, got.data()["title"])
	require.NotNil(t, got.data()["instructor"])
	require.NotNil(t, got.data()["modules"])
}

func TestCourse_Update(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	stranger := c.e.NewUser(t, models.RoleInstructor)
	course := c.e.NewCourse(t, owner, c.e.NewCategory(t))

	path := "/api/v1/courses/" + course.ID.String()

	got := c.as(owner, http.MethodPut, path, map[string]any{"title": "Renamed", "price": 99})
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.Equal(t, "Renamed", got.data()["title"])

	hijack := c.as(stranger, http.MethodPut, path, map[string]any{"title": "Hijacked"})
	require.Equal(t, http.StatusForbidden, hijack.Status, "an instructor can change only their own course")

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPut, path, map[string]any{"title": "x"}, "").Status)

	bad := c.as(owner, http.MethodPut, path, map[string]any{"price": -1, "difficulty": "expert"})
	require.Equal(t, http.StatusBadRequest, bad.Status)

	require.Equal(t, http.StatusNotFound, c.as(owner, http.MethodPut, "/api/v1/courses/"+uuid.NewString(), map[string]any{"title": "x"}).Status)
}

func TestCourse_PublishAndDelete(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	course := c.e.NewCourse(t, owner, c.e.NewCategory(t))

	path := "/api/v1/courses/" + course.ID.String()
	publish := map[string]any{"status": "published"}

	empty := c.as(owner, http.MethodPatch, path+"/status", publish)
	require.Equal(t, http.StatusBadRequest, empty.Status, "an empty course cannot be published")
	require.Contains(t, empty.errorMessage(), "at least one module")

	invalid := c.as(owner, http.MethodPatch, path+"/status", map[string]any{"status": "archived"})
	require.Equal(t, http.StatusBadRequest, invalid.Status)
	require.Contains(t, invalid.fields()["status"], "one of")

	c.e.AddLesson(t, course.ID)

	require.Equal(t, http.StatusOK, c.as(owner, http.MethodPatch, path+"/status", publish).Status)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, path, nil, "").Status)

	stranger := c.e.NewUser(t, models.RoleInstructor)
	require.Equal(t, http.StatusForbidden, c.as(stranger, http.MethodDelete, path, nil).Status)

	require.Equal(t, http.StatusNoContent, c.as(owner, http.MethodDelete, path, nil).Status)
	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, path, nil, "").Status)
}

func TestSyllabus_ModulesAndLessons(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	stranger := c.e.NewUser(t, models.RoleInstructor)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	modules := "/api/v1/courses/" + courseID.String() + "/modules"

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPost, modules, map[string]any{"title": "x"}, "").Status)
	require.Equal(t, http.StatusForbidden, c.as(stranger, http.MethodPost, modules, map[string]any{"title": "x"}).Status)

	missing := c.as(owner, http.MethodPost, modules, map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status)
	require.Contains(t, missing.fields(), "title")

	first := c.as(owner, http.MethodPost, modules, map[string]any{"title": "Basics"})
	require.Equal(t, http.StatusCreated, first.Status, first.Raw)
	require.EqualValues(t, 1, first.data()["order_number"])

	second := c.as(owner, http.MethodPost, modules, map[string]any{"title": "Advanced"})
	require.EqualValues(t, 2, second.data()["order_number"])

	secondPath := "/api/v1/modules/" + idOf(t, second).String()

	// the second module moves to the first place
	moved := c.as(owner, http.MethodPatch, secondPath+"/order", map[string]any{"order_number": 1})
	require.Equal(t, http.StatusOK, moved.Status, moved.Raw)

	badOrder := c.as(owner, http.MethodPatch, secondPath+"/order", map[string]any{"order_number": 0})
	require.Equal(t, http.StatusBadRequest, badOrder.Status)

	renamed := c.as(owner, http.MethodPut, secondPath, map[string]any{"title": "Advanced Go"})
	require.Equal(t, http.StatusOK, renamed.Status)
	require.Equal(t, "Advanced Go", renamed.data()["title"])

	lessons := secondPath + "/lessons"

	lesson := c.as(owner, http.MethodPost, lessons, map[string]any{"title": "Generics", "duration": 15, "is_preview": true})
	require.Equal(t, http.StatusCreated, lesson.Status, lesson.Raw)

	lessonPath := "/api/v1/lessons/" + idOf(t, lesson).String()

	c.e.Exec(t, `UPDATE courses SET status = 'published' WHERE id = $1`, courseID)

	// the syllabus is open to visitors when the course is published
	list := c.send(http.MethodGet, modules, nil, "")
	require.Equal(t, http.StatusOK, list.Status, list.Raw)

	var titles []string
	for _, module := range c.dataList(list) {
		titles = append(titles, module["title"].(string))
	}

	require.Equal(t, []string{"Advanced Go", "Basics"}, titles, "in the new order")

	require.Equal(t, http.StatusOK, c.send(http.MethodGet, secondPath, nil, "").Status)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, lessons, nil, "").Status)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, lessonPath, nil, "").Status)

	badDuration := c.as(owner, http.MethodPut, lessonPath, map[string]any{"duration": "long"})
	require.Equal(t, http.StatusBadRequest, badDuration.Status)

	require.Equal(t, http.StatusNoContent, c.as(owner, http.MethodDelete, lessonPath, nil).Status)
	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, lessonPath, nil, "").Status)

	require.Equal(t, http.StatusNoContent, c.as(owner, http.MethodDelete, secondPath, nil).Status)
	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, secondPath, nil, "").Status)
}

// dataList reads a "data" that is a list.
func (c *client) dataList(got answer) []map[string]any {
	c.t.Helper()

	raw, ok := got.Body["data"].([]any)
	require.True(c.t, ok, got.Raw)

	items := make([]map[string]any, len(raw))
	for i, item := range raw {
		items[i] = item.(map[string]any)
	}

	return items
}

func TestSyllabus_DraftIsHidden(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	module := c.as(owner, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/modules", map[string]any{"title": "A"})
	require.Equal(t, http.StatusCreated, module.Status)

	path := "/api/v1/courses/" + courseID.String() + "/modules"

	require.Equal(t, http.StatusNotFound, c.send(http.MethodGet, path, nil, "").Status)
	require.Equal(t, http.StatusOK, c.as(owner, http.MethodGet, path, nil).Status)
}

func TestMaterials_Access(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	student := c.e.NewUser(t, models.RoleStudent)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	module := c.as(owner, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/modules", map[string]any{"title": "A"})
	moduleID := idOf(t, module)

	preview := c.as(owner, http.MethodPost, "/api/v1/modules/"+moduleID.String()+"/lessons", map[string]any{"title": "Free", "is_preview": true})
	closed := c.as(owner, http.MethodPost, "/api/v1/modules/"+moduleID.String()+"/lessons", map[string]any{"title": "Paid"})

	previewMaterials := "/api/v1/lessons/" + idOf(t, preview).String() + "/materials"
	closedMaterials := "/api/v1/lessons/" + idOf(t, closed).String() + "/materials"

	text := map[string]any{"type": "text", "content": "Read this"}

	require.Equal(t, http.StatusUnauthorized, c.send(http.MethodPost, previewMaterials, text, "").Status)
	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodPost, previewMaterials, text).Status)

	freeMaterial := c.as(owner, http.MethodPost, previewMaterials, text)
	require.Equal(t, http.StatusCreated, freeMaterial.Status, freeMaterial.Raw)

	paidMaterial := c.as(owner, http.MethodPost, closedMaterials, text)
	require.Equal(t, http.StatusCreated, paidMaterial.Status)

	c.e.Exec(t, `UPDATE courses SET status = 'published' WHERE id = $1`, courseID)

	// a visitor opens the preview only
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, previewMaterials, nil, "").Status)
	require.Equal(t, http.StatusOK, c.send(http.MethodGet, "/api/v1/materials/"+idOf(t, freeMaterial).String(), nil, "").Status)

	denied := c.send(http.MethodGet, closedMaterials, nil, "")
	require.Equal(t, http.StatusForbidden, denied.Status)
	require.Equal(t, "FORBIDDEN", denied.errorCode())

	require.Equal(t, http.StatusForbidden, c.as(student, http.MethodGet, closedMaterials, nil).Status, "a student who did not enroll")
	require.Equal(t, http.StatusOK, c.as(owner, http.MethodGet, closedMaterials, nil).Status)

	enrolled := c.as(student, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/enrollments", nil)
	require.Equal(t, http.StatusCreated, enrolled.Status, enrolled.Raw)

	require.Equal(t, http.StatusOK, c.as(student, http.MethodGet, closedMaterials, nil).Status, "after enrolling")
}

func TestMaterials_CreateValidation(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	module := c.as(owner, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/modules", map[string]any{"title": "A"})
	lesson := c.as(owner, http.MethodPost, "/api/v1/modules/"+idOf(t, module).String()+"/lessons", map[string]any{"title": "L"})

	path := "/api/v1/lessons/" + idOf(t, lesson).String() + "/materials"

	noType := c.as(owner, http.MethodPost, path, map[string]any{"content": "x"})
	require.Equal(t, http.StatusBadRequest, noType.Status)
	require.Contains(t, noType.fields(), "type")

	badType := c.as(owner, http.MethodPost, path, map[string]any{"type": "audio", "content": "x"})
	require.Equal(t, http.StatusBadRequest, badType.Status)
	require.Contains(t, badType.fields()["type"], "one of")

	noContent := c.as(owner, http.MethodPost, path, map[string]any{"type": "text"})
	require.Equal(t, http.StatusBadRequest, noContent.Status, "the service checks that a text has content")

	badVideo := c.as(owner, http.MethodPost, path, map[string]any{"type": "video", "content": "not a link"})
	require.Equal(t, http.StatusBadRequest, badVideo.Status)

	notUploaded := c.as(owner, http.MethodPost, path, map[string]any{"type": "file", "object_key": "materials/never-uploaded.pdf"})
	require.Equal(t, http.StatusBadRequest, notUploaded.Status)
	require.Contains(t, notUploaded.errorMessage(), "not uploaded")
}

func TestMaterials_FileFlow(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	module := c.as(owner, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/modules", map[string]any{"title": "A"})
	lesson := c.as(owner, http.MethodPost, "/api/v1/modules/"+idOf(t, module).String()+"/lessons", map[string]any{"title": "L"})

	path := "/api/v1/lessons/" + idOf(t, lesson).String() + "/materials"

	// 1. ask for an address, 2. upload there (here: the fake storage gets the file), 3. attach it
	presign := c.as(owner, http.MethodPost, "/api/v1/uploads/presign", map[string]any{"purpose": "material", "file_name": "slides.pdf", "content_type": "application/pdf"})
	require.Equal(t, http.StatusOK, presign.Status, presign.Raw)

	key := presign.data()["object_key"].(string)
	c.e.Storage.Upload(key, 4096, "application/pdf")

	created := c.as(owner, http.MethodPost, path, map[string]any{"type": "file", "object_key": key, "file_name": "slides.pdf"})
	require.Equal(t, http.StatusCreated, created.Status, created.Raw)
	require.EqualValues(t, 4096, created.data()["file_size"])

	// the file moved out of the temporary folder; the material has the new key
	permanent := strings.TrimPrefix(key, "tmp/")
	require.Equal(t, permanent, created.data()["object_key"])
	require.Equal(t, "http://files.test/"+permanent, created.data()["file_url"])

	materialPath := "/api/v1/materials/" + idOf(t, created).String()

	require.Equal(t, http.StatusNoContent, c.as(owner, http.MethodDelete, materialPath, nil).Status)
	require.Contains(t, c.e.Storage.Deleted, permanent, "the file is removed with the material")
	require.Equal(t, http.StatusNotFound, c.as(owner, http.MethodGet, materialPath, nil).Status)
}

func TestMaterials_UpdateByTheOwnerOnly(t *testing.T) {
	c := newClient(t)

	owner := c.e.NewUser(t, models.RoleInstructor)
	stranger := c.e.NewUser(t, models.RoleInstructor)
	courseID := c.createCourse(owner, c.e.NewCategory(t).ID)

	module := c.as(owner, http.MethodPost, "/api/v1/courses/"+courseID.String()+"/modules", map[string]any{"title": "A"})
	lesson := c.as(owner, http.MethodPost, "/api/v1/modules/"+idOf(t, module).String()+"/lessons", map[string]any{"title": "L"})
	material := c.as(owner, http.MethodPost, "/api/v1/lessons/"+idOf(t, lesson).String()+"/materials", map[string]any{"type": "text", "content": "old"})

	path := "/api/v1/materials/" + idOf(t, material).String()

	updated := c.as(owner, http.MethodPut, path, map[string]any{"content": "new"})
	require.Equal(t, http.StatusOK, updated.Status)
	require.Equal(t, "new", updated.data()["content"])

	require.Equal(t, http.StatusForbidden, c.as(stranger, http.MethodPut, path, map[string]any{"content": "hijacked"}).Status)
	require.Equal(t, http.StatusForbidden, c.as(stranger, http.MethodDelete, path, nil).Status)
}
