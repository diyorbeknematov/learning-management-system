package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/diyorbeknematov/lms/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(cfg config.LoggerConfig) (*slog.Logger, func() error) {
	var writer io.Writer = os.Stdout

	closeFn := func() error {
		return nil
	}

	if cfg.ToFile {
		fileWriter := &lumberjack.Logger{
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		}

		writer = io.MultiWriter(os.Stdout, fileWriter)

		closeFn = fileWriter.Close
	}

	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.Level),
		AddSource: cfg.Env != "prod",
	}

	var handler slog.Handler

	if cfg.Env == "prod" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)

	slog.SetDefault(logger)

	return logger, closeFn
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
