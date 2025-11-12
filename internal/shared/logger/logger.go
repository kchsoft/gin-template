package logger

import (
	"log/slog"
	"os"
)

// Setup configures the global slog logger based on environment
func Setup(env string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	switch env {
	case "production", "prod":
		// Production: JSON format, warn level
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "local", "dev", "development":
		// Development: Text format, debug level
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		// Default: Info level
		opts.Level = slog.LevelInfo
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger 초기화", "env", env, "level", opts.Level.Level().String())
}
