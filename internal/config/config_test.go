package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupAndRestoreEnv saves original env vars and sets new ones for testing.
func setupAndRestoreEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()
	originalEnv := make(map[string]string)
	for key := range envVars {
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

func TestLoadFromEnv_DefaultValues(t *testing.T) {
	restore := setupAndRestoreEnv(t, map[string]string{})
	defer restore()

	cfg := LoadFromEnv()
	assert.Equal(t, ":8080", cfg.Server.Port)
	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "release", cfg.GinMode)
}

func TestLoadFromEnv_CustomValues(t *testing.T) {
	restore := setupAndRestoreEnv(t, map[string]string{
		"SERVER_PORT": ":9090",
		"LOG_LEVEL":   "debug",
		"GIN_MODE":    "debug",
	})
	defer restore()

	cfg := LoadFromEnv()
	assert.Equal(t, ":9090", cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "debug", cfg.GinMode)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			Logger: LoggerConfig{
				Level:  "info",
				Format: "json",
			},
			GinMode: "release",
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid server config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				ReadTimeout:  0,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			Logger: LoggerConfig{
				Level:  "info",
				Format: "json",
			},
			GinMode: "release",
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server config validation failed")
	})

	t.Run("invalid logger config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			Logger: LoggerConfig{
				Level:  "invalid",
				Format: "json",
			},
			GinMode: "release",
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logger config validation failed")
	})

	t.Run("invalid gin mode", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			Logger: LoggerConfig{
				Level:  "info",
				Format: "json",
			},
			GinMode: "invalid",
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid GIN_MODE")
	})

	t.Run("valid gin modes", func(t *testing.T) {
		validModes := []string{"debug", "release", "test"}
		for _, mode := range validModes {
			cfg := Config{
				Server: ServerConfig{
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 10 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				Logger: LoggerConfig{
					Level:  "info",
					Format: "json",
				},
				GinMode: mode,
			}
			err := cfg.Validate()
			assert.NoError(t, err, "mode %s should be valid", mode)
		}
	})
}
