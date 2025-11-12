package testutil

import (
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
)

// NewTestConfig creates a test configuration
// This removes the need for environment variables during testing
func NewTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name: "pray-together-api-test",
			Env:  "test",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            1521,
			Service:         "test",
			User:            "test",
			Password:        "test",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
			IsAutoMigrate:   true,
		},
		JWT: config.JWTConfig{
			Secret:        "test-jwt-secret-key-must-be-at-least-32-characters-long",
			Expiry:        24 * time.Hour,
			RefreshExpiry: 168 * time.Hour,
		},
		CORS: config.CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
			MaxAge:           86400,
		},
		Server: config.ServerConfig{
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			GracefulTimeout: 30 * time.Second,
		},
	}
}
