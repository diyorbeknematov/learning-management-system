package main

import (
	"os"

	"github.com/diyorbeknematov/lms/cmd/app"
	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/pkg/logger"
)

// @title Learning Management System API
// @version 1.0
// @description REST API of the learning management system: courses, lessons, quizzes, certificates, payments and reports.
// @description Every answer is {"success": true, "data": ...} or {"success": false, "error": {"code", "message", "fields"}}.
// @description Log in with POST /auth/login, then click "Authorize" and enter: Bearer <access_token>.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Bearer " and the access token
func main() {
	cfg := config.Load()

	log, closeLog := logger.New(cfg.Logger)

	if err := app.Run(cfg, log); err != nil {
		log.Error("the application stopped with an error", "error", err)

		_ = closeLog()

		os.Exit(1)
	}

	_ = closeLog()
}
