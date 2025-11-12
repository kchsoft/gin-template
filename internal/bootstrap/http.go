package bootstrap

import (
	"context"
	"fmt"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"log/slog"
	"net/http"
)

// Server represents the HTTP server (lifecycle management only)
type Server struct {
	cfg    *config.Config
	server *http.Server
}

// New creates a new server instance with the provided handler
func New(cfg *config.Config, handler http.Handler) *Server {
	return &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:           fmt.Sprintf(":%d", cfg.App.Port),
			Handler:        handler,
			ReadTimeout:    cfg.Server.ReadTimeout,
			WriteTimeout:   cfg.Server.WriteTimeout,
			IdleTimeout:    cfg.Server.IdleTimeout,
			MaxHeaderBytes: 1 << 20, // 1 MB
		},
	}
}

// Port returns the server port
func (s *Server) Port() int {
	return s.cfg.App.Port
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("서버 시작 중",
		"port", s.cfg.App.Port,
		"env", s.cfg.App.Env,
		"read_timeout", s.cfg.Server.ReadTimeout,
		"write_timeout", s.cfg.Server.WriteTimeout,
	)

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	err := s.server.Shutdown(ctx)
	return err
}
