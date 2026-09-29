package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

type userRepo struct {
	db DBTX
}

func NewUserRepository(db DBTX) *userRepo {
	return &userRepo{
		db: db,
	}
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
			status,
			created_at, 
			updated_at
		)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
	`

	id := uuid.New()
	_, err := r.db.Exec(
		ctx,
		query,
		id.String(),
		user.Username,
		user.Password,
		user.FirstName,
		user.LastName,
		user.Email,
		user.RoleID,
		"active",
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return uuid.Nil, apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"CreateUser",
			"failed to crete user",
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
			updated_at = NOW()
		WHERE deleted_at IS NULL AND id = $1
		RETURNING 
			id,
			username,
			first_name, 
			last_name, 
			email, 
			updated_at;
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		updateData.ID,
		updateData.FirstName,
		updateData.LastName,
		updateData.Username,
		updateData.Email,
		updateData.Password,
	).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"UpdateUser",
			"failed to update user",
			err,
		)
	}

	return &user, nil
}

func (r *userRepo) UpdateStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	query := `
		UPDATE users 
		SET
			status = $2
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
		return apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"UpdateUserStatus",
			"failed to update user status",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.Wrap(
			apperror.CodeNotFound,
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
			deleted_at = NOW()
		WHERE deleted_at IS NULL 
			AND id = $1
	`

	res, err := r.db.Exec(
		ctx,
		query,
		id,
	)
	if err != nil {
		return apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"DeleteUser",
			"failed to delete user",
			err,
		)
	}

	if res.RowsAffected() == 0 {
		return apperror.Wrap(
			apperror.CodeNotFound,
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
		&user.RoleID,
		&user.RoleName,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"GetUserByID",
			"failed to get user by id",
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
				)`,
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

	limit := filter.Limit
	offset := (filter.Page - 1) * filter.Limit

	baseQuery += fmt.Sprintf(
		" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d",
		argIndex,
		argIndex+1,
	)

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, listArgs...)
	if err != nil {
		return nil, 0, apperror.Wrap(
			apperror.CodeInternal,
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
			&user.RoleID,
			&user.RoleName,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(
				apperror.CodeInternal,
				"repository",
				"GetUserList",
				"failed to scan users",
				err,
			)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Wrap(
			apperror.CodeInternal,
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
		return nil, 0, apperror.Wrap(
			apperror.CodeInternal,
			"repository",
			"GetUserList",
			"failed to get total count",
			err,
		)
	}

	return users, total, nil
}
