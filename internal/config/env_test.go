package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("existing env var", func(t *testing.T) {
		os.Setenv("TEST_KEY", "test_value")
		defer os.Unsetenv("TEST_KEY")

		result := GetEnv("TEST_KEY", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("missing env var", func(t *testing.T) {
		os.Unsetenv("TEST_KEY_MISSING")

		result := GetEnv("TEST_KEY_MISSING", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("empty env var", func(t *testing.T) {
		os.Setenv("TEST_KEY_EMPTY", "")
		defer os.Unsetenv("TEST_KEY_EMPTY")

		result := GetEnv("TEST_KEY_EMPTY", "default")
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("valid integer", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		result := GetEnvInt("TEST_INT", 0)
		assert.Equal(t, 42, result)
	})

	t.Run("invalid integer", func(t *testing.T) {
		os.Setenv("TEST_INT_INVALID", "not_a_number")
		defer os.Unsetenv("TEST_INT_INVALID")

		result := GetEnvInt("TEST_INT_INVALID", 10)
		assert.Equal(t, 10, result)
	})

	t.Run("missing env var", func(t *testing.T) {
		os.Unsetenv("TEST_INT_MISSING")

		result := GetEnvInt("TEST_INT_MISSING", 5)
		assert.Equal(t, 5, result)
	})

	t.Run("negative integer", func(t *testing.T) {
		os.Setenv("TEST_INT_NEG", "-10")
		defer os.Unsetenv("TEST_INT_NEG")

		result := GetEnvInt("TEST_INT_NEG", 0)
		assert.Equal(t, -10, result)
	})
}

func TestGetEnvDuration(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		os.Setenv("TEST_DURATION", "30s")
		defer os.Unsetenv("TEST_DURATION")

		result := GetEnvDuration("TEST_DURATION", 10*time.Second)
		assert.Equal(t, 30*time.Second, result)
	})

	t.Run("invalid duration", func(t *testing.T) {
		os.Setenv("TEST_DURATION_INVALID", "invalid")
		defer os.Unsetenv("TEST_DURATION_INVALID")

		result := GetEnvDuration("TEST_DURATION_INVALID", 5*time.Second)
		assert.Equal(t, 5*time.Second, result)
	})

	t.Run("missing env var", func(t *testing.T) {
		os.Unsetenv("TEST_DURATION_MISSING")

		result := GetEnvDuration("TEST_DURATION_MISSING", 1*time.Minute)
		assert.Equal(t, 1*time.Minute, result)
	})

	t.Run("complex duration", func(t *testing.T) {
		os.Setenv("TEST_DURATION_COMPLEX", "1h30m15s")
		defer os.Unsetenv("TEST_DURATION_COMPLEX")

		result := GetEnvDuration("TEST_DURATION_COMPLEX", time.Second)
		expected := 1*time.Hour + 30*time.Minute + 15*time.Second
		assert.Equal(t, expected, result)
	})
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true value",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false value",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "1 as true",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "0 as false",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "invalid value",
			envValue:     "invalid",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "missing env var",
			envValue:     "",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL"
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)
			} else {
				os.Unsetenv(key)
			}

			result := GetEnvBool(key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
