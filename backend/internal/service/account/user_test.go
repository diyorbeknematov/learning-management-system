package account_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newUserRequest(role string) models.CreateUserRequest {
	name := testutil.Uniq()

	return models.CreateUserRequest{
		FirstName: "Test",
		LastName:  "User",
		Username:  "u" + name,
		Email:     name + "@example.com",
		Password:  testutil.Password,
		Role:      role,
	}
}

func TestUser_Create(t *testing.T) {
	e := testutil.Setup(t)

	bio := "Teaches Go"
	req := newUserRequest(models.RoleInstructor)
	req.Bio = &bio

	user, err := e.Svc.User.Create(context.Background(), req)
	require.NoError(t, err)

	e.CleanupUser(t, user.ID)

	require.Equal(t, models.RoleInstructor, user.RoleName)
	require.Equal(t, req.Username, user.Username)
	require.Equal(t, bio, *user.Bio)
	require.Equal(t, "active", user.Status)
}

func TestUser_Create_DuplicateEmail(t *testing.T) {
	e := testutil.Setup(t)

	existing := e.NewUser(t, models.RoleStudent)

	req := newUserRequest(models.RoleStudent)
	req.Email = existing.Email

	_, err := e.Svc.User.Create(context.Background(), req)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestUser_Create_DuplicateUsername(t *testing.T) {
	e := testutil.Setup(t)

	existing := e.NewUser(t, models.RoleStudent)

	req := newUserRequest(models.RoleStudent)
	req.Username = existing.Username

	_, err := e.Svc.User.Create(context.Background(), req)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestUser_Create_UnknownRole(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.User.Create(context.Background(), newUserRequest("Wizard"))
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestUser_GetByID(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	found, err := e.Svc.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, found.ID)
}

func TestUser_GetByID_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.User.GetByID(context.Background(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestUser_GetList(t *testing.T) {
	e := testutil.Setup(t)

	student := e.NewUser(t, models.RoleStudent)
	e.NewUser(t, models.RoleInstructor)

	search := student.Username

	list, err := e.Svc.User.GetList(context.Background(), models.UserFilter{Search: &search})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Equal(t, student.ID, list.Items[0].ID)
	require.Equal(t, 1, list.Page)
	require.Equal(t, 10, list.Limit)
}

func TestUser_GetList_FilterByRole(t *testing.T) {
	e := testutil.Setup(t)

	instructor := e.NewUser(t, models.RoleInstructor)
	e.NewUser(t, models.RoleStudent)

	search := instructor.Username
	role := models.RoleStudent

	list, err := e.Svc.User.GetList(context.Background(), models.UserFilter{Search: &search, RoleName: &role})
	require.NoError(t, err)
	require.Zero(t, list.Total)
	require.Empty(t, list.Items)
}

func TestUser_Update(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	firstName := "Changed"
	bio := "New bio"

	updated, err := e.Svc.User.Update(context.Background(), testutil.AdminActor(), user.ID, models.UpdateUserRequest{
		FirstName: &firstName,
		Bio:       &bio,
	})
	require.NoError(t, err)
	require.Equal(t, "Changed", updated.FirstName)
	require.Equal(t, bio, *updated.Bio)
	require.Equal(t, user.LastName, updated.LastName)
	require.Equal(t, models.RoleStudent, updated.RoleName)
}

func TestUser_Update_Role(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	role := models.RoleInstructor

	updated, err := e.Svc.User.Update(context.Background(), testutil.AdminActor(), user.ID, models.UpdateUserRequest{Role: &role})
	require.NoError(t, err)
	require.Equal(t, models.RoleInstructor, updated.RoleName)
}

func TestUser_Update_OwnRoleForbidden(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleInstructor)
	role := models.RoleSuperAdmin

	_, err := e.Svc.User.Update(context.Background(), testutil.ActorOf(user), user.ID, models.UpdateUserRequest{Role: &role})
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	found, err := e.Svc.User.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, models.RoleInstructor, found.RoleName)
}

func TestUser_Update_DuplicateEmail(t *testing.T) {
	e := testutil.Setup(t)

	first := e.NewUser(t, models.RoleStudent)
	second := e.NewUser(t, models.RoleStudent)

	_, err := e.Svc.User.Update(context.Background(), testutil.AdminActor(), second.ID, models.UpdateUserRequest{Email: &first.Email})
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestUser_Update_UnknownRole(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	role := "Wizard"

	_, err := e.Svc.User.Update(context.Background(), testutil.AdminActor(), user.ID, models.UpdateUserRequest{Role: &role})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestUser_UpdateStatus_BlockEndsSessions(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	err := e.Svc.User.UpdateStatus(ctx, testutil.AdminActor(), models.UpdateUserStatus{ID: user.ID, Status: "blocked"})
	require.NoError(t, err)

	found, err := e.Svc.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "blocked", found.Status)

	_, err = e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestUser_UpdateStatus_Unblock(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.User.UpdateStatus(ctx, testutil.AdminActor(), models.UpdateUserStatus{ID: user.ID, Status: "blocked"}))
	require.NoError(t, e.Svc.User.UpdateStatus(ctx, testutil.AdminActor(), models.UpdateUserStatus{ID: user.ID, Status: "active"}))

	_, err := e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)
}

func TestUser_UpdateStatus_OwnStatusForbidden(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleSuperAdmin)

	err := e.Svc.User.UpdateStatus(context.Background(), testutil.ActorOf(user), models.UpdateUserStatus{ID: user.ID, Status: "blocked"})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestUser_UpdateStatus_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.User.UpdateStatus(context.Background(), testutil.AdminActor(), models.UpdateUserStatus{ID: uuid.New(), Status: "blocked"})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestUser_Delete(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	require.NoError(t, e.Svc.User.Delete(ctx, testutil.AdminActor(), user.ID))

	_, err := e.Svc.User.GetByID(ctx, user.ID)
	testutil.RequireCode(t, err, apperror.CodeNotFound)

	_, err = e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestUser_Delete_Yourself(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleSuperAdmin)

	err := e.Svc.User.Delete(context.Background(), testutil.ActorOf(user), user.ID)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestUser_Delete_NotFound(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.User.Delete(context.Background(), testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestUser_GetMe(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	me, err := e.Svc.User.GetMe(context.Background(), testutil.ActorOf(user))
	require.NoError(t, err)
	require.Equal(t, user.ID, me.ID)
}

func TestUser_UpdateMe(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	bio := "About me"
	avatar := "avatars/me.png"
	e.Storage.Upload(avatar, 1024, "image/png")

	me, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Bio: &bio, Avatar: &avatar})
	require.NoError(t, err)
	require.Equal(t, bio, *me.Bio)
	require.Equal(t, avatar, *me.Avatar)
	require.Equal(t, models.RoleStudent, me.RoleName)
}

func TestUser_UpdateMe_DuplicateEmail(t *testing.T) {
	e := testutil.Setup(t)

	first := e.NewUser(t, models.RoleStudent)
	second := e.NewUser(t, models.RoleStudent)

	_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(second), models.UpdateMeRequest{Email: &first.Email})
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestUser_ChangePassword(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	err := e.Svc.User.ChangePassword(ctx, testutil.ActorOf(user), models.ChangePasswordRequest{
		OldPassword: testutil.Password,
		NewPassword: "Brand-New-Pass1!",
	})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: "Brand-New-Pass1!"})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestUser_ChangePassword_WrongOldPassword(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	err := e.Svc.User.ChangePassword(ctx, testutil.ActorOf(user), models.ChangePasswordRequest{
		OldPassword: "wrong",
		NewPassword: "Brand-New-Pass1!",
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	// nothing changed: the old password and the session still work
	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	require.NoError(t, err)
}

// uploadAvatar puts an image into the fake storage as if the client uploaded it.
func uploadAvatar(e *testutil.Env, name string) string {
	key := "avatars/" + name + ".png"
	e.Storage.Upload(key, 1024, "image/png")

	return key
}

func TestUser_Avatar_HasALink(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	require.Nil(t, user.AvatarURL, "no avatar, no link")

	key := uploadAvatar(e, testutil.Uniq())

	me, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &key})
	require.NoError(t, err)
	require.Equal(t, key, *me.Avatar)
	require.Equal(t, "http://files.test/"+key, *me.AvatarURL)

	byID, err := e.Svc.User.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+key, *byID.AvatarURL)

	mine, err := e.Svc.User.GetMe(ctx, testutil.ActorOf(user))
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+key, *mine.AvatarURL)

	search := user.Username

	list, err := e.Svc.User.GetList(ctx, models.UserFilter{Search: &search})
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+key, *list.Items[0].AvatarURL)

	login, err := e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+key, *login.User.AvatarURL)
}

