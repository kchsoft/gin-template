package middleware

import (
	"log/slog"
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/logger"
	"github.com/gin-gonic/gin"
)

// LoggerMiddleware returns a gin middleware for structured logging with slog
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get request ID
		requestID := GetRequestID(c)

		// Create logger with request_id bound
		reqLogger := slog.Default().With("request_id", requestID)

		// Store logger in context for use in handlers/services/repositories
		ctx := logger.WithLogger(c.Request.Context(), reqLogger)
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Writer.Status()

		// Build log fields (request_id will be automatically added by reqLogger)
		fields := []any{
			"method", c.Request.Method, // Core info
			"path", path, // Core info
			"status", status, // Core info
			"latency", latency.String(), // Performance info
			"ip", c.ClientIP(), // Additional info
			"userAgent", c.Request.UserAgent(), // Additional info
		}

		if raw != "" {
			fields = append(fields, "query", raw)
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			fields = append(fields, "error", c.Errors.String())
		}

		// Log based on status code using the request logger (request_id automatically included)
		msg := "Request processed"

		switch {
		case status >= 500:
			reqLogger.Error(msg, fields...)
		case status >= 400:
			reqLogger.Warn(msg, fields...)
		case status >= 300:
			reqLogger.Info(msg, fields...)
		default:
			reqLogger.Info(msg, fields...)
		}
	}
}
