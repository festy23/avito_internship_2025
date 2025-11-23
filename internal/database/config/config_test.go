package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEnvVars saves original env vars and sets new ones for testing.
func setupEnvVars(t *testing.T, envVars map[string]string) map[string]string {
	t.Helper()
	originalEnv := make(map[string]string)
	for key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	return originalEnv
}

// restoreEnvVars restores original env vars after testing.
func restoreEnvVars(envVars map[string]string, originalEnv map[string]string) {
	for key := range envVars {
		os.Unsetenv(key)
	}
	for key, value := range originalEnv {
		if value != "" {
			os.Setenv(key, value)
		}
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		originalEnv := setupEnvVars(t, map[string]string{})
		defer restoreEnvVars(map[string]string{}, originalEnv)

		cfg := LoadConfigFromEnv()
		expected := Config{
			Host:     "localhost",
			User:     "postgres",
			Password: "postgres",
			DBName:   "avito_internship",
			Port:     "5432",
			SSLMode:  "disable",
			TimeZone: "UTC",
		}
		assert.Equal(t, expected, cfg)
	})

	t.Run("custom values", func(t *testing.T) {
		envVars := map[string]string{
			"DB_HOST":     "test-host",
			"DB_USER":     "test-user",
			"DB_PASSWORD": "test-password",
			"DB_NAME":     "test-db",
			"DB_PORT":     "5433",
			"DB_SSLMODE":  "require",
			"DB_TIMEZONE": "Europe/Moscow",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadConfigFromEnv()
		expected := Config{
			Host:     "test-host",
			User:     "test-user",
			Password: "test-password",
			DBName:   "test-db",
			Port:     "5433",
			SSLMode:  "require",
			TimeZone: "Europe/Moscow",
		}
		assert.Equal(t, expected, cfg)
	})

	t.Run("partial override", func(t *testing.T) {
		envVars := map[string]string{
			"DB_HOST": "custom-host",
			"DB_PORT": "9999",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadConfigFromEnv()
		expected := Config{
			Host:     "custom-host",
			User:     "postgres",
			Password: "postgres",
			DBName:   "avito_internship",
			Port:     "9999",
			SSLMode:  "disable",
			TimeZone: "UTC",
		}
		assert.Equal(t, expected, cfg)
	})
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
			dsn := BuildDSN(tt.config)
			assert.Equal(t, tt.expected, dsn)
		})
	}
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

			result := GetEnv(tt.envKey, tt.defaultValue)
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
			err: fmt.Errorf(
				"connection failed: host=localhost user=test password=secret123 dbname=test",
			),
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
			err: fmt.Errorf(
				"failed to connect to `host=localhost user=admin password=mypass " +
					"dbname=prod port=5432 sslmode=require TimeZone=UTC`"),
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
			result := SanitizeError(tt.err, tt.cfg)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			errMsg := result.Error()

			for _, shouldContain := range tt.shouldContain {
				assert.Contains(
					t,
					errMsg,
					shouldContain,
					"error message should contain: %s",
					shouldContain,
				)
			}

			for _, shouldNotContain := range tt.shouldNotContain {
				assert.NotContains(
					t,
					errMsg,
					shouldNotContain,
					"error message should not contain sensitive data: %s",
					shouldNotContain,
				)
			}
		})
	}
}
