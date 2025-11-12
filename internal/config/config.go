package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
	Server   ServerConfig
}

type AppConfig struct {
	Name string
	Env  string
	Port int
}

type DatabaseConfig struct {
	Host            string
	Port            int
	Service         string
	User            string
	Password        string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	IsAutoMigrate   bool // true: 테이블 재생성, false: 마이그레이션 비활성화
}

type JWTConfig struct {
	Secret        string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type ServerConfig struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	GracefulTimeout time.Duration
}

func Load(env string) (*Config, error) {
	if err := loadEnvFile(env); err != nil {
		return nil, fmt.Errorf("환경 변수 로드 실패: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "pray-together-api"),
			Env:  env,
			Port: getEnvAsInt("APP_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", ""),
			Port:            getEnvAsInt("DB_PORT", 1521),
			Service:         getEnv("DB_SERVICE", ""),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "1h"),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", "10m"),
			IsAutoMigrate:   getEnvAsBool("DB_AUTO_MIGRATE", false), // 기본값: false (안전)
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			Expiry:        getEnvAsDuration("JWT_EXPIRY", "24h"),
			RefreshExpiry: getEnvAsDuration("JWT_REFRESH_EXPIRY", "168h"),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
			AllowedMethods:   getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"*"}),
			AllowCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvAsInt("CORS_MAX_AGE", 86400),
		},
		Server: ServerConfig{
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", "15s"),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", "15s"),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", "60s"),
			GracefulTimeout: getEnvAsDuration("GRACEFUL_TIMEOUT", "30s"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("환경 변수 검증 실패 : %w", err)
	}

	return cfg, nil
}

func loadEnvFile(env string) error {
	envFile := fmt.Sprintf(".env.%s", env)

	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		slog.Warn("환경 변수 파일을 찾을 수 없습니다. 시스템 환경 변수를 사용합니다.",
			"file", envFile)
		return nil
	}

	if err := godotenv.Load(envFile); err != nil {
		return fmt.Errorf("환경 변수 파일 로드 오류: %s: %w", envFile, err)
	}

	absPath, _ := filepath.Abs(envFile)
	slog.Info("환경 변수 파일 로드", "file", absPath)
	return nil
}

func (c *Config) Validate() error {
	var errors []string

	// App validation
	if c.App.Port < 1 || c.App.Port > 65535 {
		errors = append(errors, "유효하지 않은 포트 번호")
	}

	// Database validation
	if c.Database.Host == "" {
		errors = append(errors, "데이터베이스 Host가 필요합니다")
	}
	if c.Database.Service == "" {
		errors = append(errors, "데이터베이스 Service가 필요합니다")
	}
	if c.Database.User == "" {
		errors = append(errors, "데이터베이스 User가 필요합니다")
	}
	if c.Database.Password == "" {
		errors = append(errors, "데이터베이스 Password가 필요합니다")
	}

	// JWT validation
	if c.JWT.Secret == "" {
		errors = append(errors, "JWT Secret Key가 필요합니다")
	}
	if len(c.JWT.Secret) < 32 {
		errors = append(errors, "JWT Secret Key는 32자 이상이어야 합니다")
	}

	if len(errors) > 0 {
		return fmt.Errorf("유효성 검사 오류: %s", strings.Join(errors, ", "))
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "local" || c.App.Env == "dev"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "prod"
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Service,
	)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return strings.Split(valueStr, ",")
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	if duration, err := time.ParseDuration(valueStr); err == nil {
		return duration
	}
	if defaultDuration, err := time.ParseDuration(defaultValue); err == nil {
		return defaultDuration
	}
	return 0
}
