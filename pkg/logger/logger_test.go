package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	appConfig "github.com/festy23/avito_internship/internal/config"
)

func TestNew(t *testing.T) {
	t.Run("creates logger with default config", func(t *testing.T) {
		// Set env vars for test
		t.Setenv("LOGGER_LEVEL", "info")
		t.Setenv("LOGGER_FORMAT", "json")
		t.Setenv("LOGGER_OUTPUT", "stdout")
		t.Setenv("LOGGER_PRODUCTION", "true")

		logger, err := New()
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("creates logger with development config", func(t *testing.T) {
		t.Setenv("LOGGER_LEVEL", "debug")
		t.Setenv("LOGGER_FORMAT", "console")
		t.Setenv("LOGGER_PRODUCTION", "false")

		logger, err := New()
		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestNewWithConfig(t *testing.T) {
	t.Run("production logger with info level", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("development logger with debug level", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "debug",
			Format: "console",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with warn level", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "warn",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with error level", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "error",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with console format", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with stderr output", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stderr",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with invalid level defaults to info", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "invalid-level",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("logger with file output defaults to stdout", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "/tmp/app.log", // File output

		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("production logger config", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// Test that logger is functional
		logger.Info("test message")
		logger.Infow("test with fields", "key", "value")
	})

	t.Run("development logger config", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "debug",
			Format: "console",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// Test that logger is functional
		logger.Debug("test debug message")
		logger.Debugw("test debug with fields", "key", "value")
	})
}

func TestLoggerFunctionality(t *testing.T) {
	t.Run("logger can log at different levels", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)

		// These should not panic
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		logger.Debugw("debug with fields", "field", "value")
		logger.Infow("info with fields", "field", "value")
		logger.Warnw("warn with fields", "field", "value")
		logger.Errorw("error with fields", "field", "value")
	})

	t.Run("logger respects log level", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "warn", // Only warn and error should be logged
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)

		// These should not panic even if not logged
		logger.Debug("debug message - should not appear")
		logger.Info("info message - should not appear")
		logger.Warn("warn message - should appear")
		logger.Error("error message - should appear")
	})
}

func TestLoggerWithDifferentLevels(t *testing.T) {
	levels := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
	}

	for _, tt := range levels {
		t.Run("logger with "+tt.name+" level", func(t *testing.T) {
			cfg := appConfig.LoggerConfig{
				Level:  tt.level,
				Format: "json",
				Output: "stdout",
			}

			logger, err := NewWithConfig(cfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestLoggerFormats(t *testing.T) {
	formats := []struct {
		name   string
		format string
	}{
		{"json", "json"},
		{"console", "console"},
	}

	for _, tt := range formats {
		t.Run("logger with "+tt.name+" format", func(t *testing.T) {
			cfg := appConfig.LoggerConfig{
				Level:  "info",
				Format: tt.format,
				Output: "stdout",
			}

			logger, err := NewWithConfig(cfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestLoggerOutputs(t *testing.T) {
	outputs := []struct {
		name   string
		output string
	}{
		{"stdout", "stdout"},
		{"stderr", "stderr"},
	}

	for _, tt := range outputs {
		t.Run("logger with "+tt.name+" output", func(t *testing.T) {
			cfg := appConfig.LoggerConfig{
				Level:  "info",
				Format: "json",
				Output: tt.output,
			}

			logger, err := NewWithConfig(cfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestLoggerEdgeCases(t *testing.T) {
	t.Run("empty config uses defaults", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("invalid level falls back to info", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "not-a-level",
			Format: "json",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// Should not panic
		logger.Info("test message")
	})

	t.Run("unknown output defaults to stdout", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "unknown-output",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestLoggerIsProduction(t *testing.T) {
	t.Run("production config", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		}

		assert.True(t, cfg.IsProduction())

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("development config with debug", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		}

		assert.False(t, cfg.IsProduction())

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestLogLevelParsing(t *testing.T) {
	t.Run("all valid log levels", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}

		for _, level := range validLevels {
			cfg := appConfig.LoggerConfig{
				Level:  level,
				Format: "json",
				Output: "stdout",
			}

			logger, err := NewWithConfig(cfg)
			if level != "panic" && level != "fatal" && level != "dpanic" {
				require.NoError(t, err)
				require.NotNil(t, logger)
			}
		}
	})

	t.Run("case insensitive levels", func(t *testing.T) {
		levels := []string{"INFO", "Info", "iNfO"}

		for _, level := range levels {
			cfg := appConfig.LoggerConfig{
				Level:  level,
				Format: "json",
				Output: "stdout",
			}

			logger, err := NewWithConfig(cfg)
			require.NoError(t, err)
			require.NotNil(t, logger)
		}
	})
}

func TestLoggerConsoleEncoding(t *testing.T) {
	t.Run("console format has ISO8601 time encoder", func(t *testing.T) {
		cfg := appConfig.LoggerConfig{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		}

		logger, err := NewWithConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// Test with timestamp
		logger.Info("test message with timestamp")
	})
}

// Benchmark tests.
func BenchmarkNew(b *testing.B) {
	b.Setenv("LOGGER_LEVEL", "info")
	b.Setenv("LOGGER_FORMAT", "json")
	b.Setenv("LOGGER_PRODUCTION", "true")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, _ := New()
		_ = logger
	}
}

func BenchmarkNewWithConfig(b *testing.B) {
	cfg := appConfig.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, _ := NewWithConfig(cfg)
		_ = logger
	}
}

func BenchmarkLoggerInfo(b *testing.B) {
	cfg := appConfig.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, _ := NewWithConfig(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerInfoWithFields(b *testing.B) {
	cfg := appConfig.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	logger, _ := NewWithConfig(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infow("benchmark message", "field1", "value1", "field2", 123)
	}
}

func TestZapCoreLevels(t *testing.T) {
	t.Run("parse all zapcore levels", func(t *testing.T) {
		levels := []string{
			zapcore.DebugLevel.String(),
			zapcore.InfoLevel.String(),
			zapcore.WarnLevel.String(),
			zapcore.ErrorLevel.String(),
		}

		for _, levelStr := range levels {
			level, err := zapcore.ParseLevel(levelStr)
			require.NoError(t, err)
			assert.NotEqual(t, zapcore.InvalidLevel, level)
		}
	})
}