func TestUser_Avatar_ReplacingRemovesTheOldFile(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	first, second := uploadAvatar(e, testutil.Uniq()), uploadAvatar(e, testutil.Uniq())

	_, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &first})
	require.NoError(t, err)
	require.Empty(t, e.Storage.Deleted)

	_, err = e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &second})
	require.NoError(t, err)
	require.Equal(t, []string{first}, e.Storage.Deleted)

	// sending the same avatar again keeps the file
	_, err = e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &second})
	require.NoError(t, err)
	require.Equal(t, []string{first}, e.Storage.Deleted)
}

func TestUser_Avatar_MustBeUploaded(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	missing := "avatars/never-uploaded.png"

	_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &missing})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestUser_Avatar_MustBeInTheAvatarFolder(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	for _, key := range []string{"covers/x.png", "materials/x.png", "../avatars/x.png", "avatars/../materials/x.png"} {
		e.Storage.Upload(key, 1024, "image/png")

		_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &key})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestUser_Avatar_MustBeAnImage(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	for _, contentType := range []string{"application/pdf", "application/octet-stream", "text/html", "image/svg+xml"} {
		key := "avatars/" + testutil.Uniq() + ".png"
		e.Storage.Upload(key, 1024, contentType)

		_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &key})
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
		require.Contains(t, e.Storage.Deleted, key, "a rejected file is removed: "+contentType)
	}
}

