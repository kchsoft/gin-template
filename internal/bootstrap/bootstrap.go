package bootstrap

import (
	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/middleware"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
)

// Bootstrap handles common server setup that can be reused across projects
type Bootstrap struct {
	cfg *config.Config
}

// NewBootstrap creates a new bootstrap instance
func NewBootstrap(cfg *config.Config) *Bootstrap {
	return &Bootstrap{
		cfg: cfg,
	}
}

// SetupEngine creates and configures a gin engine with common middleware
// This is reusable across different projects
func (b *Bootstrap) SetupEngine() *gin.Engine {
	// Set Gin mode based on environment
	if b.cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Disable Gin's default logger (using slog)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	// Create engine without default middleware
	engine := gin.New()

	// Essential middleware (common for all projects)
	engine.Use(gin.CustomRecovery(b.recoveryHandler))
	engine.Use(middleware.RequestID())
	engine.Use(middleware.CORS(b.cfg))
	engine.Use(middleware.Timeout(middleware.DefaultTimeout)) // 30 second global timeout
	engine.Use(middleware.LoggerMiddleware())

	// Note: Health endpoints are now handled in routes.go following Clean Architecture
	// This keeps the bootstrap focused on middleware setup only

	return engine
}

// recoveryHandler handles panics
func (b *Bootstrap) recoveryHandler(c *gin.Context, recovered interface{}) {
	if err, ok := recovered.(string); ok {
		slog.Error("Panic Recovered",
			"error", err,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"request_id", middleware.GetRequestID(c),
		)
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": "Internal server error", "request_id": middleware.GetRequestID(c),
	})
}
