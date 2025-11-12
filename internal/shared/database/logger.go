package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger adapts slog for GORM
type GormLogger struct {
	logger               *slog.Logger
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
	HideSqlInLog         bool
	LogLevel             gormlogger.LogLevel
}

// newLogger creates a new GORM logger with slog
func newLogger(cfg *config.Config) gormlogger.Interface {
	var logLevel gormlogger.LogLevel

	// User requirement: local/dev = info level, prod = error level only
	if cfg.IsProduction() {
		logLevel = gormlogger.Error
	} else {
		// local or dev environment
		logLevel = gormlogger.Info
	}

	return &GormLogger{
		logger:               slog.With("component", "gorm"),
		SlowThreshold:        200 * time.Millisecond,
		IgnoreRecordNotFound: true,               // not logging db level not found
		HideSqlInLog:         cfg.IsProduction(), // Hide query parameters in production
		LogLevel:             logLevel,
	}
}

// LogMode sets the log level
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info level messages
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.logger.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn logs warning level messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.logger.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error logs error level messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.logger.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL queries with timing information
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFound):
		l.logger.ErrorContext(ctx, "Database query error",
			"error", err,
			"elapsed", elapsed.String(),
			"rows", rows,
			"sql", sql,
		)

	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		l.logger.WarnContext(ctx, "Slow SQL query detected",
			"elapsed", elapsed.String(),
			"threshold", l.SlowThreshold.String(),
			"rows", rows,
			"sql", sql,
		)

	case l.LogLevel >= gormlogger.Info:
		if l.HideSqlInLog {
			l.logger.DebugContext(ctx, "SQL query executed",
				"elapsed", elapsed.String(),
				"rows", rows,
			)
		} else {
			l.logger.DebugContext(ctx, "SQL query executed",
				"elapsed", elapsed.String(),
				"rows", rows,
				"sql", sql,
			)
		}
	}
}
