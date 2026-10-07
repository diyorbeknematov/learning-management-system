package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/password"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/google/uuid"
)

const (
	resetPasswordKeyPrefix = "password_reset:"
	userStatusActive       = "active"

	// After maxLoginFailures wrong passwords for one username, that username
	// cannot log in for loginLockWindow. Somebody who guesses passwords is
	// stopped, and the owner of the account waits a few minutes at worst.
	maxLoginFailures = 5
	loginLockWindow  = 15 * time.Minute

	// A person gets at most maxResetEmails reset emails an hour, so nobody can
	// fill the mailbox of somebody else.
	maxResetEmails  = 3
	resetEmailsSpan = time.Hour
)

type Auth struct {
	repo    *repo.Repository
	tokens  *token.Manager
	redis   core.Redis
	mailer  core.Mailer
	storage core.Storage
	cfg     core.Config

	revocations *core.Revocations
}

func NewAuth(deps core.Dependencies) *Auth {
	return &Auth{
		repo:    deps.Repo,
		tokens:  deps.Tokens,
		redis:   deps.Redis,
		mailer:  deps.Mailer,
		storage: deps.Storage,
		cfg:     deps.Config,

		revocations: core.NewRevocations(deps.Redis, deps.Config.AccessTokenTTL),
	}
}

func (s *Auth) Register(ctx context.Context, req models.RegisterRequest) (*models.TokenPair, error) {
	var err error

	req.Email, err = core.NormalizeEmail(req.Email, "Register")
	if err != nil {
		return nil, err
	}

	req.Username, err = core.NormalizeUsername(req.Username, "Register")
	if err != nil {
		return nil, err
	}

	if err := core.CheckPassword(req.Password, "Register"); err != nil {
		return nil, err
	}

	exists, err := s.repo.User.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, apperror.Conflict("service", "Register", "email already exists", apperror.ErrEmailExists)
	}

	exists, err = s.repo.User.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, apperror.Conflict("service", "Register", "username already exists", apperror.ErrAlreadyExists)
	}

	role, err := s.repo.User.GetRoleByName(ctx, models.RoleStudent)
	if err != nil {
		return nil, err
	}

	roleID, err := uuid.Parse(role.ID)
	if err != nil {
		return nil, apperror.Internal("service", "Register", "invalid role id", err)
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperror.Internal("service", "Register", "failed to hash password", err)
	}

	var pair *models.TokenPair

	// the user and the first refresh token are saved together
	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		userID, err := tx.User.Create(ctx, models.CreateUser{
			RoleID:    role.ID,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Username:  req.Username,
			Email:     req.Email,
			Password:  hash,
		})
		if err != nil {
			return err
		}

		pair, err = s.issueTokens(ctx, tx, userID, roleID, role.Name)

		return err
	})
	if err != nil {
		return nil, err
	}

	return pair, nil
}

func (s *Auth) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	failuresKey := "login_failures:" + req.Username

	if s.tooManyLoginFailures(ctx, failuresKey) {
		return nil, apperror.TooManyRequests(
			"service",
			"Login",
			"too many failed login attempts, try again in a few minutes",
			apperror.ErrTooMany,
		)
	}

	account, err := s.repo.User.GetByUsername(ctx, req.Username)
	if err != nil {
		if core.IsNotFound(err) {
			s.recordLoginFailure(ctx, failuresKey)

			return nil, errInvalidCredentials("Login")
		}

		return nil, err
	}

	// the password is checked before the status, so a blocked account is not
	// revealed to someone who does not know its password
	if !password.Compare(account.Password, req.Password) {
		s.recordLoginFailure(ctx, failuresKey)

		return nil, errInvalidCredentials("Login")
	}

	// the right password ends the count
	_ = s.redis.Del(ctx, failuresKey)

	if account.Status != userStatusActive {
		return nil, apperror.Forbidden("service", "Login", "user is blocked", apperror.ErrForbidden)
	}

	userID, err := uuid.Parse(account.ID)
	if err != nil {
		return nil, apperror.Internal("service", "Login", "invalid user id", err)
	}

	roleID, err := uuid.Parse(account.RoleID)
	if err != nil {
		return nil, apperror.Internal("service", "Login", "invalid role id", err)
	}

	pair, err := s.issueTokens(ctx, s.repo, userID, roleID, account.RoleName)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.User.GetByID(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	user.AvatarURL, err = core.DownloadURL(ctx, s.storage, user.Avatar)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{TokenPair: *pair, User: *user}, nil
}

// Refresh exchanges a refresh token for a new pair. The old refresh token is
// deleted, so every refresh token works once.
func (s *Auth) Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	hash := token.Hash(refreshToken)

	stored, err := s.repo.RefreshToken.GetByTokenHash(ctx, hash)
	if err != nil {
		if core.IsNotFound(err) {
			return nil, errInvalidRefreshToken("Refresh")
		}

		return nil, err
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.repo.RefreshToken.DeleteByTokenHash(ctx, hash)

		return nil, errInvalidRefreshToken("Refresh")
	}

	user, err := s.repo.User.GetByID(ctx, stored.UserID.String())
	if err != nil {
		if core.IsNotFound(err) {
			return nil, errInvalidRefreshToken("Refresh")
		}

		return nil, err
	}

	if user.Status != userStatusActive {
		return nil, apperror.Forbidden("service", "Refresh", "user is blocked", apperror.ErrForbidden)
	}

	var pair *models.TokenPair

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		// when two requests refresh with the same token, only one deletes it
		if err := tx.RefreshToken.DeleteByTokenHash(ctx, hash); err != nil {
			if core.IsNotFound(err) {
				return errInvalidRefreshToken("Refresh")
			}

			return err
		}

		var err error

		pair, err = s.issueTokens(ctx, tx, user.ID, user.RoleID, user.RoleName)

		return err
	})
	if err != nil {
		return nil, err
	}

	return pair, nil
}

