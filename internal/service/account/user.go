package account

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/password"
	"github.com/google/uuid"
)

const userStatusBlocked = "blocked"

type User struct {
	repo        *repo.Repository
	storage     core.Storage
	revocations *core.Revocations
}

func NewUser(deps core.Dependencies) *User {
	return &User{
		repo:        deps.Repo,
		storage:     deps.Storage,
		revocations: core.NewRevocations(deps.Redis, deps.Config.AccessTokenTTL),
	}
}

// Create lets a SuperAdmin add a user with any role, for example an
// instructor (the public registration gives only the Student role).
func (s *User) Create(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	var err error

	req.Email, err = core.NormalizeEmail(req.Email, "CreateUser")
	if err != nil {
		return nil, err
	}

	req.Username, err = core.NormalizeUsername(req.Username, "CreateUser")
	if err != nil {
		return nil, err
	}

	if err := core.CheckPassword(req.Password, "CreateUser"); err != nil {
		return nil, err
	}

	if req.Avatar != nil {
		if _, err := core.CheckUpload(ctx, s.storage, models.UploadPurposeAvatar, req.Avatar, "CreateUser"); err != nil {
			return nil, err
		}
	}

	exists, err := s.repo.User.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, apperror.Conflict("service", "CreateUser", "email already exists", apperror.ErrEmailExists)
	}

	exists, err = s.repo.User.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, apperror.Conflict("service", "CreateUser", "username already exists", apperror.ErrAlreadyExists)
	}

	role, err := s.repo.User.GetRoleByName(ctx, req.Role)
	if err != nil {
		return nil, err
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, apperror.Internal("service", "CreateUser", "failed to hash password", err)
	}

	id, err := s.repo.User.Create(ctx, models.CreateUser{
		RoleID:    role.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Password:  hash,
		Avatar:    req.Avatar,
		Bio:       req.Bio,
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// EnsureSuperAdmin creates the first SuperAdmin when the system has none, and
// does nothing otherwise, so it is safe to call at every start. It reports
// whether it created one.
func (s *User) EnsureSuperAdmin(ctx context.Context, req models.CreateUserRequest) (bool, error) {
	role := models.RoleSuperAdmin

	_, total, err := s.repo.User.GetList(ctx, models.UserFilter{RoleName: &role, Limit: 1, Page: 1})
	if err != nil {
		return false, err
	}

	if total > 0 {
		return false, nil
	}

	req.Role = models.RoleSuperAdmin

	if _, err := s.Create(ctx, req); err != nil {
		return false, err
	}

	return true, nil
}

func (s *User) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.repo.User.GetByID(ctx, id.String())
	if err != nil {
		return nil, err
	}

	return user, s.withAvatarURL(ctx, user)
}

// withAvatarURL adds the temporary link to the avatar of the user.
func (s *User) withAvatarURL(ctx context.Context, user *models.User) error {
	var err error

	user.AvatarURL, err = core.DownloadURL(ctx, s.storage, user.Avatar)

	return err
}

// replaceAvatar checks a new avatar before the user is changed, and returns a
// function that removes the old file once the change is saved.
func (s *User) replaceAvatar(ctx context.Context, userID uuid.UUID, avatar *string, op string) (cleanup func(), err error) {
	if avatar == nil {
		return func() {}, nil
	}

	current, err := s.repo.User.GetByID(ctx, userID.String())
	if err != nil {
		return nil, err
	}

	if _, err := core.CheckUpload(ctx, s.storage, models.UploadPurposeAvatar, avatar, op); err != nil {
		return nil, err
	}

	return func() {
		// the old file is garbage now; failing to remove it is not worth failing the update
		if current.Avatar != nil && *current.Avatar != *avatar {
			_ = s.storage.Delete(ctx, *current.Avatar)
		}
	}, nil
}

func (s *User) GetList(ctx context.Context, filter models.UserFilter) (*models.ListResponse[models.User], error) {
	users, total, err := s.repo.User.GetList(ctx, filter)
	if err != nil {
		return nil, err
	}

	for i := range users {
		if err := s.withAvatarURL(ctx, &users[i]); err != nil {
			return nil, err
		}
	}

	response := core.NewListResponse(users, total, filter.Page, filter.Limit)

	return &response, nil
}

// Update lets a SuperAdmin change any user, including the role. Nobody can
// change their own role this way.
func (s *User) Update(
	ctx context.Context,
	actor models.Actor,
	id uuid.UUID,
	req models.UpdateUserRequest,
) (*models.User, error) {
	email, err := normalizeOptionalEmail(req.Email, "UpdateUser")
	if err != nil {
		return nil, err
	}

	username, err := normalizeOptionalUsername(req.Username, "UpdateUser")
	if err != nil {
		return nil, err
	}

	update := models.UpdateUser{
		ID:        id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  username,
		Email:     email,
		Avatar:    req.Avatar,
		Bio:       req.Bio,
	}

	if req.Role != nil {
		if actor.UserID == id {
			return nil, apperror.Forbidden(
				"service",
				"UpdateUser",
				"you cannot change your own role",
				apperror.ErrForbidden,
			)
		}

		role, err := s.repo.User.GetRoleByName(ctx, *req.Role)
		if err != nil {
			return nil, err
		}

		roleID, err := uuid.Parse(role.ID)
		if err != nil {
			return nil, apperror.Internal("service", "UpdateUser", "invalid role id", err)
		}

		update.RoleID = &roleID
	}

	cleanup, err := s.replaceAvatar(ctx, id, req.Avatar, "UpdateUser")
	if err != nil {
		return nil, err
	}

	user, err := s.repo.User.Update(ctx, update)
	if err != nil {
		return nil, err
	}

	cleanup()

	// the role is inside the access token, so the old tokens must not be used
	if update.RoleID != nil {
		if err := s.revocations.Revoke(ctx, id); err != nil {
			return nil, err
		}
	}

	return user, s.withAvatarURL(ctx, user)
}

// UpdateStatus blocks or unblocks a user. Blocking also ends the sessions of
// the user, but an access token that was already issued works until it
// expires. Nobody can block themselves.
func (s *User) UpdateStatus(ctx context.Context, actor models.Actor, req models.UpdateUserStatus) error {
	if actor.UserID == req.ID {
		return apperror.Forbidden(
			"service",
			"UpdateUserStatus",
			"you cannot change your own status",
			apperror.ErrForbidden,
		)
	}

	err := s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if err := tx.User.UpdateStatus(ctx, req.ID.String(), req.Status); err != nil {
			return err
		}

		if req.Status == userStatusBlocked {
			return deleteSessions(ctx, tx, req.ID)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// a blocked user is thrown out at once; the unblocked one can come back
	if req.Status == userStatusBlocked {
		return s.revocations.Revoke(ctx, req.ID)
	}

	return s.revocations.Reinstate(ctx, req.ID)
}

// Delete removes a user (soft delete) and ends their sessions. Nobody can
// delete themselves.
func (s *User) Delete(ctx context.Context, actor models.Actor, id uuid.UUID) error {
	if actor.UserID == id {
		return apperror.Forbidden(
			"service",
			"DeleteUser",
			"you cannot delete yourself",
			apperror.ErrForbidden,
		)
	}

	err := s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		if err := tx.User.Delete(ctx, id.String()); err != nil {
			return err
		}

		return deleteSessions(ctx, tx, id)
	})
	if err != nil {
		return err
	}

	return s.revocations.Revoke(ctx, id)
}

