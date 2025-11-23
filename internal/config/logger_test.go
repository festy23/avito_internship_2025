package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupAndRestoreLoggerEnv saves original env vars and sets new ones for testing.
func setupAndRestoreLoggerEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()
	originalEnv := make(map[string]string)
	envKeys := []string{"LOG_LEVEL", "LOG_FORMAT", "LOG_OUTPUT"}
	for _, key := range envKeys {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	return func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			}
		}
	}
}

func TestLoadLoggerConfigFromEnv_DefaultValues(t *testing.T) {
	restore := setupAndRestoreLoggerEnv(t, map[string]string{})
	defer restore()

	cfg := LoadLoggerConfigFromEnv()
	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
}

func TestLoadLoggerConfigFromEnv_CustomValues(t *testing.T) {
	restore := setupAndRestoreLoggerEnv(t, map[string]string{
		"LOG_LEVEL":  "debug",
		"LOG_FORMAT": "console",
		"LOG_OUTPUT": "stderr",
	})
	defer restore()

	cfg := LoadLoggerConfigFromEnv()
	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "console", cfg.Format)
	assert.Equal(t, "stderr", cfg.Output)
}

func TestLoggerConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    LoggerConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: LoggerConfig{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantError: false,
		},
		{
			name: "invalid level",
			config: LoggerConfig{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantError: true,
		},
		{
			name: "invalid format",
			config: LoggerConfig{
				Level:  "info",
				Format: "invalid",
				Output: "stdout",
			},
			wantError: true,
		},
		{
			name: "valid debug level",
			config: LoggerConfig{
				Level:  "debug",
				Format: "console",
				Output: "stdout",
			},
			wantError: false,
		},
		{
			name: "valid warn level",
			config: LoggerConfig{
				Level:  "warn",
				Format: "json",
				Output: "stderr",
			},
			wantError: false,
		},
		{
			name: "valid error level",
			config: LoggerConfig{
				Level:  "error",
				Format: "json",
				Output: "stdout",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoggerConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name     string
		config   LoggerConfig
		expected bool
	}{
		{
			name: "production config",
			config: LoggerConfig{
				Level:  "info",
				Format: "json",
			},
			expected: true,
		},
		{
			name: "debug level is not production",
			config: LoggerConfig{
				Level:  "debug",
				Format: "json",
			},
			expected: false,
		},
		{
			name: "console format is not production",
			config: LoggerConfig{
				Level:  "info",
				Format: "console",
			},
			expected: false,
		},
		{
			name: "warn level is production",
			config: LoggerConfig{
				Level:  "warn",
				Format: "json",
			},
			expected: true,
		},
		{
			name: "error level is production",
			config: LoggerConfig{
				Level:  "error",
				Format: "json",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsProduction()
			assert.Equal(t, tt.expected, result)
		})
	}
}
