// Package pool provides database connection pool configuration.
package pool

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Config holds database connection pool configuration.
type Config struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultPoolConfig returns default connection pool configuration.
func DefaultPoolConfig() Config {
	return Config{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// SetupConnectionPool configures database connection pool settings.
func SetupConnectionPool(db *gorm.DB, poolCfg Config) error {
	if poolCfg.MaxOpenConns <= 0 {
		return fmt.Errorf("MaxOpenConns must be greater than 0")
	}
	if poolCfg.MaxIdleConns < 0 {
		return fmt.Errorf("MaxIdleConns must be non-negative")
	}
	if poolCfg.MaxIdleConns > poolCfg.MaxOpenConns {
		return fmt.Errorf(
			"MaxIdleConns (%d) cannot be greater than MaxOpenConns (%d)",
			poolCfg.MaxIdleConns, poolCfg.MaxOpenConns)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(poolCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolCfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(poolCfg.ConnMaxIdleTime)

	return nil
}
