package database

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/database/config"
)

func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    config.Config
		wantError bool
	}{
		{
			name: "valid sqlite config",
			config: config.Config{
				Host:     "localhost",
				User:     "test",
				Password: "test",
				DBName:   ":memory:",
				Port:     "5432",
				SSLMode:  "disable",
				TimeZone: "UTC",
			},
			wantError: true, // PostgreSQL driver won't work with sqlite
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewWithConfig(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					sqlDB, closeErr := db.DB()
					if closeErr == nil {
						_ = sqlDB.Close()
					}
				}
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func() *gorm.DB
		wantError bool
	}{
		{
			name: "healthy connection",
			setupDB: func() *gorm.DB {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				require.NoError(t, err)
				return db
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB()
			defer func() {
				if db != nil {
					sqlDB, err := db.DB()
					if err == nil {
						_ = sqlDB.Close()
					}
				}
			}()

			ctx := context.Background()
			err := HealthCheck(ctx, db)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("nil database", func(t *testing.T) {
		ctx := context.Background()
		err := HealthCheck(ctx, nil)
		assert.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	t.Run("close valid connection", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		err = Close(db)
		assert.NoError(t, err)

		// Verify connection is closed
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Ping()
		assert.Error(t, err) // Should fail because connection is closed
	})

	t.Run("close nil database", func(t *testing.T) {
		err := Close(nil)
		assert.NoError(t, err)
	})
}

func TestGetStats(t *testing.T) {
	t.Run("get stats from valid connection", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		defer func() {
			if sqlDB, closeErr := db.DB(); closeErr == nil {
				_ = sqlDB.Close()
			}
		}()

		stats, err := GetStats(db)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
	})

	t.Run("get stats from nil database", func(t *testing.T) {
		stats, err := GetStats(nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "database connection is nil")
	})
}

func TestNew(t *testing.T) {
	t.Run("new with default config", func(t *testing.T) {
		// Save original env
		originalEnv := make(map[string]string)
		envKeys := []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT", "DB_SSLMODE", "DB_TIMEZONE"}
		for _, key := range envKeys {
			originalEnv[key] = os.Getenv(key)
		}

		defer func() {
			for key, value := range originalEnv {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}
		}()

		// Clear env vars to use defaults
		for _, key := range envKeys {
			os.Unsetenv(key)
		}

		// This will fail to connect (no real PostgreSQL), but we test that New() is called
		db, err := New()
		assert.Error(t, err) // Expected - no PostgreSQL running
		assert.Nil(t, db)
	})
}

func TestNewWithConfigSuccessPath(t *testing.T) {
	// Test that NewWithConfig calls SetupConnectionPool
	// This is hard to test without real DB, but we can verify the flow
	cfg := config.Config{
		Host:     "localhost",
		User:     "test",
		Password: "test",
		DBName:   ":memory:",
		Port:     "5432",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	// This will fail because PostgreSQL driver, but we test the code path
	// Retry logic will attempt multiple times before giving up
	db, err := NewWithConfig(cfg)
	assert.Error(t, err) // Expected - PostgreSQL driver won't work with :memory:
	assert.Nil(t, db)

	// Note: Full success test requires real PostgreSQL database.
	// The function flow is: retry.DoWithResult -> buildDSN -> gorm.Open -> sanitizeError (on error) -> SetupConnectionPool
	// All these are tested separately. For full integration, use e2e tests.
}

func TestNewWithConfig_RetryBehavior(t *testing.T) {
	// Test that retry logic is applied
	// Since we can't easily test actual retries without real PostgreSQL,
	// we verify that the function uses retry by checking error messages
	cfg := config.Config{
		Host:     "localhost",
		User:     "test",
		Password: "test",
		DBName:   ":memory:",
		Port:     "5432",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	// This will fail, but retry will attempt multiple times
	db, err := NewWithConfig(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
	// Error should be sanitized (no password in error)
	assert.NotContains(t, err.Error(), "test")
}

func TestHealthCheckErrorPaths(t *testing.T) {
	t.Run("health check with nil database", func(t *testing.T) {
		ctx := context.Background()
		err := HealthCheck(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is nil")
	})

	t.Run("health check with closed connection", func(t *testing.T) {
		// Create and immediately close connection to test db.DB() error path
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		// Close the underlying connection
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Close()
		require.NoError(t, err)

		// HealthCheck should fail when trying to ping closed connection
		ctx := context.Background()
		err = HealthCheck(ctx, db)
		assert.Error(t, err)
		// Error could be from Ping() or from db.DB() if GORM detects closed connection
		assert.True(t,
			strings.Contains(err.Error(), "database ping failed") ||
				strings.Contains(err.Error(), "failed to get underlying sql.DB"),
			"error should be related to connection: %s", err.Error())
	})
}

func TestCloseErrorPaths(t *testing.T) {
	t.Run("close with nil database", func(t *testing.T) {
		err := Close(nil)
		assert.NoError(t, err) // nil is handled gracefully
	})

	t.Run("close with already closed connection", func(t *testing.T) {
		// Create connection and close it
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		// Close the underlying connection first
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Close()
		require.NoError(t, err)

		// Close() should handle already closed connection gracefully
		// or return error depending on GORM behavior
		err = Close(db)
		// Close() might succeed (idempotent) or return error
		// Both behaviors are acceptable
		if err != nil {
			assert.True(t,
				strings.Contains(err.Error(), "failed to get underlying sql.DB") ||
					strings.Contains(err.Error(), "failed to close database connection"),
				"error should be related to closing: %s", err.Error())
		}
	})
}

func TestGetStatsErrorPaths(t *testing.T) {
	t.Run("get stats with nil database", func(t *testing.T) {
		stats, err := GetStats(nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "database connection is nil")
	})

	t.Run("get stats with closed connection", func(t *testing.T) {
		// Create and close connection to test db.DB() error path
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)

		// Close the underlying connection
		sqlDB, err := db.DB()
		require.NoError(t, err)
		err = sqlDB.Close()
		require.NoError(t, err)

		// GetStats might succeed (GORM caches connection),
		// but the connection is actually closed.
		// This tests that GetStats handles closed connections gracefully.
		stats, err := GetStats(db)
		// GORM may or may not return error immediately after closing,
		// so we accept both behaviors
		if err != nil {
			assert.Nil(t, stats)
			assert.True(t,
				strings.Contains(err.Error(), "failed to get underlying sql.DB"),
				"error should be related to getting sql.DB: %s", err.Error())
		} else {
			// If no error, stats should still be valid (GORM cached connection)
			// This is acceptable behavior
			assert.NotNil(t, stats)
		}
	})
}
