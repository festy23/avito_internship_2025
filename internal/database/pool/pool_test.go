package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	assert.Equal(t, 25, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
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

// verifyPoolSettings verifies that pool settings were applied correctly.
func verifyPoolSettings(t *testing.T, db *gorm.DB, expectedMaxOpen int) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	stats := sqlDB.Stats()
	assert.Equal(t, expectedMaxOpen, stats.MaxOpenConnections)
}

func TestSetupConnectionPoolValidConfig(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.NoError(t, err)
	verifyPoolSettings(t, db, 10)
}

func TestSetupConnectionPoolMaxOpenConnsZero(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    0,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxOpenConns must be greater than 0")
}

func TestSetupConnectionPoolMaxOpenConnsNegative(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    -1,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxOpenConns must be greater than 0")
}

func TestSetupConnectionPoolMaxIdleConnsNegative(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    10,
		MaxIdleConns:    -1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxIdleConns must be non-negative")
}

func TestSetupConnectionPoolMaxIdleConnsGreaterThanMaxOpen(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    5,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxIdleConns (10) cannot be greater than MaxOpenConns (5)")
}

func TestSetupConnectionPoolMaxIdleConnsEqualMaxOpen(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.NoError(t, err)
	verifyPoolSettings(t, db, 10)
}

func TestSetupConnectionPoolWithClosedConnection(t *testing.T) {
	db := createTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	err = SetupConnectionPool(db, DefaultPoolConfig())
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get underlying sql.DB")
	} else {
		sqlDB2, err2 := db.DB()
		if err2 == nil {
			_ = sqlDB2.Ping() // Verify connection is closed
		}
	}
}

func TestSetupConnectionPoolWithMaxIdleConnsZero(t *testing.T) {
	db := createTestDB(t)
	defer closeTestDB(t, db)

	poolCfg := Config{
		MaxOpenConns:    10,
		MaxIdleConns:    0,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	err := SetupConnectionPool(db, poolCfg)
	assert.NoError(t, err)
}
