package testutil

import (
	"testing"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
// This can be reused across all integration tests
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Silent mode for tests
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Auto-migrate all models
	err = db.AutoMigrate(
		&model.Member{},
		// Add other models here as needed
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// CleanupTestDB cleans up the test database
func CleanupTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Errorf("Failed to get database instance: %v", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
}

// TruncateTable truncates a table for test isolation
func TruncateTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()

	if err := db.Exec("DELETE FROM " + tableName).Error; err != nil {
		t.Fatalf("Failed to truncate table %s: %v", tableName, err)
	}
}
