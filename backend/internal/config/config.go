package config

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type ServerConfig struct {
	Host string
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type MinIOConfig struct {
	Endpoint string
	// PublicEndpoint is the address of MinIO as a browser sees it. The links
	// that clients upload to and download from are made for it; when it is not
	// set, Endpoint is used. They differ in Docker, where the API reaches MinIO
	// at minio:9000 and the browser at localhost:9000.
	PublicEndpoint  string
	AccessKey       string
	SecretKey       string
	Bucket          string
	UseSSL          bool
	PresignedExpiry time.Duration
}

type LoggerConfig struct {
	Level    string
	Env      string
	FilePath string
	ToFile   bool
}

// AdminConfig is the first SuperAdmin. When a password is set and the system
// has no SuperAdmin yet, one is created at start-up. It is empty by default.
type AdminConfig struct {
	Username  string
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type Config struct {
	Server ServerConfig
	DB     DBConfig
	Redis  RedisConfig
	MinIO  MinIOConfig
	SMTP   SMTPConfig
	Logger LoggerConfig
	Admin  AdminConfig

	TokenSecret string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	ResetTokenTTL   time.Duration

	// ResetPasswordURL is the page of the frontend that takes the reset token.
	ResetPasswordURL string
	// CertificateVerifyURL is the public address that checks a certificate.
	CertificateVerifyURL string
	// CORSOrigins are the frontend addresses that may call the API.
	CORSOrigins []string
	// TrustedProxies are the reverse proxies in front of the API (addresses or
	// networks like 10.0.0.0/8). Without any, the client address is the
	// address of the connection.
	TrustedProxies []string
	// RateLimit turns the request limits on or off.
	RateLimit bool
}

func Load() *Config {
	if path, ok := findEnvFile(); !ok {
		log.Println("warning: .env file not found")
	} else if err := godotenv.Load(path); err != nil {
		log.Println("warning: .env file not loaded:", err)
	}

	return &Config{
		Server: ServerConfig{
			Host: cast.ToString(coalesce("HTTP_HOST", "0.0.0.0")),
			Port: cast.ToInt(coalesce("HTTP_PORT", 8080)),
		},

		DB: DBConfig{
			Host:     cast.ToString(coalesce("DB_HOST", "localhost")),
			Port:     cast.ToInt(coalesce("DB_PORT", 5432)),
			Name:     cast.ToString(coalesce("DB_NAME", "lms")),
			User:     cast.ToString(coalesce("DB_USER", "postgres")),
			Password: cast.ToString(coalesce("DB_PASSWORD", "")),
		},

		Redis: RedisConfig{
			Host:     cast.ToString(coalesce("REDIS_HOST", "localhost")),
			Port:     cast.ToInt(coalesce("REDIS_PORT", 6379)),
			Password: cast.ToString(coalesce("REDIS_PASSWORD", "")),
			Username: cast.ToString(coalesce("REDIS_USERNAME", "")),
		},

		MinIO: MinIOConfig{
			Endpoint:        cast.ToString(coalesce("MINIO_ENDPOINT", "localhost:9000")),
			PublicEndpoint:  cast.ToString(coalesce("MINIO_PUBLIC_ENDPOINT", "")),
			AccessKey:       cast.ToString(coalesce("MINIO_ACCESS_KEY", "")),
			SecretKey:       cast.ToString(coalesce("MINIO_SECRET_KEY", "")),
			Bucket:          cast.ToString(coalesce("MINIO_BUCKET", "lms")),
			UseSSL:          cast.ToBool(coalesce("MINIO_USE_SSL", false)),
			PresignedExpiry: cast.ToDuration(coalesce("MINIO_PRESIGNED_EXPIRY", "15m")),
		},

		Admin: AdminConfig{
			Username:  cast.ToString(coalesce("ADMIN_USERNAME", "admin")),
			Email:     cast.ToString(coalesce("ADMIN_EMAIL", "")),
			Password:  cast.ToString(coalesce("ADMIN_PASSWORD", "")),
			FirstName: cast.ToString(coalesce("ADMIN_FIRST_NAME", "Super")),
			LastName:  cast.ToString(coalesce("ADMIN_LAST_NAME", "Admin")),
		},

		SMTP: SMTPConfig{
			Host:     cast.ToString(coalesce("SMTP_HOST", "")),
			Port:     cast.ToInt(coalesce("SMTP_PORT", 587)),
			Username: cast.ToString(coalesce("SMTP_USERNAME", "")),
			Password: cast.ToString(coalesce("SMTP_PASSWORD", "")),
			From:     cast.ToString(coalesce("SMTP_FROM", "")),
		},

		Logger: LoggerConfig{
			Level:    cast.ToString(coalesce("LOG_LEVEL", "info")),
			Env:      cast.ToString(coalesce("APP_ENV", "dev")),
			FilePath: cast.ToString(coalesce("LOG_FILE_PATH", "logs/app.log")),
			ToFile:   cast.ToBool(coalesce("LOG_TO_FILE", true)),
		},

		TokenSecret: cast.ToString(coalesce("TOKEN_SECRET", "token-secret")),

		AccessTokenTTL:  cast.ToDuration(coalesce("ACCESS_TOKEN_TTL", "15m")),
		RefreshTokenTTL: cast.ToDuration(coalesce("REFRESH_TOKEN_TTL", "168h")),
		ResetTokenTTL:   cast.ToDuration(coalesce("RESET_TOKEN_TTL", "15m")),

		ResetPasswordURL:     cast.ToString(coalesce("RESET_PASSWORD_URL", "http://localhost:3000/reset-password")),
		CertificateVerifyURL: cast.ToString(coalesce("CERTIFICATE_VERIFY_URL", "http://localhost:8080/api/v1/certificates/verify")),
		CORSOrigins:          splitList(cast.ToString(coalesce("CORS_ORIGINS", "http://localhost:3000"))),
		TrustedProxies:       splitList(cast.ToString(coalesce("TRUSTED_PROXIES", ""))),
		RateLimit:            cast.ToBool(coalesce("RATE_LIMIT_ENABLED", true)),
	}
}

// findEnvFile looks for .env in the working directory and then in each parent
// directory, so it is found whether the app or a test in a subfolder runs.
func findEnvFile() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		path := filepath.Join(dir, ".env")

		if _, err := os.Stat(path); err == nil {
			return path, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}

		dir = parent
	}
}

func coalesce(key string, defaultValue any) any {
	value, exists := os.LookupEnv(key)

	if !exists || value == "" {
		return defaultValue
	}

	return value
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// splitList splits a comma separated value and drops the empty parts.
func splitList(value string) []string {
	var items []string

	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}
