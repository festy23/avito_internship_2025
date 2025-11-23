package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/festy23/avito_internship/pkg/retry"
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
		// Explicitly unset all DB_* env vars to test defaults
		envVarsToUnset := []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT", "DB_SSLMODE", "DB_TIMEZONE"}
		originalEnv := make(map[string]string)
		for _, key := range envVarsToUnset {
			originalEnv[key] = os.Getenv(key)
			os.Unsetenv(key)
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
		// Explicitly unset other DB_* env vars to test partial override
		envVarsToUnset := []string{"DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "DB_TIMEZONE"}
		originalEnv := make(map[string]string)
		for _, key := range envVarsToUnset {
			originalEnv[key] = os.Getenv(key)
			os.Unsetenv(key)
		}
		// Set only the vars we want to override
		os.Setenv("DB_HOST", "custom-host")
		os.Setenv("DB_PORT", "9999")
		defer func() {
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			for key, value := range originalEnv {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}
		}()

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

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "env var set with valid int",
			envKey:       "TEST_INT_VAR",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "env var not set",
			envKey:       "TEST_INT_VAR_NOT_SET",
			envValue:     "",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "env var with invalid int",
			envKey:       "TEST_INT_VAR_INVALID",
			envValue:     "not-a-number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "env var with negative int",
			envKey:       "TEST_INT_VAR_NEGATIVE",
			envValue:     "-5",
			defaultValue: 10,
			expected:     -5,
		},
		{
			name:         "env var with zero",
			envKey:       "TEST_INT_VAR_ZERO",
			envValue:     "0",
			defaultValue: 10,
			expected:     0,
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

			result := getEnvInt(tt.envKey, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "env var set with valid duration",
			envKey:       "TEST_DURATION_VAR",
			envValue:     "5s",
			defaultValue: 1 * time.Second,
			expected:     5 * time.Second,
		},
		{
			name:         "env var not set",
			envKey:       "TEST_DURATION_VAR_NOT_SET",
			envValue:     "",
			defaultValue: 1 * time.Second,
			expected:     1 * time.Second,
		},
		{
			name:         "env var with invalid duration",
			envKey:       "TEST_DURATION_VAR_INVALID",
			envValue:     "not-a-duration",
			defaultValue: 1 * time.Second,
			expected:     1 * time.Second,
		},
		{
			name:         "env var with minutes",
			envKey:       "TEST_DURATION_VAR_MINUTES",
			envValue:     "2m",
			defaultValue: 1 * time.Second,
			expected:     2 * time.Minute,
		},
		{
			name:         "env var with milliseconds",
			envKey:       "TEST_DURATION_VAR_MS",
			envValue:     "500ms",
			defaultValue: 1 * time.Second,
			expected:     500 * time.Millisecond,
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

			result := getEnvDuration(tt.envKey, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvFloat(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue float64
		expected     float64
	}{
		{
			name:         "env var set with valid float",
			envKey:       "TEST_FLOAT_VAR",
			envValue:     "3.14",
			defaultValue: 1.0,
			expected:     3.14,
		},
		{
			name:         "env var not set",
			envKey:       "TEST_FLOAT_VAR_NOT_SET",
			envValue:     "",
			defaultValue: 1.0,
			expected:     1.0,
		},
		{
			name:         "env var with invalid float",
			envKey:       "TEST_FLOAT_VAR_INVALID",
			envValue:     "not-a-float",
			defaultValue: 1.0,
			expected:     1.0,
		},
		{
			name:         "env var with negative float",
			envKey:       "TEST_FLOAT_VAR_NEGATIVE",
			envValue:     "-2.5",
			defaultValue: 1.0,
			expected:     -2.5,
		},
		{
			name:         "env var with zero",
			envKey:       "TEST_FLOAT_VAR_ZERO",
			envValue:     "0",
			defaultValue: 1.0,
			expected:     0.0,
		},
		{
			name:         "env var with scientific notation",
			envKey:       "TEST_FLOAT_VAR_SCIENTIFIC",
			envValue:     "1.5e2",
			defaultValue: 1.0,
			expected:     150.0,
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

			result := getEnvFloat(tt.envKey, tt.defaultValue)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestLoadRetryConfigFromEnv(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		envVars := map[string]string{
			"DB_RETRY_MAX_ATTEMPTS":  "",
			"DB_RETRY_INITIAL_DELAY": "",
			"DB_RETRY_MAX_DELAY":     "",
			"DB_RETRY_MULTIPLIER":    "",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadRetryConfigFromEnv()
		defaultCfg := retry.PostgresConfig()
		assert.Equal(t, defaultCfg.MaxAttempts, cfg.MaxAttempts)
		assert.Equal(t, defaultCfg.InitialDelay, cfg.InitialDelay)
		assert.Equal(t, defaultCfg.MaxDelay, cfg.MaxDelay)
		assert.Equal(t, defaultCfg.Multiplier, cfg.Multiplier)
	})

	t.Run("custom values from env", func(t *testing.T) {
		envVars := map[string]string{
			"DB_RETRY_MAX_ATTEMPTS":  "10",
			"DB_RETRY_INITIAL_DELAY": "2s",
			"DB_RETRY_MAX_DELAY":     "30s",
			"DB_RETRY_MULTIPLIER":    "2.5",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadRetryConfigFromEnv()
		assert.Equal(t, 10, cfg.MaxAttempts)
		assert.Equal(t, 2*time.Second, cfg.InitialDelay)
		assert.Equal(t, 30*time.Second, cfg.MaxDelay)
		assert.InDelta(t, 2.5, cfg.Multiplier, 0.0001)
	})

	t.Run("partial override", func(t *testing.T) {
		envVars := map[string]string{
			"DB_RETRY_MAX_ATTEMPTS":  "5",
			"DB_RETRY_INITIAL_DELAY": "",
			"DB_RETRY_MAX_DELAY":     "",
			"DB_RETRY_MULTIPLIER":    "",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadRetryConfigFromEnv()
		defaultCfg := retry.PostgresConfig()
		assert.Equal(t, 5, cfg.MaxAttempts)
		assert.Equal(t, defaultCfg.InitialDelay, cfg.InitialDelay)
		assert.Equal(t, defaultCfg.MaxDelay, cfg.MaxDelay)
		assert.Equal(t, defaultCfg.Multiplier, cfg.Multiplier)
	})

	t.Run("invalid values fallback to defaults", func(t *testing.T) {
		envVars := map[string]string{
			"DB_RETRY_MAX_ATTEMPTS":  "invalid",
			"DB_RETRY_INITIAL_DELAY": "invalid",
			"DB_RETRY_MAX_DELAY":     "invalid",
			"DB_RETRY_MULTIPLIER":    "invalid",
		}
		originalEnv := setupEnvVars(t, envVars)
		defer restoreEnvVars(envVars, originalEnv)

		cfg := LoadRetryConfigFromEnv()
		defaultCfg := retry.PostgresConfig()
		assert.Equal(t, defaultCfg.MaxAttempts, cfg.MaxAttempts)
		assert.Equal(t, defaultCfg.InitialDelay, cfg.InitialDelay)
		assert.Equal(t, defaultCfg.MaxDelay, cfg.MaxDelay)
		assert.Equal(t, defaultCfg.Multiplier, cfg.Multiplier)
	})
}
