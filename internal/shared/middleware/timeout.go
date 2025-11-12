package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

const DefaultTimeout = 30 * time.Second

// Timeout middleware sets a timeout context for request processing
// This is the Best Practice implementation using context propagation
// Handlers must check context and handle timeout appropriately
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Store timeout information for handlers to use if needed
		deadline, _ := ctx.Deadline()
		c.Set("request_deadline", deadline)
		c.Set("request_timeout", timeout)

		// Execute the handler chain
		// No goroutine needed - handlers will respect context timeout
		c.Next()

		// After handler completes, check if timeout occurred
		if ctx.Err() == context.DeadlineExceeded {
			// Log the timeout occurrence
			requestID, _ := c.Get(RequestIDKey)

			slog.Warn("Request deadline exceeded",
				"request_id", requestID,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"timeout", timeout.String(),
				"status", c.Writer.Status(),
			)

			// Note: We don't send a response here because:
			// 1. The handler might have already sent a response
			// 2. The handler should check ctx.Err() and handle timeout appropriately
			// 3. This follows Go's context best practices
		}
	}
}

// TimeoutError is a helper function handlers can use to check for timeout
func IsTimeout(c *gin.Context) bool {
	ctx := c.Request.Context()
	return ctx.Err() == context.DeadlineExceeded
}

// GetRequestContext is a helper to get the request context with timeout
func GetRequestContext(c *gin.Context) context.Context {
	return c.Request.Context()
}