func TestUser_Avatar_ImageTypeWithParametersIsFine(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	key := "avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(key, 1024, "Image/PNG; charset=binary")

	_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &key})
	require.NoError(t, err)
}

func TestUser_Avatar_TooLarge(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	key := "avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(key, 6<<20, "image/png")

	_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &key})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	require.Contains(t, e.Storage.Deleted, key)

	exact := "avatars/" + testutil.Uniq() + ".png"
	e.Storage.Upload(exact, 5<<20, "image/png")

	_, err = e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &exact})
	require.NoError(t, err, "exactly the limit is allowed")
}

func TestUser_Avatar_ByTheAdmin(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	key := uploadAvatar(e, testutil.Uniq())

	updated, err := e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Avatar: &key})
	require.NoError(t, err)
	require.Equal(t, "http://files.test/"+key, *updated.AvatarURL)

	missing := "avatars/missing.png"

	_, err = e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Avatar: &missing})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestUser_Create_WithAnAvatar(t *testing.T) {
	e := testutil.Setup(t)

	key := uploadAvatar(e, testutil.Uniq())
	req := newUserRequest(models.RoleStudent)
	req.Avatar = &key

	user, err := e.Svc.User.Create(context.Background(), req)
	require.NoError(t, err)

	e.CleanupUser(t, user.ID)

	require.Equal(t, "http://files.test/"+key, *user.AvatarURL)

	missing := "avatars/missing.png"
	bad := newUserRequest(models.RoleStudent)
	bad.Avatar = &missing

	_, err = e.Svc.User.Create(context.Background(), bad)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestUser_Create_WeakPasswordAndBadEmail(t *testing.T) {
	e := testutil.Setup(t)

	weak := newUserRequest(models.RoleStudent)
	weak.Password = "password"

	_, err := e.Svc.User.Create(context.Background(), weak)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	bad := newUserRequest(models.RoleStudent)
	bad.Email = "nope"

	_, err = e.Svc.User.Create(context.Background(), bad)
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestUser_Create_EmailInLowerCase(t *testing.T) {
	e := testutil.Setup(t)

	req := newUserRequest(models.RoleStudent)
	req.Email = strings.ToUpper(req.Email)

	user, err := e.Svc.User.Create(context.Background(), req)
	require.NoError(t, err)

	e.CleanupUser(t, user.ID)

	require.Equal(t, strings.ToLower(req.Email), user.Email)
}

func TestUser_ChangePassword_WeakNewPassword(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	err := e.Svc.User.ChangePassword(ctx, testutil.ActorOf(user), models.ChangePasswordRequest{
		OldPassword: testutil.Password,
		NewPassword: "weakpass",
	})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	// nothing changed
	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	require.NoError(t, err)
}

func TestUser_Email_IsCheckedWhenItChanges(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	bad := "not-an-email"

	_, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Email: &bad})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	_, err = e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Email: &bad})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	mixed := "  New." + testutil.Uniq() + "@Example.COM "

	updated, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Email: &mixed})
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(strings.TrimSpace(mixed)), updated.Email)

	other := "Other." + testutil.Uniq() + "@Example.com"

	byAdmin, err := e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Email: &other})
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(other), byAdmin.Email)
}

func TestUser_Create_InvalidUsername(t *testing.T) {
	e := testutil.Setup(t)

	for _, username := range []string{"ab", "ali baba", "аdmin", "ali'; DROP TABLE users;--"} {
		req := newUserRequest(models.RoleStudent)
		req.Username = username

		_, err := e.Svc.User.Create(context.Background(), req)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestUser_Username_IsCheckedWhenItChanges(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	bad := "not valid!"

	_, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Username: &bad})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	_, err = e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Username: &bad})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	good := "New_" + testutil.Uniq()

	updated, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Username: &good})
	require.NoError(t, err)
	require.Equal(t, good, updated.Username)
}

// revokedKey is where the service notes that the tokens of a user were taken back.
func revokedKey(user *models.User) string {
	return "revoked_user:" + user.ID.String()
}

