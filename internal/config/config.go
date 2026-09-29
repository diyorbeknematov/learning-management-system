package config

import (
	"log"
	"net"
	"os"
	"strconv"
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
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	Bucket          string
	UseSSL          bool
	PresignedExpiry time.Duration
}

type LoggerConfig struct {
	Level  string
	Env    string
	FilePath   string
	ToFile bool
}

type Config struct {
	Server ServerConfig
	DB     DBConfig
	Redis  RedisConfig
	MinIO  MinIOConfig
	Logger LoggerConfig

	AccessTokenSecret  string
	RefreshTokenSecret string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() *Config {
	if err := godotenv.Load("../../../.env"); err != nil {
		log.Println("warning: .env file not found")
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
			DB:       cast.ToInt(coalesce("REDIS_DB", 0)),
		},

		MinIO: MinIOConfig{
			Endpoint:        cast.ToString(coalesce("MINIO_ENDPOINT", "localhost:9000")),
			AccessKey:       cast.ToString(coalesce("MINIO_ACCESS_KEY", "")),
			SecretKey:       cast.ToString(coalesce("MINIO_SECRET_KEY", "")),
			Bucket:          cast.ToString(coalesce("MINIO_BUCKET", "lms")),
			UseSSL:          cast.ToBool(coalesce("MINIO_USE_SSL", false)),
			PresignedExpiry: cast.ToDuration(coalesce("MINIO_PRESIGNED_EXPIRY", "15m")),
		},

		Logger: LoggerConfig{
			Level:  cast.ToString(coalesce("LOG_LEVEL", "info")),
			Env:    cast.ToString(coalesce("APP_ENV", "dev")),
			FilePath:   cast.ToString(coalesce("LOG_FILE_PATH", "logs/app.log")),
			ToFile: cast.ToBool(coalesce("LOG_TO_FILE", true)),
		},

		AccessTokenSecret:  cast.ToString(coalesce("ACCESS_TOKEN_SECRET", "access-secret")),
		RefreshTokenSecret: cast.ToString(coalesce("REFRESH_TOKEN_SECRET", "refresh-secret")),

		AccessTokenTTL:  cast.ToDuration(coalesce("ACCESS_TOKEN_TTL", "15m")),
		RefreshTokenTTL: cast.ToDuration(coalesce("REFRESH_TOKEN_TTL", "168h")),
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
