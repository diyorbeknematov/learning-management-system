package models

import "github.com/google/uuid"

const (
	RoleSuperAdmin = "SuperAdmin"
	RoleInstructor = "Instructor"
	RoleStudent    = "Student"
)

// Actor is the authenticated user who makes a request. The middleware builds
// it from the access token and the handler passes it to the service, which
// decides whether this user may act on a specific resource.
type Actor struct {
	UserID   uuid.UUID
	RoleName string
}

func (a Actor) IsSuperAdmin() bool {
	return a.RoleName == RoleSuperAdmin
}