func TestUser_Revocation_BlockAndUnblock(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.User.UpdateStatus(ctx, testutil.AdminActor(), models.UpdateUserStatus{ID: user.ID, Status: "blocked"}))
	require.Contains(t, e.Redis.Values, revokedKey(user))
	require.Equal(t, time.Minute, e.Redis.TTLs[revokedKey(user)], "kept for as long as an access token lives")

	require.NoError(t, e.Svc.User.UpdateStatus(ctx, testutil.AdminActor(), models.UpdateUserStatus{ID: user.ID, Status: "active"}))
	require.NotContains(t, e.Redis.Values, revokedKey(user), "an unblocked user is let back in")
}

func TestUser_Revocation_FailedBlockRevokesNothing(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.User.UpdateStatus(context.Background(), testutil.AdminActor(), models.UpdateUserStatus{ID: uuid.New(), Status: "blocked"})
	testutil.RequireCode(t, err, apperror.CodeNotFound)
	require.Empty(t, e.Redis.Values)

	self := e.NewUser(t, models.RoleSuperAdmin)

	err = e.Svc.User.UpdateStatus(context.Background(), testutil.ActorOf(self), models.UpdateUserStatus{ID: self.ID, Status: "blocked"})
	testutil.RequireCode(t, err, apperror.CodeForbidden)
	require.NotContains(t, e.Redis.Values, revokedKey(self))
}

func TestUser_Revocation_Delete(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.User.Delete(context.Background(), testutil.AdminActor(), user.ID))
	require.Contains(t, e.Redis.Values, revokedKey(user))
}

func TestUser_Revocation_ChangePassword(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	err := e.Svc.User.ChangePassword(context.Background(), testutil.ActorOf(user), models.ChangePasswordRequest{
		OldPassword: testutil.Password,
		NewPassword: "Brand-New-Pass1!",
	})
	require.NoError(t, err)
	require.Contains(t, e.Redis.Values, revokedKey(user))
}

func TestUser_Revocation_RoleChangeOnly(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	// the name is not in the token, so the old tokens stay
	name := "Changed"

	_, err := e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{FirstName: &name})
	require.NoError(t, err)
	require.NotContains(t, e.Redis.Values, revokedKey(user))

	_, err = e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{FirstName: &name})
	require.NoError(t, err)
	require.NotContains(t, e.Redis.Values, revokedKey(user))

	// the role is in the token, so they go
	role := models.RoleInstructor

	_, err = e.Svc.User.Update(ctx, testutil.AdminActor(), user.ID, models.UpdateUserRequest{Role: &role})
	require.NoError(t, err)
	require.Contains(t, e.Redis.Values, revokedKey(user))
}

func TestAuth_ResetPassword_Revokes(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))
	require.NoError(t, e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: resetToken(t, e), NewPassword: "Brand-New-Pass1!"}))

	require.Contains(t, e.Redis.Values, revokedKey(user))
}

func TestUser_EnsureSuperAdmin_DoesNothingWhenOneExists(t *testing.T) {
	e := testutil.Setup(t)

	e.NewUser(t, models.RoleSuperAdmin)

	suffix := testutil.Uniq()

	created, err := e.Svc.User.EnsureSuperAdmin(context.Background(), models.CreateUserRequest{
		FirstName: "Super",
		LastName:  "Admin",
		Username:  "boot" + suffix,
		Email:     "boot" + suffix + "@example.com",
		Password:  testutil.Password,
	})
	require.NoError(t, err)
	require.False(t, created)

	_, err = e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: "boot" + suffix, Password: testutil.Password})
	require.Error(t, err, "no user was made")
}

func TestUser_Avatar_MovesOutOfTheTemporaryFolder(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	name := testutil.Uniq()
	temporary := "tmp/avatars/" + name + ".png"
	e.Storage.Upload(temporary, 1024, "image/png")

	me, err := e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &temporary})
	require.NoError(t, err)

	permanent := "avatars/" + name + ".png"
	require.Equal(t, permanent, *me.Avatar, "the saved key is the permanent one")
	require.Equal(t, "http://files.test/"+permanent, *me.AvatarURL)

	require.Contains(t, e.Storage.Objects, permanent)
	require.Contains(t, e.Storage.Objects, temporary, "the temporary file is left to the lifecycle rule of the bucket")

	// sending the same avatar again changes nothing and removes nothing
	again := permanent
	me, err = e.Svc.User.UpdateMe(ctx, testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &again})
	require.NoError(t, err)
	require.Equal(t, permanent, *me.Avatar)
	require.NotContains(t, e.Storage.Deleted, permanent)
}

func TestUser_Avatar_TemporaryFileOfAnotherPurposeIsRejected(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	cover := "tmp/covers/" + testutil.Uniq() + ".png"
	e.Storage.Upload(cover, 1024, "image/png")

	_, err := e.Svc.User.UpdateMe(context.Background(), testutil.ActorOf(user), models.UpdateMeRequest{Avatar: &cover})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}
