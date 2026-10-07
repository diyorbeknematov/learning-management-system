package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// bearer returns the Authorization header of a user.
func (c *client) bearer(user *models.User) string {
	c.t.Helper()

	raw, err := c.e.Tokens.GenerateAccessToken(user.ID, user.RoleID, user.RoleName)
	require.NoError(c.t, err)

	return "Bearer " + raw
}

// as calls a route as a user.
func (c *client) as(user *models.User, method, path string, body any) answer {
	c.t.Helper()

	return c.send(method, path, body, c.bearer(user))
}

func TestPrivateRoutes_NeedAToken(t *testing.T) {
	c := newClient(t)

	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodPut, "/api/v1/users/me"},
		{http.MethodPut, "/api/v1/users/me/password"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodPost, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users/" + uuid.NewString()},
		{http.MethodPut, "/api/v1/users/" + uuid.NewString()},
		{http.MethodPatch, "/api/v1/users/" + uuid.NewString() + "/status"},
		{http.MethodDelete, "/api/v1/users/" + uuid.NewString()},
		{http.MethodPost, "/api/v1/uploads/presign"},
	} {
		got := c.send(tt.method, tt.path, map[string]any{}, "")
		require.Equal(t, http.StatusUnauthorized, got.Status, "%s %s", tt.method, tt.path)
		require.Equal(t, "UNAUTHORIZED", got.errorCode())
	}
}

func TestAdminRoutes_AreClosedToOtherRoles(t *testing.T) {
	c := newClient(t)

	student := c.e.NewUser(t, models.RoleStudent)
	instructor := c.e.NewUser(t, models.RoleInstructor)
	other := c.e.NewUser(t, models.RoleStudent)

	for _, user := range []*models.User{student, instructor} {
		for _, tt := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/users"},
			{http.MethodPost, "/api/v1/users"},
			{http.MethodGet, "/api/v1/users/" + other.ID.String()},
			{http.MethodPut, "/api/v1/users/" + other.ID.String()},
			{http.MethodPatch, "/api/v1/users/" + other.ID.String() + "/status"},
			{http.MethodDelete, "/api/v1/users/" + other.ID.String()},
		} {
			got := c.as(user, tt.method, tt.path, map[string]any{})
			require.Equal(t, http.StatusForbidden, got.Status, "%s %s %s", user.RoleName, tt.method, tt.path)
			require.Equal(t, "FORBIDDEN", got.errorCode())
		}
	}

	// nothing was changed by the refused calls
	found, err := c.e.Svc.User.GetByID(t.Context(), other.ID)
	require.NoError(t, err)
	require.Equal(t, "active", found.Status)
}

func TestMe(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleInstructor)

	got := c.as(user, http.MethodGet, "/api/v1/users/me", nil)
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.Equal(t, user.Username, got.data()["username"])
	require.Equal(t, user.Email, got.data()["email"])
	require.Equal(t, models.RoleInstructor, got.data()["role_name"])
	require.NotContains(t, got.Raw, "password")
}

func TestMe_UpdateProfile(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	got := c.as(user, http.MethodPut, "/api/v1/users/me", map[string]any{"bio": "I learn Go", "first_name": "Changed"})
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.Equal(t, "I learn Go", got.data()["bio"])
	require.Equal(t, "Changed", got.data()["first_name"])
	require.Equal(t, user.LastName, got.data()["last_name"], "what was not sent stays")
}

func TestMe_UpdateCannotChangeTheRole(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	got := c.as(user, http.MethodPut, "/api/v1/users/me", map[string]any{"role": "SuperAdmin", "role_id": uuid.NewString(), "status": "active"})
	require.Equal(t, http.StatusOK, got.Status, "unknown fields are ignored")
	require.Equal(t, models.RoleStudent, got.data()["role_name"])

	found, err := c.e.Svc.User.GetByID(t.Context(), user.ID)
	require.NoError(t, err)
	require.Equal(t, models.RoleStudent, found.RoleName)
}

