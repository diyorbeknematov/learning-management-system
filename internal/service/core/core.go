// Package core holds what every service package shares: the dependencies the
// services are built from and small helpers for authorization and paging.
package core

import (
	"context"
	"time"

	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/pkg/token"
)

// Config holds the values the services need that come from the environment.
type Config struct {
	RefreshTokenTTL time.Duration
	// ResetTokenTTL is how long a password reset link works.
	ResetTokenTTL time.Duration
	// ResetPasswordURL is the frontend page that receives the reset token, the
	// token is appended as "?token=...".
	ResetPasswordURL string
	// CertificateVerifyURL is the public address that checks a certificate;
	// the unique id of the certificate is appended to it. The QR code of the
	// certificate points there.
	CertificateVerifyURL string
	// AccessTokenTTL is how long an access token lives. A revoked token is
	// remembered for this long, no longer: after it the token is dead anyway.
	AccessTokenTTL time.Duration
}

// The interfaces below describe what the services need from outside systems.
// redis.Client and mailer.Mailer satisfy them without any extra code.

type Redis interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	// Incr counts a hit on key within a window and says how many hits the
	// window has and how long it still lasts.
	Incr(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
	// GetDel reads and deletes the key atomically, for one-time values.
	GetDel(ctx context.Context, key string, dest any) error
}

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Storage is the file storage (MinIO). Files are uploaded by the client with
// a presigned URL; the services only read and remove them.
type Storage interface {
	PresignUpload(ctx context.Context, objectKey string) (string, error)
	PresignDownload(ctx context.Context, objectKey string) (string, error)
	Stat(ctx context.Context, objectKey string) (*minio.ObjectInfo, error)
	Copy(ctx context.Context, sourceKey, destinationKey string) error
	Delete(ctx context.Context, objectKey string) error
}

type Dependencies struct {
	Repo    *repo.Repository
	Tokens  *token.Manager
	Redis   Redis
	Mailer  Mailer
	Storage Storage
	Config  Config
}
