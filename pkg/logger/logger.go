// Package logger provides structured logging using zap.
package logger

import (
	"go.uber.org/zap"
)

// New creates a new production logger with JSON encoding.
func New() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}