func TestMe_UpdateValidation(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)

	got := c.as(user, http.MethodPut, "/api/v1/users/me", map[string]any{"email": "nope", "username": "a b"})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Contains(t, got.fields()["email"], "valid email")
	require.Contains(t, got.fields()["username"], "3 to 30")
}

func TestMe_UpdateDuplicateEmail(t *testing.T) {
	c := newClient(t)

	first := c.e.NewUser(t, models.RoleStudent)
	second := c.e.NewUser(t, models.RoleStudent)

	got := c.as(second, http.MethodPut, "/api/v1/users/me", map[string]any{"email": strings.ToUpper(first.Email)})
	require.Equal(t, http.StatusConflict, got.Status)
	require.Equal(t, "CONFLICT", got.errorCode())
}

func TestMe_ChangePassword(t *testing.T) {
	c := newClient(t)

	user := c.e.NewUser(t, models.RoleStudent)
	login := c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password})
	refresh := login.data()["refresh_token"].(string)

	wrongOld := c.as(user, http.MethodPut, "/api/v1/users/me/password", map[string]any{"old_password": "Wrong-Passw0rd!", "new_password": "Brand-New-Pass1!"})
	require.Equal(t, http.StatusBadRequest, wrongOld.Status)
	require.Contains(t, wrongOld.errorMessage(), "old password")

	weak := c.as(user, http.MethodPut, "/api/v1/users/me/password", map[string]any{"old_password": testutil.Password, "new_password": "weak"})
	require.Equal(t, http.StatusBadRequest, weak.Status)
	require.Contains(t, weak.fields()["new_password"], "capital letter")

	missing := c.as(user, http.MethodPut, "/api/v1/users/me/password", map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status)
	require.Contains(t, missing.fields(), "old_password")

	// nothing changed yet
	require.Equal(t, http.StatusOK, c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password}).Status)

	changed := c.as(user, http.MethodPut, "/api/v1/users/me/password", map[string]any{"old_password": testutil.Password, "new_password": "Brand-New-Pass1!"})
	require.Equal(t, http.StatusOK, changed.Status, changed.Raw)

	require.Equal(t, http.StatusUnauthorized, c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": testutil.Password}).Status)
	require.Equal(t, http.StatusOK, c.post("/api/v1/auth/login", map[string]any{"username": user.Username, "password": "Brand-New-Pass1!"}).Status)
	require.Equal(t, http.StatusUnauthorized, c.post("/api/v1/auth/refresh", map[string]any{"refresh_token": refresh}).Status, "the old sessions ended")
}

func TestAdmin_CreateUser(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	name := uniq()

	got := c.as(admin, http.MethodPost, "/api/v1/users", map[string]any{
		"first_name": "New",
		"last_name":  "Instructor",
		"username":   "i" + name,
		"email":      name + "@example.com",
		"password":   testutil.Password,
		"role":       "Instructor",
	})
	require.Equal(t, http.StatusCreated, got.Status, got.Raw)
	require.Equal(t, models.RoleInstructor, got.data()["role_name"])

	id, err := uuid.Parse(got.data()["id"].(string))
	require.NoError(t, err)

	c.e.CleanupUser(t, id)
}

func TestAdmin_CreateUser_Validation(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	got := c.as(admin, http.MethodPost, "/api/v1/users", map[string]any{
		"first_name": "New",
		"last_name":  "User",
		"username":   "ok_name",
		"email":      "ok@example.com",
		"password":   "weak",
		"role":       "Wizard",
	})
	require.Equal(t, http.StatusBadRequest, got.Status)
	require.Contains(t, got.fields()["role"], "one of")
	require.Contains(t, got.fields()["password"], "capital letter")

	missing := c.as(admin, http.MethodPost, "/api/v1/users", map[string]any{})
	require.Equal(t, http.StatusBadRequest, missing.Status)

	for _, field := range []string{"first_name", "last_name", "username", "email", "password", "role"} {
		require.Contains(t, missing.fields(), field)
	}
}

