package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, tc *TestContext) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	var roleID uuid.UUID

	err := tc.DB.Pool.QueryRow(
		ctx,
		`SELECT id FROM roles WHERE name = 'Student'`,
	).Scan(&roleID)

	require.NoError(t, err)

	user := models.CreateUser{
		RoleID:    roleID.String(),
		FirstName: "Diyorbek",
		LastName:  "Nematov",
		Username:  "test_" + uuid.NewString(),
		Email:     uuid.NewString() + "@example.com",
		Password:  "password123",
	}

	id, err := tc.Repo.Create(ctx, user)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	return id
}

func deleteTestUser(t *testing.T, tc *TestContext, id uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	_, err := tc.DB.Pool.Exec(
		ctx,
		`DELETE FROM users WHERE id = $1`,
		id,
	)

	require.NoError(t, err)
}

func TestUserRepo_Create(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	var roleID uuid.UUID

	err := tc.DB.Pool.QueryRow(
		ctx,
		`SELECT id FROM roles WHERE name = 'Student'`,
	).Scan(&roleID)

	require.NoError(t, err)

	user := models.CreateUser{
		RoleID:    roleID.String(),
		FirstName: "Diyorbek",
		LastName:  "Nematov",
		Username:  "test_" + uuid.NewString(),
		Email:     uuid.NewString() + "@example.com",
		Password:  "password123",
	}

	id, err := tc.Repo.Create(ctx, user)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	var username string

	err = tc.DB.Pool.QueryRow(
		ctx,
		`SELECT username FROM users WHERE id = $1`,
		id,
	).Scan(&username)

	require.NoError(t, err)
	require.Equal(t, user.Username, username)
}

func TestUserRepo_Update(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestUser(t, tc)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	firstName := "Ali"
	lastName := "Valiyev"
	username := "ali_" + uuid.NewString()
	email := uuid.NewString() + "@example.com"
	password := "newpassword123"

	updateData := models.UpdateUser{
		ID:        id,
		FirstName: &firstName,
		LastName:  &lastName,
		Username:  &username,
		Email:     &email,
		Password:  &password,
	}

	updatedUser, err := tc.Repo.Update(ctx, updateData)

	require.NoError(t, err)
	require.NotNil(t, updatedUser)

	require.Equal(t, id, updatedUser.ID)
	require.Equal(t, firstName, updatedUser.FirstName)
	require.Equal(t, lastName, updatedUser.LastName)
	require.Equal(t, username, updatedUser.Username)
	require.Equal(t, email, updatedUser.Email)
}

func TestUserRepo_UpdateStatus(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestUser(t, tc)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	err := tc.Repo.UpdateStatus(
		ctx,
		id.String(),
		"blocked",
	)

	require.NoError(t, err)

	var status string

	err = tc.DB.Pool.QueryRow(
		ctx,
		`SELECT status FROM users WHERE id = $1`,
		id,
	).Scan(&status)

	require.NoError(t, err)
	require.Equal(t, "blocked", status)
}

func TestUserRepo_GetByID(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestUser(t, tc)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	user, err := tc.Repo.GetByID(ctx, id.String())

	require.NoError(t, err)
	require.NotNil(t, user)

	require.Equal(t, id, user.ID)
	require.Equal(t, "Diyorbek", user.FirstName)
	require.Equal(t, "Nematov", user.LastName)
	require.Equal(t, "Student", user.RoleName)
	require.Equal(t, "active", user.Status)
}

func TestUserRepo_GetList(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestUser(t, tc)

	t.Cleanup(func() {
		deleteTestUser(t, tc, id)
	})

	filter := models.UserFilter{
		Page:  1,
		Limit: 10,
	}

	users, total, err := tc.Repo.GetList(ctx, filter)

	require.NoError(t, err)
	require.NotNil(t, users)
	require.GreaterOrEqual(t, total, 1)

	var found bool

	for _, user := range users {
		if user.ID == id {
			found = true

			require.Equal(t, "Diyorbek", user.FirstName)
			require.Equal(t, "Nematov", user.LastName)
			require.Equal(t, "Student", user.RoleName)
			require.Equal(t, "active", user.Status)

			break
		}
	}

	require.True(t, found)
}

func TestUserRepo_Delete(t *testing.T) {
	tc := setupTest(t)

	ctx := context.Background()

	id := createTestUser(t, tc)

	err := tc.Repo.Delete(ctx, id.String())

	require.NoError(t, err)

	var deletedAt *time.Time

	err = tc.DB.Pool.QueryRow(
		ctx,
		`SELECT deleted_at FROM users WHERE id = $1`,
		id,
	).Scan(&deletedAt)

	require.NoError(t, err)
	require.NotNil(t, deletedAt)
}

