// Package database provides database connection management for PostgreSQL.
package database

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config holds database connection configuration.
type Config struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	SSLMode  string
	TimeZone string
}

// getEnv reads an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// buildDSN constructs PostgreSQL DSN string from configuration.
func buildDSN(cfg Config) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)
}

// loadConfigFromEnv loads database configuration from environment variables.
func loadConfigFromEnv() Config {
	return Config{
		Host:     getEnv("DB_HOST", "localhost"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "avito_internship"),
		Port:     getEnv("DB_PORT", "5432"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		TimeZone: getEnv("DB_TIMEZONE", "UTC"),
	}
}

// New creates a new database connection using environment variables.
func New() (*gorm.DB, error) {
	cfg := loadConfigFromEnv()
	return NewWithConfig(cfg)
}

// sanitizeError removes sensitive information (password) from error messages.
func sanitizeError(err error, cfg Config) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	// Remove password from error message if present
	errMsg = strings.ReplaceAll(errMsg, cfg.Password, "***")
	// Also remove full DSN if it appears in error
	safeDSN := fmt.Sprintf("host=%s user=%s password=*** dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)
	dsn := buildDSN(cfg)
	errMsg = strings.ReplaceAll(errMsg, dsn, safeDSN)
	return fmt.Errorf("failed to connect to database: %s", errMsg)
}

// NewWithConfig creates a new database connection with custom configuration.
func NewWithConfig(cfg Config) (*gorm.DB, error) {
	dsn := buildDSN(cfg)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, sanitizeError(err, cfg)
	}
	return db, nil
}

// HealthCheck verifies database connection availability.
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}
