package account_test

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func registerRequest() models.RegisterRequest {
	name := testutil.Uniq()

	return models.RegisterRequest{
		FirstName: "Ali",
		LastName:  "Valiyev",
		Username:  "u" + name,
		Email:     name + "@example.com",
		Password:  testutil.Password,
	}
}

func register(t *testing.T, e *testutil.Env, req models.RegisterRequest) *models.TokenPair {
	t.Helper()

	pair, err := e.Svc.Auth.Register(context.Background(), req)
	require.NoError(t, err)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)

	e.CleanupUser(t, claims.UserID)

	return pair
}

func TestAuth_Register(t *testing.T) {
	e := testutil.Setup(t)

	pair := register(t, e, registerRequest())

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, models.RoleStudent, claims.RoleName)
	require.NotEqual(t, uuid.Nil, claims.UserID)
	require.NotEmpty(t, pair.RefreshToken)
}

func TestAuth_Register_StoresHashedPassword(t *testing.T) {
	e := testutil.Setup(t)

	req := registerRequest()
	pair := register(t, e, req)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)

	hash, err := e.Repo.User.GetPasswordHash(context.Background(), claims.UserID.String())
	require.NoError(t, err)
	require.NotEqual(t, req.Password, hash)
}

func TestAuth_Register_DuplicateEmail(t *testing.T) {
	e := testutil.Setup(t)

	req := registerRequest()
	register(t, e, req)

	again := registerRequest()
	again.Email = req.Email

	_, err := e.Svc.Auth.Register(context.Background(), again)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestAuth_Register_DuplicateUsername(t *testing.T) {
	e := testutil.Setup(t)

	req := registerRequest()
	register(t, e, req)

	again := registerRequest()
	again.Username = req.Username

	_, err := e.Svc.Auth.Register(context.Background(), again)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestAuth_Login(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleInstructor)

	response, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{
		Username: user.Username,
		Password: testutil.Password,
	})
	require.NoError(t, err)
	require.Equal(t, user.ID, response.User.ID)
	require.NotEmpty(t, response.RefreshToken)

	claims, err := e.Tokens.ParseAccessToken(response.AccessToken)
	require.NoError(t, err)
	require.Equal(t, models.RoleInstructor, claims.RoleName)
	require.Equal(t, user.RoleID, claims.RoleID)
}

func TestAuth_Login_WrongPassword(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: user.Username, Password: "wrong"})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Login_UnknownUser(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: "nobody-" + testutil.Uniq(), Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Login_BlockedUser(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	e.Exec(t, `UPDATE users SET status = 'blocked' WHERE id = $1`, user.ID)

	_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	// the status must not be revealed to someone with a wrong password
	_, err = e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: user.Username, Password: "wrong"})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Refresh(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	pair, err := e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, login.RefreshToken, pair.RefreshToken)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, user.ID, claims.UserID)
}

func TestAuth_Refresh_TokenWorksOnce(t *testing.T) {
	e := testutil.Setup(t)

	login := e.Login(t, e.NewUser(t, models.RoleStudent))

	_, err := e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	require.NoError(t, err)

	_, err = e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Refresh_Invalid(t *testing.T) {
	e := testutil.Setup(t)

	_, err := e.Svc.Auth.Refresh(context.Background(), "not-a-real-token")
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Refresh_Expired(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	e.Exec(t, `UPDATE refresh_tokens SET expires_at = $2 WHERE user_id = $1`, user.ID, time.Now().Add(-time.Hour))

	_, err := e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Refresh_BlockedUser(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	e.Exec(t, `UPDATE users SET status = 'blocked' WHERE id = $1`, user.ID)

	_, err := e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeForbidden)
}

func TestAuth_Logout(t *testing.T) {
	e := testutil.Setup(t)

	login := e.Login(t, e.NewUser(t, models.RoleStudent))

	require.NoError(t, e.Svc.Auth.Logout(context.Background(), login.RefreshToken))

	_, err := e.Svc.Auth.Refresh(context.Background(), login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Logout_UnknownToken(t *testing.T) {
	e := testutil.Setup(t)

	require.NoError(t, e.Svc.Auth.Logout(context.Background(), "not-a-real-token"))
}

func resetToken(t *testing.T, e *testutil.Env) string {
	t.Helper()

	match := regexp.MustCompile(`\?token=(\S+)`).FindStringSubmatch(e.Mailer.Body)
	require.Len(t, match, 2, "the email has no reset link")

	raw, err := url.QueryUnescape(match[1])
	require.NoError(t, err)

	return raw
}

func TestAuth_ForgotPassword(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(context.Background(), user.Email))

	require.Equal(t, 1, e.Mailer.Sent)
	require.Equal(t, user.Email, e.Mailer.To)
	require.Contains(t, e.Mailer.Body, "http://localhost:3000/reset-password?token=")
	require.Contains(t, e.Mailer.Body, "15 minutes")

	require.Len(t, e.Redis.Values, 1)

	for key, ttl := range e.Redis.TTLs {
		if strings.HasPrefix(key, "forgot_password:") {
			continue // the counter of the per-email limit
		}

		require.Contains(t, key, "password_reset:")
		require.NotContains(t, key, resetToken(t, e), "the raw token must not be stored")
		require.Equal(t, 15*time.Minute, ttl)
	}
}

func TestAuth_ForgotPassword_UnknownEmail(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.Auth.ForgotPassword(context.Background(), "nobody-"+testutil.Uniq()+"@example.com")

	testutil.RequireCode(t, err, apperror.CodeNotFound)
	require.Zero(t, e.Mailer.Sent)
}

func TestAuth_ResetPassword(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))

	err := e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: resetToken(t, e), NewPassword: "Brand-New-Pass1!"})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: "Brand-New-Pass1!"})
	require.NoError(t, err)
}

