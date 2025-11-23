package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupAndRestoreServerEnv saves original env vars and sets new ones for testing.
func setupAndRestoreServerEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()
	originalEnv := make(map[string]string)
	envKeys := []string{
		"SERVER_HOST",
		"SERVER_PORT",
		"SERVER_READ_TIMEOUT",
		"SERVER_WRITE_TIMEOUT",
		"SERVER_IDLE_TIMEOUT",
	}
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

func TestLoadServerConfigFromEnv_DefaultValues(t *testing.T) {
	restore := setupAndRestoreServerEnv(t, map[string]string{})
	defer restore()

	cfg := LoadServerConfigFromEnv()
	assert.Equal(t, "", cfg.Host)
	assert.Equal(t, ":8080", cfg.Port)
	assert.Equal(t, 10*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.IdleTimeout)
}

func TestLoadServerConfigFromEnv_CustomValues(t *testing.T) {
	restore := setupAndRestoreServerEnv(t, map[string]string{
		"SERVER_HOST":          "0.0.0.0",
		"SERVER_PORT":          "9090",
		"SERVER_READ_TIMEOUT":  "30s",
		"SERVER_WRITE_TIMEOUT": "30s",
		"SERVER_IDLE_TIMEOUT":  "300s",
	})
	defer restore()

	cfg := LoadServerConfigFromEnv()
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, 30*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 300*time.Second, cfg.IdleTimeout)
}

func TestServerConfig_GetAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected string
	}{
		{
			name: "port only with colon",
			config: ServerConfig{
				Host: "",
				Port: ":8080",
			},
			expected: ":8080",
		},
		{
			name: "port only without colon",
			config: ServerConfig{
				Host: "",
				Port: "8080",
			},
			expected: "8080",
		},
		{
			name: "host and port",
			config: ServerConfig{
				Host: "localhost",
				Port: "8080",
			},
			expected: "localhost:8080",
		},
		{
			name: "host and port with colon",
			config: ServerConfig{
				Host: "0.0.0.0",
				Port: ":8080",
			},
			expected: "0.0.0.0:8080",
		},
		{
			name: "empty host and empty port",
			config: ServerConfig{
				Host: "",
				Port: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetAddress()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := ServerConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid read timeout", func(t *testing.T) {
		cfg := ServerConfig{
			ReadTimeout:  0,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ReadTimeout")
	})

	t.Run("invalid write timeout", func(t *testing.T) {
		cfg := ServerConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: -1 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "WriteTimeout")
	})

	t.Run("invalid idle timeout", func(t *testing.T) {
		cfg := ServerConfig{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  0,
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "IdleTimeout")
	})
}
