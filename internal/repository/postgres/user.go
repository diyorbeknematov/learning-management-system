package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/pgerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userRepo struct {
	db DBTX
}

func NewUserRepository(db DBTX) *userRepo {
	return &userRepo{
		db: db,
	}
}

func (r *userRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users 
			WHERE email = $1 
				AND deleted_at IS NULL
		);
	`
	var exists bool
	err := r.db.QueryRow(
		ctx,
		query,
		email,
	).Scan(&exists)

	if err != nil {
		return false, apperror.Internal(
			"repository",
			"ExistsByEmail",
			"failed to check weather email exists",
			err,
		)
	}

	return exists, nil
}

func (r *userRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE username = $1
				AND deleted_at IS NULL
		);
	`

	var exists bool
	err := r.db.QueryRow(
		ctx,
		query,
		username,
	).Scan(&exists)

	if err != nil {
		return false, apperror.Internal(
			"repository",
			"ExistsByUsername",
			"failed to check weather username exists",
			err,
		)
	}

	return exists, err
}

func (r *userRepo) Create(ctx context.Context, user models.CreateUser) (uuid.UUID, error) {
	query := `
		INSERT INTO users (
			id,
			username,
			password,
			first_name,
			last_name,
			email,
			role_id,
			avatar,
			bio
		)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`

	id := uuid.New()
	_, err := r.db.Exec(
		ctx,
		query,
		id,
		user.Username,
		user.Password,
		user.FirstName,
		user.LastName,
		user.Email,
		user.RoleID,
		user.Avatar,
		user.Bio,
	)

	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return uuid.Nil, apperror.Conflict(
				"repository",
				"CreateUser",
				"email or username already exists",
				apperror.ErrAlreadyExists,
			)
		}

		return uuid.Nil, apperror.Internal(
			"repository",
			"CreateUser",
			"failed to create user",
			err,
		)
	}

	return id, nil
}

func (r *userRepo) Update(ctx context.Context, updateData models.UpdateUser) (*models.User, error) {
	query := `
		UPDATE users
		SET
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			username = COALESCE($4, username),
			email = COALESCE($5, email),
			password = COALESCE($6, password),
			avatar = COALESCE($7, avatar),
			bio = COALESCE($8, bio),
			updated_at = CURRENT_TIMESTAMP
		WHERE deleted_at IS NULL AND id = $1;
	`

	res, err := r.db.Exec(
		ctx,
		query,
		updateData.ID,
		updateData.FirstName,
		updateData.LastName,
		updateData.Username,
		updateData.Email,
		updateData.Password,
		updateData.Avatar,
		updateData.Bio,
	)
	if err != nil {
		if pgerr.IsUniqueViolation(err) {
			return nil, apperror.Conflict(
				"repository",
				"UpdateUser",
				"email or username already exists",
				apperror.ErrAlreadyExists,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"UpdateUser",
			"failed to update user",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return nil, apperror.NotFound(
			"repository",
			"UpdateUser",
			"user not found",
			apperror.ErrUserNotFound,
		)
	}

	return r.GetByID(ctx, updateData.ID.String())
}

func (r *userRepo) UpdateStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	query := `
		UPDATE users 
		SET
			status = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE deleted_at IS NULL 
			AND id = $1
	`

	res, err := r.db.Exec(
		ctx,
		query,
		id,
		status,
	)
	if err != nil {
		return apperror.Internal(
			"repository",
			"UpdateUserStatus",
			"failed to update user status",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"UpdateUserStatus",
			"user not found",
			apperror.ErrUserNotFound,
		)
	}

	return nil
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE users 
		SET 
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE deleted_at IS NULL 
			AND id = $1
	`

	res, err := r.db.Exec(
		ctx,
		query,
		id,
	)
	if err != nil {
		return apperror.Internal(
			"repository",
			"DeleteUser",
			"failed to delete user",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.NotFound(
			"repository",
			"DeleteUser",
			"user not found",
			apperror.ErrUserNotFound,
		)
	}

	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT 
			u.id,
			u.first_name,
			u.last_name,
			u.username,
			u.email,
			u.avatar,
			u.bio,
			u.role_id,
			r.name AS role_name,
			u.status,
			u.created_at,
			u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL 
			AND u.id = $1;
	`
	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Email,
		&user.Avatar,
		&user.Bio,
		&user.RoleID,
		&user.RoleName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetUserByID",
				"user not found",
				apperror.ErrUserNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetUserByID",
			"failed to get user by id",
			err,
		)
	}

	return &user, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*models.GetByUsername, error) {
	query := `
		SELECT 
			u.id,
			u.username,
			u.password,
			u.status,
			u.role_id,
			r.name AS role_name
		FROM users u 
		INNER JOIN roles r 
			ON u.role_id = r.id 
		WHERE u.username = $1
			AND u.deleted_at IS NULL;
	`

	var user models.GetByUsername

	err := r.db.QueryRow(
		ctx,
		query,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Status,
		&user.RoleID,
		&user.RoleName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound(
				"repository",
				"GetUserByUsername",
				"user not found",
				apperror.ErrUserNotFound,
			)
		}

		return nil, apperror.Internal(
			"repository",
			"GetUserByUsername",
			"failed to get user by username",
			err,
		)
	}

	return &user, nil
}

func (r *userRepo) GetList(
	ctx context.Context,
	filter models.UserFilter,
) ([]models.User, int, error) {
	baseQuery := `
		SELECT 
			u.id,
			u.first_name,
			u.last_name,
			u.username,
			u.email,
			u.avatar,
			u.bio,
			u.role_id,
			r.name AS role_name,
			u.status,
			u.created_at,
			u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL
	`

	countQuery := `
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL
	`

	conditions := []string{}
	args := []any{}
	argIndex := 1

	if filter.Search != nil {
		conditions = append(
			conditions,
			fmt.Sprintf(
				`(
					u.username ILIKE $%d
					OR u.first_name ILIKE $%d
					OR u.last_name ILIKE $%d
					OR u.email ILIKE $%d
				)`,
				argIndex,
				argIndex,
				argIndex,
				argIndex,
			),
		)

		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("u.status = $%d", argIndex),
		)

		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.RoleName != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("r.name = $%d", argIndex),
		)

		args = append(args, *filter.RoleName)
		argIndex++
	}

	if len(conditions) > 0 {
		whereClause := " AND " + strings.Join(conditions, " AND ")

		baseQuery += whereClause
		countQuery += whereClause
	}

	limit, offset := helpers.Pagination(filter.Page, filter.Limit)

	baseQuery += fmt.Sprintf(
		" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetUserList",
			"failed to get users",
			err,
		)
	}
	defer rows.Close()

	users := make([]models.User, 0)

	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.Email,
			&user.Avatar,
			&user.Bio,
			&user.RoleID,
			&user.RoleName,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Internal(
				"repository",
				"GetUserList",
				"failed to scan users",
				err,
			)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetUserList",
			"failed to read users",
			err,
		)
	}

	var total int

	err = r.db.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, 0, apperror.Internal(
			"repository",
			"GetUserList",
			"failed to get total count",
			err,
		)
	}

	return users, total, nil
}
