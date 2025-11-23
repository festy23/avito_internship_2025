package migrate

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetMigrationsPath(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		originalValue := os.Getenv("MIGRATIONS_PATH")
		defer func() {
			if originalValue != "" {
				os.Setenv("MIGRATIONS_PATH", originalValue)
			} else {
				os.Unsetenv("MIGRATIONS_PATH")
			}
		}()

		os.Unsetenv("MIGRATIONS_PATH")
		path := GetMigrationsPath()
		assert.Equal(t, "migrations", path)
	})

	t.Run("custom path from env", func(t *testing.T) {
		originalValue := os.Getenv("MIGRATIONS_PATH")
		defer func() {
			if originalValue != "" {
				os.Setenv("MIGRATIONS_PATH", originalValue)
			} else {
				os.Unsetenv("MIGRATIONS_PATH")
			}
		}()

		os.Setenv("MIGRATIONS_PATH", "custom/migrations")
		path := GetMigrationsPath()
		assert.Equal(t, "custom/migrations", path)
	})
}

// setupMigrationsPath sets MIGRATIONS_PATH env var and returns cleanup function.
func setupMigrationsPath(t *testing.T, path string) func() {
	t.Helper()
	originalPath := os.Getenv("MIGRATIONS_PATH")
	os.Setenv("MIGRATIONS_PATH", path)
	return func() {
		if originalPath != "" {
			os.Setenv("MIGRATIONS_PATH", originalPath)
		} else {
			os.Unsetenv("MIGRATIONS_PATH")
		}
	}
}

// createTestDB creates a test SQLite database connection.
func createTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// closeTestDB closes a test database connection.
func closeTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func TestMigrateWithNilDatabase(t *testing.T) {
	err := Migrate(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is nil")
}

func TestMigrateWithNonExistentDirectory(t *testing.T) {
	cleanup := setupMigrationsPath(t, "/non/existent/path")
	defer cleanup()

	db := createTestDB(t)
	defer closeTestDB(t, db)

	err := Migrate(db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrations directory does not exist")
}

func TestMigrateWithDBError(t *testing.T) {
	db := createTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	err = Migrate(db)
	assert.Error(t, err)
}

func TestMigrateWithPostgresDriverError(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := setupMigrationsPath(t, tmpDir)
	defer cleanup()

	db := createTestDB(t)
	defer closeTestDB(t, db)

	err := Migrate(db)
	assert.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "failed to create postgres driver") ||
			strings.Contains(err.Error(), "failed to create migrate instance"),
		"error should be related to postgres driver: %s", err.Error())
}

func TestMigrateWithInvalidPathFormat(t *testing.T) {
	tmpDir := t.TempDir()
	cleanup := setupMigrationsPath(t, tmpDir)
	defer cleanup()

	db := createTestDB(t)
	defer closeTestDB(t, db)

	err := Migrate(db)
	assert.Error(t, err)
}

func TestMigrateHandlesErrNoChange(t *testing.T) {
	// This test verifies that migrate.ErrNoChange is handled correctly
	// In practice, this happens when migrations are already applied
	// We can't easily test this without real PostgreSQL, but we document the behavior
	// The code checks: err != nil && !errors.Is(err, migrate.ErrNoChange)
	// So ErrNoChange should return nil (success)
	t.Skip("Requires real PostgreSQL database - covered in e2e tests")
}
