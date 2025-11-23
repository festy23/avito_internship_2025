package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expectedConfig: Config{
				Host:     "localhost",
				User:     "postgres",
				Password: "postgres",
				DBName:   "avito_internship",
				Port:     "5432",
				SSLMode:  "disable",
				TimeZone: "UTC",
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"DB_HOST":     "test-host",
				"DB_USER":     "test-user",
				"DB_PASSWORD": "test-password",
				"DB_NAME":     "test-db",
				"DB_PORT":     "5433",
				"DB_SSLMODE":  "require",
				"DB_TIMEZONE": "Europe/Moscow",
			},
			expectedConfig: Config{
				Host:     "test-host",
				User:     "test-user",
				Password: "test-password",
				DBName:   "test-db",
				Port:     "5433",
				SSLMode:  "require",
				TimeZone: "Europe/Moscow",
			},
		},
		{
			name: "partial override",
			envVars: map[string]string{
				"DB_HOST": "custom-host",
				"DB_PORT": "9999",
			},
			expectedConfig: Config{
				Host:     "custom-host",
				User:     "postgres",
				Password: "postgres",
				DBName:   "avito_internship",
				Port:     "9999",
				SSLMode:  "disable",
				TimeZone: "UTC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Clean up env vars
			for key := range tt.envVars {
				os.Unsetenv(key)
			}

			// Set test env vars
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
				// Restore original values
				for key, value := range originalEnv {
					if value != "" {
						os.Setenv(key, value)
					}
				}
			}()

			cfg := loadConfigFromEnv()
			assert.Equal(t, tt.expectedConfig, cfg)
		})
	}
}

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "standard config",
			config: Config{
				Host:     "localhost",
				User:     "postgres",
				Password: "postgres",
				DBName:   "avito_internship",
				Port:     "5432",
				SSLMode:  "disable",
				TimeZone: "UTC",
			},
			expected: "host=localhost user=postgres password=postgres dbname=avito_internship port=5432 sslmode=disable TimeZone=UTC",
		},
		{
			name: "custom config",
			config: Config{
				Host:     "db.example.com",
				User:     "admin",
				Password: "secret123",
				DBName:   "production",
				Port:     "5433",
				SSLMode:  "require",
				TimeZone: "Europe/Moscow",
			},
			expected: "host=db.example.com user=admin password=secret123 dbname=production port=5433 sslmode=require TimeZone=Europe/Moscow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := buildDSN(tt.config)
			assert.Equal(t, tt.expected, dsn)
		})
	}
}

func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid sqlite config",
			config: Config{
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

			err := HealthCheck(db)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("nil database", func(t *testing.T) {
		err := HealthCheck(nil)
		assert.Error(t, err)
	})
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "env var set",
			envKey:       "TEST_ENV_VAR",
			envValue:     "test-value",
			defaultValue: "default-value",
			expected:     "test-value",
		},
		{
			name:         "env var not set",
			envKey:       "TEST_ENV_VAR_NOT_SET",
			envValue:     "",
			defaultValue: "default-value",
			expected:     "default-value",
		},
		{
			name:         "env var empty string",
			envKey:       "TEST_ENV_VAR_EMPTY",
			envValue:     "",
			defaultValue: "default-value",
			expected:     "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalValue := os.Getenv(tt.envKey)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envKey, originalValue)
				} else {
					os.Unsetenv(tt.envKey)
				}
			}()

			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
			} else {
				os.Unsetenv(tt.envKey)
			}

			result := getEnv(tt.envKey, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		cfg              Config
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "password in error message",
			err:  fmt.Errorf("connection failed: host=localhost user=test password=secret123 dbname=test"),
			cfg: Config{
				Host:     "localhost",
				User:     "test",
				Password: "secret123",
				DBName:   "test",
				Port:     "5432",
				SSLMode:  "disable",
				TimeZone: "UTC",
			},
			shouldContain:    []string{"failed to connect to database", "password=***"},
			shouldNotContain: []string{"secret123", "password=secret123"},
		},
		{
			name: "full DSN in error message",
			err:  fmt.Errorf("failed to connect to `host=localhost user=admin password=mypass dbname=prod port=5432 sslmode=require TimeZone=UTC`"),
			cfg: Config{
				Host:     "localhost",
				User:     "admin",
				Password: "mypass",
				DBName:   "prod",
				Port:     "5432",
				SSLMode:  "require",
				TimeZone: "UTC",
			},
			shouldContain:    []string{"failed to connect to database", "password=***"},
			shouldNotContain: []string{"mypass", "password=mypass"},
		},
		{
			name: "nil error",
			err:  nil,
			cfg: Config{
				Password: "secret",
			},
			shouldContain:    []string{},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.err, tt.cfg)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			errMsg := result.Error()

			for _, shouldContain := range tt.shouldContain {
				assert.Contains(t, errMsg, shouldContain, "error message should contain: %s", shouldContain)
			}

			for _, shouldNotContain := range tt.shouldNotContain {
				assert.NotContains(t, errMsg, shouldNotContain, "error message should not contain sensitive data: %s", shouldNotContain)
			}
		})
	}
}
