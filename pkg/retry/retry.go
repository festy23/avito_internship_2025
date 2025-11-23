// Package retry provides retry logic with exponential backoff for operations.
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// Config holds retry strategy configuration.
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (including initial attempt).
	MaxAttempts int
	// InitialDelay is the initial delay before first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retry attempts.
	MaxDelay time.Duration
	// Multiplier is the exponential backoff multiplier.
	Multiplier float64
	// RetryableErrors is a list of error patterns to retry on.
	// If empty, all errors are considered retryable.
	RetryableErrors []string
}

// DefaultConfig returns default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:     5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		Multiplier:      2.0,
		RetryableErrors: []string{},
	}
}

// Do executes a function with retry logic.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	_, err := DoWithResult(ctx, cfg, func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

// DoWithResult executes a function with retry logic and returns the result.
func DoWithResult[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var zero T

	// Validate config
	if cfg.MaxAttempts <= 0 {
		return zero, fmt.Errorf("MaxAttempts must be greater than 0")
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before attempt
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Execute function
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err, cfg) {
			return zero, err
		}

		// Don't wait after last attempt
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Calculate delay
		delay := calculateDelay(attempt, cfg)
		delay = addJitter(delay)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return zero, lastErr
}

// calculateDelay calculates exponential backoff delay.
func calculateDelay(attempt int, cfg Config) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff: initialDelay * (multiplier ^ attempt)
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	// Cap at maxDelay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	return time.Duration(delay)
}

// addJitter adds random jitter to delay to avoid thundering herd.
func addJitter(delay time.Duration) time.Duration {
	// Add ±10% jitter
	jitterPercent := 0.1
	//nolint:gosec // math/rand is sufficient for jitter calculation, no security requirement
	jitter := float64(delay) * jitterPercent * (rand.Float64()*2 - 1)
	return delay + time.Duration(jitter)
}

// IsRetryableError checks if error should trigger a retry.
func IsRetryableError(err error, cfg Config) bool {
	if err == nil {
		return false
	}

	// If no retryable errors specified, all errors are retryable
	if len(cfg.RetryableErrors) == 0 {
		return true
	}

	errMsg := strings.ToLower(err.Error())

	// Check if error message matches any retryable pattern
	for _, pattern := range cfg.RetryableErrors {
		if strings.Contains(errMsg, strings.ToLower(pattern)) {
			return true
		}
	}

	// Error doesn't match any retryable pattern
	return false
}

// DefaultPostgresRetryableErrors returns default retryable error patterns for PostgreSQL.
func DefaultPostgresRetryableErrors() []string {
	return []string{
		"connection refused",
		"i/o timeout",
		"connection reset",
		"server closed the connection",
		"too many connections",
		"database system is starting up",
		"the database system is starting up",
		"connection reset by peer",
		"no connection could be made",
		"network is unreachable",
		"dial tcp",
		"connection timed out",
	}
}

// PostgresConfig returns retry configuration optimized for PostgreSQL connections.
func PostgresConfig() Config {
	cfg := DefaultConfig()
	cfg.RetryableErrors = DefaultPostgresRetryableErrors()
	return cfg
}
