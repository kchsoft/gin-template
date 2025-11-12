package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/bootstrap"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/router"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/database"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/logger"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/validator"
)

func main() {
	// Parse command line flags
	env := parseFlags()

	// Initialize logger
	logger.Setup(env)
	slog.Info("서버 초기화 시작", "env", env)

	// Run application
	if err := run(env); err != nil {
		slog.Error("서버 초기화 실패", "error", err)
		os.Exit(1)
	}

	slog.Info("서버 종료 완료", "env", env)
}

// parseFlags parses command line arguments
func parseFlags() string {
	env := flag.String("env", "local", "Environment (local|dev|production)")
	flag.Parse()
	return *env
}

// run contains the main application logic
func run(env string) error {
	// Create root context for application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load(env)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	slog.Info("환경 변수 로드 성공")

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("데이터베이스 연결 실패: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("데이터베이스 종료 실패", "error", err)
		}
	}()

	// Setup server
	srv := setupServer(cfg, db)

	// Start server with graceful shutdown
	return startWithGracefulShutdown(ctx, srv, cfg.Server.GracefulTimeout)
}

// setupServer initializes and configures the HTTP server
func setupServer(cfg *config.Config, db *database.DB) *bootstrap.Server {
	// Bootstrap server with common setup
	boot := bootstrap.NewBootstrap(cfg)
	ginEngine := boot.SetupEngine()

	// Register common validators
	if err := validator.RegisterAll(); err != nil {
		slog.Error("공통 Validator 등록 실패", "error", err)
		panic(err)
	}

	// Setup application-specific routes
	router.Setup(ginEngine, cfg, db)

	slog.Info("서버 설정 완료",
		"env", cfg.App.Env,
	)

	return bootstrap.New(cfg, ginEngine)
}

// startWithGracefulShutdown starts the server and handles graceful shutdown
func startWithGracefulShutdown(ctx context.Context, srv *bootstrap.Server, gracefulTimeout time.Duration) error {
	// Channel to receive server errors
	serverErrors := make(chan error, 1)

	// Start server in goroutine
	go func() {
		serverErrors <- srv.Start()
	}()

	// Channel to receive OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error or interrupt signal
	select {
	case err := <-serverErrors:
		// Server failed to start or stopped unexpectedly
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("서버 오류: %w", err)
		}
		return nil

	case sig := <-quit:
		// Received shutdown signal
		slog.Info("종료 신호 수신됨", "signal", sig.String())

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, gracefulTimeout)
		defer cancel()

		// Attempt graceful shutdown
		slog.Info("서버 종료 중...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("서버 강제 종료: %w", err)
		}
		return nil
	}
}