// Logout ends the session of the refresh token. An unknown token is not an
// error: the user is logged out either way.
func (s *Auth) Logout(ctx context.Context, refreshToken string) error {
	err := s.repo.RefreshToken.DeleteByTokenHash(ctx, token.Hash(refreshToken))
	if err != nil && !core.IsNotFound(err) {
		return err
	}

	return nil
}

// ForgotPassword emails a one-time link. The reset token is kept in Redis for
// ResetTokenTTL, only its hash is stored.
func (s *Auth) ForgotPassword(ctx context.Context, email string) error {
	email, err := core.NormalizeEmail(email, "ForgotPassword")
	if err != nil {
		return err
	}

	if err := s.checkResetEmailLimit(ctx, email); err != nil {
		return err
	}

	user, err := s.repo.User.GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	raw, err := token.Generate()
	if err != nil {
		return apperror.Internal("service", "ForgotPassword", "failed to generate reset token", err)
	}

	err = s.redis.Set(ctx, resetPasswordKey(raw), user.ID, s.cfg.ResetTokenTTL)
	if err != nil {
		return apperror.Internal("service", "ForgotPassword", "failed to save reset token", err)
	}

	link := s.cfg.ResetPasswordURL + "?token=" + url.QueryEscape(raw)

	body := fmt.Sprintf(
		"Hello %s,\n\nUse this link to reset your password:\n%s\n\nThe link works once and expires in %d minutes. "+
			"If you did not ask for it, ignore this email.",
		user.FirstName,
		link,
		int(s.cfg.ResetTokenTTL.Minutes()),
	)

	if err := s.mailer.Send(ctx, user.Email, "Reset your password", body); err != nil {
		return apperror.Internal("service", "ForgotPassword", "failed to send reset email", err)
	}

	return nil
}

// ResetPassword sets a new password with a reset token and ends all sessions
// of the user.
func (s *Auth) ResetPassword(ctx context.Context, req models.ResetPasswordRequest) error {
	// a weak password is refused before the token is used up
	if err := core.CheckPassword(req.NewPassword, "ResetPassword"); err != nil {
		return err
	}

	var userID uuid.UUID

	err := s.redis.GetDel(ctx, resetPasswordKey(req.Token), &userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return apperror.InvalidInput(
				"service",
				"ResetPassword",
				"invalid or expired reset token",
				apperror.ErrInvalidInput,
			)
		}

		return apperror.Internal("service", "ResetPassword", "failed to read reset token", err)
	}

	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return apperror.Internal("service", "ResetPassword", "failed to hash password", err)
	}

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		_, err := tx.User.Update(ctx, models.UpdateUser{ID: userID, Password: &hash})
		if err != nil {
			return err
		}

		return deleteSessions(ctx, tx, userID)
	})
	if err != nil {
		return err
	}

	return s.revocations.Revoke(ctx, userID)
}

// issueTokens creates an access token and a new refresh token. The refresh
// token is returned as is, only its hash is saved.
func (s *Auth) issueTokens(
	ctx context.Context,
	r *repo.Repository,
	userID, roleID uuid.UUID,
	roleName string,
) (*models.TokenPair, error) {
	access, err := s.tokens.GenerateAccessToken(userID, roleID, roleName)
	if err != nil {
		return nil, apperror.Internal("service", "IssueTokens", "failed to create access token", err)
	}

	refresh, err := token.Generate()
	if err != nil {
		return nil, apperror.Internal("service", "IssueTokens", "failed to create refresh token", err)
	}

	_, err = r.RefreshToken.Create(ctx, models.RefreshToken{
		UserID:    userID,
		TokenHash: token.Hash(refresh),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
	})
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func resetPasswordKey(rawToken string) string {
	return resetPasswordKeyPrefix + token.Hash(rawToken)
}

func errInvalidCredentials(op string) error {
	return apperror.Unauthorized("service", op, "invalid username or password", apperror.ErrUnauthorized)
}

func errInvalidRefreshToken(op string) error {
	return apperror.Unauthorized("service", op, "invalid or expired refresh token", apperror.ErrUnauthorized)
}

// tooManyLoginFailures says whether the username has used up its attempts. When
// Redis cannot be reached the limit is skipped (and logged): an outage of the
// counter must not close the door for everybody.
func (s *Auth) tooManyLoginFailures(ctx context.Context, key string) bool {
	var failures int

	err := s.redis.Get(ctx, key, &failures)
	if err != nil {
		if !errors.Is(err, apperror.ErrNotFound) {
			slog.WarnContext(ctx, "login failures could not be read", "error", err)
		}

		return false
	}

	return failures >= maxLoginFailures
}

func (s *Auth) recordLoginFailure(ctx context.Context, key string) {
	if _, _, err := s.redis.Incr(ctx, key, loginLockWindow); err != nil {
		slog.WarnContext(ctx, "a failed login could not be counted", "error", err)
	}
}

// checkResetEmailLimit counts the request for this address, whether it is a
// real account or not, and refuses the one after the limit. When Redis cannot
// be reached the limit is skipped (the reset itself needs Redis, so it fails
// right after anyway).
func (s *Auth) checkResetEmailLimit(ctx context.Context, email string) error {
	count, _, err := s.redis.Incr(ctx, "forgot_password:"+email, resetEmailsSpan)
	if err != nil {
		slog.WarnContext(ctx, "reset emails could not be counted", "error", err)

		return nil
	}

	if count > maxResetEmails {
		return apperror.TooManyRequests(
			"service",
			"ForgotPassword",
			"too many reset requests for this address, try again later",
			apperror.ErrTooMany,
		)
	}

	return nil
}