func (s *User) GetMe(ctx context.Context, actor models.Actor) (*models.User, error) {
	return s.GetByID(ctx, actor.UserID)
}

func (s *User) UpdateMe(ctx context.Context, actor models.Actor, req models.UpdateMeRequest) (*models.User, error) {
	email, err := normalizeOptionalEmail(req.Email, "UpdateMe")
	if err != nil {
		return nil, err
	}

	username, err := normalizeOptionalUsername(req.Username, "UpdateMe")
	if err != nil {
		return nil, err
	}

	cleanup, err := s.replaceAvatar(ctx, actor.UserID, req.Avatar, "UpdateMe")
	if err != nil {
		return nil, err
	}

	user, err := s.repo.User.Update(ctx, models.UpdateUser{
		ID:        actor.UserID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  username,
		Email:     email,
		Avatar:    req.Avatar,
		Bio:       req.Bio,
	})
	if err != nil {
		return nil, err
	}

	cleanup()

	return user, s.withAvatarURL(ctx, user)
}

// ChangePassword checks the old password and sets the new one. All sessions
// end, so the user logs in again with the new password.
func (s *User) ChangePassword(ctx context.Context, actor models.Actor, req models.ChangePasswordRequest) error {
	if err := core.CheckPassword(req.NewPassword, "ChangePassword"); err != nil {
		return err
	}

	current, err := s.repo.User.GetPasswordHash(ctx, actor.UserID.String())
	if err != nil {
		return err
	}

	if !password.Compare(current, req.OldPassword) {
		return apperror.InvalidInput(
			"service",
			"ChangePassword",
			"old password is incorrect",
			apperror.ErrInvalidInput,
		)
	}

	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return apperror.Internal("service", "ChangePassword", "failed to hash password", err)
	}

	err = s.repo.WithTx(ctx, func(tx *repo.Repository) error {
		_, err := tx.User.Update(ctx, models.UpdateUser{ID: actor.UserID, Password: &hash})
		if err != nil {
			return err
		}

		return deleteSessions(ctx, tx, actor.UserID)
	})
	if err != nil {
		return err
	}

	return s.revocations.Revoke(ctx, actor.UserID)
}

// normalizeOptionalEmail normalizes an email that may be missing.
func normalizeOptionalEmail(email *string, op string) (*string, error) {
	if email == nil {
		return nil, nil
	}

	normalized, err := core.NormalizeEmail(*email, op)
	if err != nil {
		return nil, err
	}

	return &normalized, nil
}

// normalizeOptionalUsername checks a username that may be missing.
func normalizeOptionalUsername(username *string, op string) (*string, error) {
	if username == nil {
		return nil, nil
	}

	normalized, err := core.NormalizeUsername(*username, op)
	if err != nil {
		return nil, err
	}

	return &normalized, nil
}
