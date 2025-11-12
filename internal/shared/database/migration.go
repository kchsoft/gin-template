package database

import (
	"fmt"
	"log/slog"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/config"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/model"

	"gorm.io/gorm"
)

// Migrate executes database migration based on configuration
func Migrate(db *gorm.DB, cfg *config.Config) error {
	if !cfg.Database.IsAutoMigrate {
		slog.Info("⏭️  데이터베이스 마이그레이션 비활성화됨",
			"auto_migrate", false, "env", cfg.App.Env,
		)
		return nil
	}

	slog.Warn("🔧 데이터베이스 마이그레이션 시작 - 모든 테이블이 삭제되고 재생성됩니다!",
		"auto_migrate", true, "env", cfg.App.Env,
	)

	// Safety check: prevent accidental data loss in production
	if cfg.App.Env == "prod" || cfg.App.Env == "production" {
		return fmt.Errorf("🚨 PRODUCTION 환경에서는 DB_AUTO_MIGRATE=true를 사용할 수 없습니다! 데이터 손실 방지를 위해 차단됨")
	}

	// Step 1: Drop all tables (Oracle)
	slog.Info("🗑️  기존 테이블 삭제 중...")

	// Order matters: drop in reverse dependency order (FK constraints)
	tableNames := []string{"member"}

	for _, tableName := range tableNames {
		// Check if table exists (Oracle)
		var count int64
		db.Raw("SELECT COUNT(*) FROM USER_TABLES WHERE UPPER(TABLE_NAME) = UPPER(?)", tableName).Scan(&count)

		if count > 0 {
			// Oracle: DROP TABLE with CASCADE CONSTRAINTS
			dropSQL := fmt.Sprintf("DROP TABLE %s CASCADE CONSTRAINTS", tableName)
			if err := db.Exec(dropSQL).Error; err != nil {
				slog.Debug("테이블 삭제 실패", "table", tableName, "error", err)
			} else {
				slog.Debug("테이블 삭제 성공", "table", tableName)
			}
		}
	}

	// Step 2: Create tables with IDENTITY columns
	slog.Info("📦 새 테이블 생성 중...")
	if err := runAutoMigrate(db); err != nil {
		return fmt.Errorf("테이블 생성 실패: %w", err)
	}

	slog.Info("✅ 마이그레이션 완료!")
	return nil
}

// runAutoMigrate creates tables based on model definitions
func runAutoMigrate(db *gorm.DB) error {
	// 중요: 의존성 순서대로 생성 (FK 참조 순서)
	// 1. 독립 테이블 먼저
	// 2. FK 참조하는 테이블은 나중에
	models := []interface{}{
		// Independent tables (no foreign keys)
		&model.Member{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("%T 마이그레이션 실패: %w", m, err)
		}
		slog.Debug("테이블 생성됨", "model", fmt.Sprintf("%T", m))
	}

	return nil
}