func TestAuth_ResetPassword_TokenWorksOnce(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))
	raw := resetToken(t, e)

	require.NoError(t, e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: raw, NewPassword: "Brand-New-Pass1!"}))

	err := e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: raw, NewPassword: "Another-Pass12!"})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestAuth_ResetPassword_InvalidToken(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.Auth.ResetPassword(context.Background(), models.ResetPasswordRequest{Token: "nope", NewPassword: "Brand-New-Pass1!"})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestAuth_ResetPassword_EndsSessions(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)
	login := e.Login(t, user)

	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))
	require.NoError(t, e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: resetToken(t, e), NewPassword: "Brand-New-Pass1!"}))

	_, err := e.Svc.Auth.Refresh(ctx, login.RefreshToken)
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Register_WeakPasswords(t *testing.T) {
	e := testutil.Setup(t)

	for name, password := range map[string]string{
		"too short":           "Ab1!",
		"no capital letter":   "password1!",
		"no small letter":     "PASSWORD1!",
		"no digit":            "Password!!",
		"no special":          "Password12",
		"too long for bcrypt": "Ab1!" + strings.Repeat("x", 70),
		"empty":               "",
	} {
		req := registerRequest()
		req.Password = password

		_, err := e.Svc.Auth.Register(context.Background(), req)
		require.Error(t, err, name)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)

		exists, err := e.Repo.User.ExistsByUsername(context.Background(), req.Username)
		require.NoError(t, err)
		require.False(t, exists, "nothing is saved: "+name)
	}
}

func TestAuth_Register_TheErrorSaysWhatIsMissing(t *testing.T) {
	e := testutil.Setup(t)

	req := registerRequest()
	req.Password = "abcdefgh"

	_, err := e.Svc.Auth.Register(context.Background(), req)

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Contains(t, appErr.Message, "a capital letter")
	require.Contains(t, appErr.Message, "a digit")
	require.Contains(t, appErr.Message, "a special character")
	require.NotContains(t, appErr.Message, "a small letter")
}

func TestAuth_Register_InvalidEmails(t *testing.T) {
	e := testutil.Setup(t)

	for _, email := range []string{"", "plainaddress", "ali@", "@example.com", "ali@example", "Ali <ali@example.com>", "a b@example.com"} {
		req := registerRequest()
		req.Email = email

		_, err := e.Svc.Auth.Register(context.Background(), req)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestAuth_Register_EmailIsStoredInLowerCase(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	req := registerRequest()
	req.Email = "  " + strings.ToUpper(req.Email) + "  "

	pair := register(t, e, req)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)

	user, err := e.Svc.User.GetByID(ctx, claims.UserID)
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(strings.TrimSpace(req.Email)), user.Email)

	// the same address in another case is the same account
	again := registerRequest()
	again.Email = strings.ToLower(strings.TrimSpace(req.Email))

	_, err = e.Svc.Auth.Register(ctx, again)
	testutil.RequireCode(t, err, apperror.CodeConflict)
}

func TestAuth_ForgotPassword_FindsTheUserInAnyCase(t *testing.T) {
	e := testutil.Setup(t)

	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(context.Background(), "  "+strings.ToUpper(user.Email)+" "))
	require.Equal(t, user.Email, e.Mailer.To)
}

func TestAuth_ForgotPassword_InvalidEmail(t *testing.T) {
	e := testutil.Setup(t)

	err := e.Svc.Auth.ForgotPassword(context.Background(), "not-an-email")
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	require.Zero(t, e.Mailer.Sent)
}

func TestAuth_ResetPassword_WeakPasswordDoesNotUseUpTheToken(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))
	raw := resetToken(t, e)

	err := e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: raw, NewPassword: "weak"})
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)

	// the same link still works with a good password
	err = e.Svc.Auth.ResetPassword(ctx, models.ResetPasswordRequest{Token: raw, NewPassword: "Brand-New-Pass1!"})
	require.NoError(t, err)
}

