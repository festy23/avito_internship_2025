// Package migrate provides database migration management.
package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/database/config"
)

// GetMigrationsPath returns the default path to migrations directory.
func GetMigrationsPath() string {
	return config.GetEnv("MIGRATIONS_PATH", "migrations")
}

// Migrate applies database migrations from the migrations directory using golang-migrate.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Get migrations directory path
	migrationsDir := GetMigrationsPath()

	// Convert relative path to absolute if needed
	migrationsPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations: %w", err)
	}

	// Check if migrations directory exists
	if _, statErr := os.Stat(migrationsPath); os.IsNotExist(statErr) {
		return fmt.Errorf("migrations directory does not exist: %s", migrationsPath)
	}

	// Create postgres driver instance
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrate instance with file source
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Apply all pending migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
