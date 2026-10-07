package token

import (
	"time"

	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	RoleID   uuid.UUID `json:"role_id"`
	RoleName string    `json:"role_name"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *Manager) GenerateAccessToken(userID, roleID uuid.UUID, roleName string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:   userID,
		RoleID:   roleID,
		RoleName: roleName,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ParseAccessToken checks the signature and the expiry and returns the claims.
func (m *Manager) ParseAccessToken(raw string) (*Claims, error) {
	var claims Claims

	parsed, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return nil, apperror.Unauthorized(
			"token",
			"ParseAccessToken",
			"invalid or expired token",
			apperror.ErrUnauthorized,
		)
	}

	return &claims, nil
}
