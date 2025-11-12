package database

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"

	oracle "github.com/godoes/gorm-oracle"
	"gorm.io/gorm"
)

// DB wraps the GORM database instance
type DB struct {
	*gorm.DB
}

// New creates a new database connection
func New(cfg *config.Config) (*DB, error) {
	dsn := buildDSN(cfg.Database)

	gormConfig := &gorm.Config{
		Logger:                 newLogger(cfg),
		PrepareStmt:            true, // Prepared statements for better performance
		SkipDefaultTransaction: true, // Skip default transaction for better performance, pass tx 1.BEGIN 2.INSERT(QUERY) 3.COMMIT (3 network)
		NowFunc: func() time.Time {
			return time.Now().UTC() // created_at, updated_at 등에 UTC 사용
		},
	}

	db, err := gorm.Open(oracle.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 실패: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 인스턴스 가져오기 실패: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("데이터베이스 핑 실패: %w", err)
	}

	slog.Info("데이터베이스 연결 성공",
		"host", cfg.Database.Host,
		"service", cfg.Database.Service,
		"max_idle_conns", cfg.Database.MaxIdleConns,
		"max_open_conns", cfg.Database.MaxOpenConns,
		"conn_max_lifetime", cfg.Database.ConnMaxLifetime.String(),
		"conn_max_idle_time", cfg.Database.ConnMaxIdleTime.String(),
	)

	// Run migration based on configuration
	if err := Migrate(db, cfg); err != nil {
		return nil, fmt.Errorf("마이그레이션 실패: %w", err)
	}

	return &DB{DB: db}, nil
}

// buildDSN constructs the Oracle connection string
func buildDSN(cfg config.DatabaseConfig) string {
	// ORACLE_CLOUD_MINIMAL_SETUP.md 참고
	// 패스워드 URL 인코딩 필수 (특수문자 처리)
	encodedPassword := url.QueryEscape(cfg.Password)

	// Oracle Cloud ATP는 기본적으로 SSL=true 필요
	// Format: oracle://user:password@host:port/service?SSL=true
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s?SSL=true",
		cfg.User,
		encodedPassword,
		cfg.Host,
		cfg.Port,
		cfg.Service,
	)

	return dsn
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("데이터베이스 종료 실패: %w", err)
	}

	slog.Info("데이터베이스 연결이 종료되었습니다")
	return nil
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("데이터베이스 인스턴스 가져오기 실패: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("데이터베이스 상태 확인 실패: %w", err)
	}

	return nil
}

// WithContext returns a new DB with context
func (db *DB) WithContext(ctx context.Context) *gorm.DB {
	return db.DB.WithContext(ctx)
}
