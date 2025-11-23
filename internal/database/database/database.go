// Package database provides database connection management for PostgreSQL.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/database/config"
	"github.com/festy23/avito_internship/internal/database/pool"
	"github.com/festy23/avito_internship/pkg/retry"
)

// New creates a new database connection using environment variables.
func New() (*gorm.DB, error) {
	cfg := config.LoadConfigFromEnv()
	return NewWithConfig(cfg)
}

// NewWithConfig creates a new database connection with custom configuration.
func NewWithConfig(cfg config.Config) (*gorm.DB, error) {
	retryCfg := config.LoadRetryConfigFromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dsn := config.BuildDSN(cfg)
	var db *gorm.DB
	var err error

	db, err = retry.DoWithResult(ctx, retryCfg, func() (*gorm.DB, error) {
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	})

	if err != nil {
		return nil, config.SanitizeError(err, cfg)
	}

	// Setup default connection pool
	if err := pool.SetupConnectionPool(db, pool.DefaultPoolConfig()); err != nil {
		return nil, fmt.Errorf("failed to setup connection pool: %w", err)
	}

	return db, nil
}

// HealthCheck verifies database connection availability.
func HealthCheck(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// Close gracefully closes database connection.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}

// GetStats returns database connection pool statistics.
func GetStats(db *gorm.DB) (*sql.DBStats, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	stats := sqlDB.Stats()
	return &stats, nil
}
