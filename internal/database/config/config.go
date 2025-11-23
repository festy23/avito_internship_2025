// Package config provides database configuration management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/festy23/avito_internship/pkg/retry"
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

// GetEnv reads an environment variable with a default fallback.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// BuildDSN constructs PostgreSQL DSN string from configuration.
func BuildDSN(cfg Config) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)
}

// LoadConfigFromEnv loads database configuration from environment variables.
func LoadConfigFromEnv() Config {
	return Config{
		Host:     GetEnv("DB_HOST", "localhost"),
		User:     GetEnv("DB_USER", "postgres"),
		Password: GetEnv("DB_PASSWORD", "postgres"),
		DBName:   GetEnv("DB_NAME", "avito_internship"),
		Port:     GetEnv("DB_PORT", "5432"),
		SSLMode:  GetEnv("DB_SSLMODE", "disable"),
		TimeZone: GetEnv("DB_TIMEZONE", "UTC"),
	}
}

// SanitizeError removes sensitive information (password) from error messages.
func SanitizeError(err error, cfg Config) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	// Remove password from error message if present
	errMsg = strings.ReplaceAll(errMsg, cfg.Password, "***")
	// Also remove full DSN if it appears in error
	safeDSN := fmt.Sprintf("host=%s user=%s password=*** dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.Host, cfg.User, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)
	dsn := BuildDSN(cfg)
	errMsg = strings.ReplaceAll(errMsg, dsn, safeDSN)
	return fmt.Errorf("failed to connect to database: %s", errMsg)
}

// getEnvInt reads an integer environment variable with a default fallback.
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// getEnvDuration reads a duration environment variable with a default fallback.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// getEnvFloat reads a float environment variable with a default fallback.
func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

// LoadRetryConfigFromEnv loads retry configuration from environment variables.
func LoadRetryConfigFromEnv() retry.Config {
	cfg := retry.PostgresConfig()
	cfg.MaxAttempts = getEnvInt("DB_RETRY_MAX_ATTEMPTS", cfg.MaxAttempts)
	cfg.InitialDelay = getEnvDuration("DB_RETRY_INITIAL_DELAY", cfg.InitialDelay)
	cfg.MaxDelay = getEnvDuration("DB_RETRY_MAX_DELAY", cfg.MaxDelay)
	cfg.Multiplier = getEnvFloat("DB_RETRY_MULTIPLIER", cfg.Multiplier)
	return cfg
}