func TestAdmin_ListUsers(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)

	got := c.as(admin, http.MethodGet, "/api/v1/users?search="+student.Username, nil)
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.EqualValues(t, 1, got.data()["total"])
	require.EqualValues(t, 1, got.data()["page"])
	require.EqualValues(t, 10, got.data()["limit"])
	require.Len(t, got.data()["items"], 1)

	filtered := c.as(admin, http.MethodGet, "/api/v1/users?search="+student.Username+"&role=Instructor", nil)
	require.Equal(t, http.StatusOK, filtered.Status)
	require.EqualValues(t, 0, filtered.data()["total"])
	require.NotNil(t, filtered.data()["items"], "an empty page is [], not null")
	require.Empty(t, filtered.data()["items"])
}

func TestAdmin_ListUsers_Paging(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	c.e.NewUser(t, models.RoleStudent)
	c.e.NewUser(t, models.RoleStudent)

	got := c.as(admin, http.MethodGet, "/api/v1/users?limit=2&page=1", nil)
	require.Equal(t, http.StatusOK, got.Status)
	require.Len(t, got.data()["items"], 2)
	require.EqualValues(t, 2, got.data()["limit"])

	huge := c.as(admin, http.MethodGet, "/api/v1/users?limit=100000", nil)
	require.Equal(t, http.StatusOK, huge.Status)
	require.EqualValues(t, 100, huge.data()["limit"], "the largest page is 100")
}

func TestAdmin_ListUsers_BadQuery(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)

	for _, query := range []string{"limit=abc", "page=x", "role=Wizard", "status=deleted"} {
		got := c.as(admin, http.MethodGet, "/api/v1/users?"+query, nil)
		require.Equal(t, http.StatusBadRequest, got.Status, query)
		require.Equal(t, "INVALID_INPUT", got.errorCode(), query)
	}
}

func TestAdmin_GetUser(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)

	got := c.as(admin, http.MethodGet, "/api/v1/users/"+student.ID.String(), nil)
	require.Equal(t, http.StatusOK, got.Status)
	require.Equal(t, student.Username, got.data()["username"])

	missing := c.as(admin, http.MethodGet, "/api/v1/users/"+uuid.NewString(), nil)
	require.Equal(t, http.StatusNotFound, missing.Status)
	require.Equal(t, "NOT_FOUND", missing.errorCode())

	bad := c.as(admin, http.MethodGet, "/api/v1/users/not-an-id", nil)
	require.Equal(t, http.StatusBadRequest, bad.Status)
	require.Equal(t, "must be a valid id", bad.fields()["id"])
}

func TestAdmin_UpdateUser(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)

	got := c.as(admin, http.MethodPut, "/api/v1/users/"+student.ID.String(), map[string]any{"role": "Instructor", "bio": "promoted"})
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.Equal(t, models.RoleInstructor, got.data()["role_name"])
	require.Equal(t, "promoted", got.data()["bio"])

	self := c.as(admin, http.MethodPut, "/api/v1/users/"+admin.ID.String(), map[string]any{"role": "Student"})
	require.Equal(t, http.StatusForbidden, self.Status, "nobody changes their own role")

	bad := c.as(admin, http.MethodPut, "/api/v1/users/"+student.ID.String(), map[string]any{"role": "Wizard"})
	require.Equal(t, http.StatusBadRequest, bad.Status)
}

