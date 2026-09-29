package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	RoleID    uuid.UUID `db:"role_id" json:"role_id"`
	RoleName  string    `db:"role_name" json:"role_name"`
	Password  string    `db:"password" json:"-"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Role struct {
	ID   string `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

type CreateUser struct {
	RoleID    string `db:"role_id" json:"role_id"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
	Username  string `db:"username" json:"username"`
	Email     string `db:"email" json:"email"`
	Password  string `db:"password" json:"password"`
}

type UpdateUser struct {
	ID        uuid.UUID `db:"id" json:"id"`
	FirstName *string   `db:"first_name" json:"first_name"`
	LastName  *string   `db:"last_name" json:"last_name"`
	Username  *string   `db:"username" json:"usename"`
	Email     *string   `db:"email" json:"email"`
	Password  *string   `db:"password" json:"password"`
}

type UserFilter struct {
	Search *string `json:"search"`
	RoleName *string `json:"role_id"`
	Status *string `json:"status"`
	Limit  int     `json:"limit"`
	Page   int     `json:"page"`
}
