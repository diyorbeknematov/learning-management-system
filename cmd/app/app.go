// Package app builds the whole application from the configuration and runs the
// HTTP server.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/diyorbeknematov/lms/internal/api"
	"github.com/diyorbeknematov/lms/internal/api/authz"
	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/repo/postgres"
	"github.com/diyorbeknematov/lms/internal/service"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/internal/storage/redis"
	"github.com/diyorbeknematov/lms/pkg/mailer"
	"github.com/diyorbeknematov/lms/pkg/token"
)

const (
	shutdownTimeout = 15 * time.Second

	// defaultTokenSecret is the secret of the config when none is set. It is
	// fine on a developer's machine and never in production.
	defaultTokenSecret = "token-secret"
	minSecretLength    = 32
)

// Run connects to the database and the other systems, starts the server and
// stops it gracefully on Ctrl+C or SIGTERM: requests that are running are
// finished first.
func Run(cfg *config.Config, log *slog.Logger) error {
	production := cfg.Logger.Env == "prod"

	if production && (cfg.TokenSecret == defaultTokenSecret || len(cfg.TokenSecret) < minSecretLength) {
		return fmt.Errorf("TOKEN_SECRET must be a random string of at least %d characters in production", minSecretLength)
	}

	db, err := postgres.New(cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	cache, err := redis.New(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer cache.Close()

	files, err := minio.New(cfg.MinIO)
	if err != nil {
		return fmt.Errorf("connect to minio: %w", err)
	}

	mail, err := newMailer(cfg, log, production)
	if err != nil {
		return err
	}

	tokens := token.NewManager(cfg.TokenSecret, cfg.AccessTokenTTL)

	services := service.New(service.Dependencies{
		Repo:    repo.NewRepository(db.Pool),
		Tokens:  tokens,
		Redis:   cache,
		Mailer:  mail,
		Storage: files,
		Config: core.Config{
			RefreshTokenTTL:      cfg.RefreshTokenTTL,
			ResetTokenTTL:        cfg.ResetTokenTTL,
			ResetPasswordURL:     cfg.ResetPasswordURL,
			CertificateVerifyURL: cfg.CertificateVerifyURL,
			AccessTokenTTL:       cfg.AccessTokenTTL,
		},
	})

	if err := ensureSuperAdmin(services, cfg.Admin, log); err != nil {
		return err
	}

	enforcer, err := authz.New()
	if err != nil {
		return err
	}

	routerDeps := api.Dependencies{
		Service:        services,
		Tokens:         tokens,
		Enforcer:       enforcer,
		Logger:         log,
		CORSOrigins:    cfg.CORSOrigins,
		Production:     production,
		Revocations:    core.NewRevocations(cache, cfg.AccessTokenTTL),
		TrustedProxies: cfg.TrustedProxies,
	}

	if cfg.RateLimit {
		routerDeps.Limiter = cache
	}

	server := &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           api.NewRouter(routerDeps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	return serve(server, log)
}

// ensureSuperAdmin creates the first SuperAdmin from ADMIN_* when the system
// has none. Without ADMIN_PASSWORD nothing happens.
func ensureSuperAdmin(services *service.Service, admin config.AdminConfig, log *slog.Logger) error {
	if admin.Password == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	created, err := services.User.EnsureSuperAdmin(ctx, models.CreateUserRequest{
		FirstName: admin.FirstName,
		LastName:  admin.LastName,
		Username:  admin.Username,
		Email:     admin.Email,
		Password:  admin.Password,
	})
	if err != nil {
		return fmt.Errorf("create the first SuperAdmin from ADMIN_*: %w", err)
	}

	if created {
		log.Info("the first SuperAdmin was created", "username", admin.Username)
	}

	return nil
}

// serve runs the server until it fails or the program is told to stop.
func serve(server *http.Server, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	failed := make(chan error, 1)

	go func() {
		log.Info("the server is listening", "address", server.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}

	log.Info("the server is stopping")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("stop the server: %w", err)
	}

	return nil
}

// newMailer connects the SMTP server. Without SMTP settings a developer's
// machine writes the emails to the log (so the reset link can be copied from
// there); production must have a real SMTP server.
func newMailer(cfg *config.Config, log *slog.Logger, production bool) (core.Mailer, error) {
	if cfg.SMTP.Host == "" {
		if production {
			return nil, errors.New("SMTP_HOST must be set in production")
		}

		log.Warn("SMTP_HOST is not set: emails are written to the log, not sent")

		return logMailer{log: log}, nil
	}

	return mailer.New(mailer.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
}

// logMailer writes an email to the log instead of sending it.
type logMailer struct {
	log *slog.Logger
}

func (m logMailer) Send(ctx context.Context, to, subject, body string) error {
	m.log.InfoContext(ctx, "email (not sent)", "to", to, "subject", subject, "body", body)

	return nil
}