func TestAdmin_UpdateStatus(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)

	blocked := c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusOK, blocked.Status, blocked.Raw)

	login := c.post("/api/v1/auth/login", map[string]any{"username": student.Username, "password": testutil.Password})
	require.Equal(t, http.StatusForbidden, login.Status)

	invalid := c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{"status": "deleted"})
	require.Equal(t, http.StatusBadRequest, invalid.Status)
	require.Contains(t, invalid.fields()["status"], "one of")

	empty := c.as(admin, http.MethodPatch, "/api/v1/users/"+student.ID.String()+"/status", map[string]any{})
	require.Equal(t, http.StatusBadRequest, empty.Status)

	self := c.as(admin, http.MethodPatch, "/api/v1/users/"+admin.ID.String()+"/status", map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusForbidden, self.Status)

	unknown := c.as(admin, http.MethodPatch, "/api/v1/users/"+uuid.NewString()+"/status", map[string]any{"status": "blocked"})
	require.Equal(t, http.StatusNotFound, unknown.Status)
}

func TestAdmin_DeleteUser(t *testing.T) {
	c := newClient(t)

	admin := c.e.NewUser(t, models.RoleSuperAdmin)
	student := c.e.NewUser(t, models.RoleStudent)

	self := c.as(admin, http.MethodDelete, "/api/v1/users/"+admin.ID.String(), nil)
	require.Equal(t, http.StatusForbidden, self.Status)

	deleted := c.as(admin, http.MethodDelete, "/api/v1/users/"+student.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, deleted.Status)
	require.Empty(t, deleted.Raw)

	gone := c.as(admin, http.MethodGet, "/api/v1/users/"+student.ID.String(), nil)
	require.Equal(t, http.StatusNotFound, gone.Status)

	again := c.as(admin, http.MethodDelete, "/api/v1/users/"+student.ID.String(), nil)
	require.Equal(t, http.StatusNotFound, again.Status)
}

func TestUpload_Presign(t *testing.T) {
	c := newClient(t)

	student := c.e.NewUser(t, models.RoleStudent)

	got := c.as(student, http.MethodPost, "/api/v1/uploads/presign", map[string]any{
		"purpose":      "avatar",
		"file_name":    "me.png",
		"content_type": "image/png",
	})
	require.Equal(t, http.StatusOK, got.Status, got.Raw)
	require.True(t, strings.HasPrefix(got.data()["object_key"].(string), "tmp/avatars/"))
	require.NotEmpty(t, got.data()["upload_url"])
	require.EqualValues(t, 5<<20, got.data()["max_file_size"])
}

func TestUpload_Presign_StudentCannotUploadACover(t *testing.T) {
	c := newClient(t)

	student := c.e.NewUser(t, models.RoleStudent)
	instructor := c.e.NewUser(t, models.RoleInstructor)

	body := map[string]any{"purpose": "course_cover", "file_name": "c.png", "content_type": "image/png"}

	got := c.as(student, http.MethodPost, "/api/v1/uploads/presign", body)
	require.Equal(t, http.StatusForbidden, got.Status)

	ok := c.as(instructor, http.MethodPost, "/api/v1/uploads/presign", body)
	require.Equal(t, http.StatusOK, ok.Status)
}

func TestUpload_Presign_Validation(t *testing.T) {
	c := newClient(t)

	instructor := c.e.NewUser(t, models.RoleInstructor)

	empty := c.as(instructor, http.MethodPost, "/api/v1/uploads/presign", map[string]any{})
	require.Equal(t, http.StatusBadRequest, empty.Status)

	for _, field := range []string{"purpose", "file_name", "content_type"} {
		require.Contains(t, empty.fields(), field)
	}

	badPurpose := c.as(instructor, http.MethodPost, "/api/v1/uploads/presign", map[string]any{"purpose": "backup", "file_name": "a", "content_type": "image/png"})
	require.Equal(t, http.StatusBadRequest, badPurpose.Status)
	require.Contains(t, badPurpose.fields()["purpose"], "one of")

	badType := c.as(instructor, http.MethodPost, "/api/v1/uploads/presign", map[string]any{"purpose": "material", "file_name": "a.exe", "content_type": "application/x-msdownload"})
	require.Equal(t, http.StatusBadRequest, badType.Status)
	require.Equal(t, "INVALID_INPUT", badType.errorCode())
}
