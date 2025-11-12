package meta

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/database"
	"github.com/gin-gonic/gin"
)

// Handler handles meta endpoints (health check, app version, legal documents, etc.)
type Handler struct {
	cfg *config.Config
	db  *database.DB
}

// NewHandler creates a new meta handler
func NewHandler(cfg *config.Config, db *database.DB) *Handler {
	return &Handler{
		cfg: cfg,
		db:  db,
	}
}

// Health checks service and database health
func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database connectivity
	dbStatus := "up"
	var dbError string
	start := time.Now()

	if err := h.db.HealthCheck(ctx); err != nil {
		dbStatus = "down"
		dbError = err.Error()
		slog.Error("Health check 실패", "error", err)

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"service": gin.H{
				"name":        h.cfg.App.Name,
				"environment": h.cfg.App.Env,
			},
			"checks": gin.H{
				"database": gin.H{
					"status": dbStatus,
					"error":  dbError,
				},
			},
		})
		return
	}

	dbLatency := time.Since(start).Milliseconds()

	// All checks passed
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": gin.H{
			"name":        h.cfg.App.Name,
			"environment": h.cfg.App.Env,
			"port":        h.cfg.App.Port,
		},
		"checks": gin.H{
			"database": gin.H{
				"status":     dbStatus,
				"latency_ms": dbLatency,
			},
		},
	})
}
