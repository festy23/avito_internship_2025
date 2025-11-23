// Package logger provides structured logging using zap.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	appConfig "github.com/festy23/avito_internship/internal/config"
)

// New creates a new logger with default production configuration.
func New() (*zap.SugaredLogger, error) {
	cfg := appConfig.LoadLoggerConfigFromEnv()
	return NewWithConfig(cfg)
}

// NewWithConfig creates a new logger with custom configuration.
func NewWithConfig(cfg appConfig.LoggerConfig) (*zap.SugaredLogger, error) {
	var zapConfig zap.Config

	if cfg.IsProduction() {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set encoding
	if cfg.Format == "console" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig.Encoding = "json"
	}

	// Set output
	if cfg.Output != "stdout" && cfg.Output != "stderr" {
		// File output would require additional setup
		// For now, default to stdout
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	} else {
		zapConfig.OutputPaths = []string{cfg.Output}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