func TestAuth_Register_InvalidUsernames(t *testing.T) {
	e := testutil.Setup(t)

	for _, username := range []string{"", "ab", "ali baba", "ali@x", "_ali", "аdmin", "<script>", strings.Repeat("a", 31)} {
		req := registerRequest()
		req.Username = username

		_, err := e.Svc.Auth.Register(context.Background(), req)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestAuth_Register_UsernameKeepsItsCase(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	req := registerRequest()
	req.Username = "Ali" + testutil.Uniq()

	pair := register(t, e, req)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)

	user, err := e.Svc.User.GetByID(ctx, claims.UserID)
	require.NoError(t, err)
	require.Equal(t, req.Username, user.Username)

	// Ali and ali are two different usernames
	other := registerRequest()
	other.Username = strings.ToLower(req.Username)

	register(t, e, other)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: req.Username, Password: testutil.Password})
	require.NoError(t, err)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: strings.ToUpper(req.Username), Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeUnauthorized)
}

func TestAuth_Register_UsernameIsTrimmed(t *testing.T) {
	e := testutil.Setup(t)

	req := registerRequest()
	name := req.Username
	req.Username = "  " + name + "  "

	pair := register(t, e, req)

	claims, err := e.Tokens.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)

	user, err := e.Svc.User.GetByID(context.Background(), claims.UserID)
	require.NoError(t, err)
	require.Equal(t, name, user.Username)
}

func wrongLogins(t *testing.T, e *testutil.Env, username string, times int) {
	t.Helper()

	for i := 0; i < times; i++ {
		_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: username, Password: "Wrong-Passw0rd!"})
		testutil.RequireCode(t, err, apperror.CodeUnauthorized)
	}
}

func TestAuth_Login_LocksAfterTooManyWrongPasswords(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	wrongLogins(t, e, user.Username, 5)

	// the sixth try is refused, even with the right password
	_, err := e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	testutil.RequireCode(t, err, apperror.CodeTooMany)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: "Wrong-Passw0rd!"})
	testutil.RequireCode(t, err, apperror.CodeTooMany)

	// another account is not touched
	other := e.NewUser(t, models.RoleStudent)
	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: other.Username, Password: testutil.Password})
	require.NoError(t, err)

	// when the window ends the user can try again
	e.Redis.Reset()

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)
}

func TestAuth_Login_FewMistakesAreFine(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	wrongLogins(t, e, user.Username, 4)

	_, err := e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err, "the right password after four mistakes works")
}

func TestAuth_Login_ARightPasswordStartsTheCountAgain(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	wrongLogins(t, e, user.Username, 4)

	_, err := e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)

	// four more mistakes are fine again: the count started over
	wrongLogins(t, e, user.Username, 4)

	_, err = e.Svc.Auth.Login(ctx, models.LoginRequest{Username: user.Username, Password: testutil.Password})
	require.NoError(t, err)
}

func TestAuth_Login_UnknownUsernamesAreCountedToo(t *testing.T) {
	e := testutil.Setup(t)

	name := "ghost" + testutil.Uniq()

	for i := 0; i < 5; i++ {
		_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: name, Password: "Whatever1!"})
		testutil.RequireCode(t, err, apperror.CodeUnauthorized)
	}

	_, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: name, Password: "Whatever1!"})
	testutil.RequireCode(t, err, apperror.CodeTooMany)
}

func TestAuth_ForgotPassword_LimitedPerEmail(t *testing.T) {
	e := testutil.Setup(t)

	ctx := context.Background()
	user := e.NewUser(t, models.RoleStudent)

	for i := 0; i < 3; i++ {
		require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, user.Email))
	}

	err := e.Svc.Auth.ForgotPassword(ctx, user.Email)
	testutil.RequireCode(t, err, apperror.CodeTooMany)
	require.Equal(t, 3, e.Mailer.Sent, "no fourth email")

	// another address is not touched
	other := e.NewUser(t, models.RoleStudent)
	require.NoError(t, e.Svc.Auth.ForgotPassword(ctx, other.Email))

	// the limit counts the address in any case
	err = e.Svc.Auth.ForgotPassword(ctx, strings.ToUpper(user.Email))
	testutil.RequireCode(t, err, apperror.CodeTooMany)
}

func TestAuth_ForgotPassword_UnknownEmailsCountToo(t *testing.T) {
	e := testutil.Setup(t)

	email := "nobody" + testutil.Uniq() + "@example.com"

	for i := 0; i < 3; i++ {
		err := e.Svc.Auth.ForgotPassword(context.Background(), email)
		testutil.RequireCode(t, err, apperror.CodeNotFound)
	}

	err := e.Svc.Auth.ForgotPassword(context.Background(), email)
	testutil.RequireCode(t, err, apperror.CodeTooMany)
}
