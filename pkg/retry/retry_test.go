package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.InitialDelay)
	assert.Equal(t, 30*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Empty(t, cfg.RetryableErrors)
}

func TestDo_Success(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()

	err := Do(ctx, cfg, func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestDo_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.InitialDelay = 10 * time.Millisecond

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestDo_MaxAttempts(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.InitialDelay = 10 * time.Millisecond

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("persistent error")
	})

	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Contains(t, err.Error(), "persistent error")
}

func TestDo_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := DefaultConfig()
	cfg.MaxAttempts = 10
	cfg.InitialDelay = 100 * time.Millisecond

	attempts := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("temporary error")
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled"))
	assert.Less(t, attempts, 10)
}

func TestDo_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 5
	cfg.InitialDelay = 10 * time.Millisecond
	cfg.RetryableErrors = []string{"connection refused"}

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("invalid credentials")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should not retry non-retryable errors
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestDo_RetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.InitialDelay = 10 * time.Millisecond
	cfg.RetryableErrors = []string{"connection refused"}

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("connection refused")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestDoWithResult_Success(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()

	result, err := DoWithResult(ctx, cfg, func() (string, error) {
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestDoWithResult_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3
	cfg.InitialDelay = 10 * time.Millisecond

	attempts := 0
	result, err := DoWithResult(ctx, cfg, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, attempts)
}

func TestDoWithResult_MaxAttempts(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 2
	cfg.InitialDelay = 10 * time.Millisecond

	attempts := 0
	result, err := DoWithResult(ctx, cfg, func() (string, error) {
		attempts++
		return "", errors.New("persistent error")
	})

	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 2, attempts)
}

func TestCalculateDelay(t *testing.T) {
	cfg := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first retry",
			attempt:  0,
			expected: 1 * time.Second,
		},
		{
			name:     "second retry",
			attempt:  1,
			expected: 2 * time.Second,
		},
		{
			name:     "third retry",
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "fourth retry",
			attempt:  3,
			expected: 8 * time.Second,
		},
		{
			name:     "fifth retry",
			attempt:  4,
			expected: 16 * time.Second,
		},
		{
			name:     "sixth retry (capped)",
			attempt:  5,
			expected: 30 * time.Second, // Capped at maxDelay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(tt.attempt, cfg)
			// Allow small variance due to floating point arithmetic
			assert.InDelta(t, float64(tt.expected), float64(delay), float64(100*time.Millisecond))
		})
	}
}

func TestCalculateDelay_EdgeCases(t *testing.T) {
	cfg := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	// Negative attempt
	delay := calculateDelay(-1, cfg)
	assert.Equal(t, 1*time.Second, delay)

	// Zero multiplier
	cfg.Multiplier = 0
	delay = calculateDelay(1, cfg)
	assert.Equal(t, time.Duration(0), delay)
}

func TestAddJitter(t *testing.T) {
	delay := 1 * time.Second
	jittered := addJitter(delay)

	// Jitter should be within ±10% of original delay
	minDelay := delay - time.Duration(float64(delay)*0.1)
	maxDelay := delay + time.Duration(float64(delay)*0.1)

	assert.GreaterOrEqual(t, jittered, minDelay)
	assert.LessOrEqual(t, jittered, maxDelay)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		retryableErrs []string
		expectedRetry bool
	}{
		{
			name:          "nil error",
			err:           nil,
			retryableErrs: []string{"connection refused"},
			expectedRetry: false,
		},
		{
			name:          "no retryable errors specified - all retryable",
			err:           errors.New("any error"),
			retryableErrs: []string{},
			expectedRetry: true,
		},
		{
			name:          "matching retryable error",
			err:           errors.New("connection refused"),
			retryableErrs: []string{"connection refused"},
			expectedRetry: true,
		},
		{
			name:          "case insensitive match",
			err:           errors.New("CONNECTION REFUSED"),
			retryableErrs: []string{"connection refused"},
			expectedRetry: true,
		},
		{
			name:          "non-matching error",
			err:           errors.New("invalid credentials"),
			retryableErrs: []string{"connection refused"},
			expectedRetry: false,
		},
		{
			name:          "partial match",
			err:           errors.New("dial tcp: connection refused"),
			retryableErrs: []string{"connection refused"},
			expectedRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				RetryableErrors: tt.retryableErrs,
			}
			result := IsRetryableError(tt.err, cfg)
			assert.Equal(t, tt.expectedRetry, result)
		})
	}
}

func TestDefaultPostgresRetryableErrors(t *testing.T) {
	errors := DefaultPostgresRetryableErrors()
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors, "connection refused")
	assert.Contains(t, errors, "i/o timeout")
}

func TestPostgresConfig(t *testing.T) {
	cfg := PostgresConfig()
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.NotEmpty(t, cfg.RetryableErrors)
	assert.Contains(t, cfg.RetryableErrors, "connection refused")
}

func TestDo_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := DefaultConfig()
	cfg.MaxAttempts = 10
	cfg.InitialDelay = 100 * time.Millisecond

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("temporary error")
	})

	assert.Error(t, err)
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "context deadline exceeded"))
	assert.Less(t, attempts, 10)
}

func TestDoWithResult_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := DefaultConfig()
	cfg.MaxAttempts = 10
	cfg.InitialDelay = 100 * time.Millisecond

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	attempts := 0
	result, err := DoWithResult(ctx, cfg, func() (int, error) {
		attempts++
		return 0, errors.New("temporary error")
	})

	assert.Error(t, err)
	assert.Equal(t, 0, result)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled"))
}

func TestDo_ZeroMaxAttempts(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 0

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxAttempts must be greater than 0")
	assert.Equal(t, 0, attempts) // Should not execute with 0 attempts
}

func TestDo_OneAttempt(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.MaxAttempts = 1

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}

func TestCalculateDelay_MaxDelayCap(t *testing.T) {
	cfg := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	// Attempt that would exceed maxDelay
	delay := calculateDelay(10, cfg)
	assert.Equal(t, 5*time.Second, delay)
}

func TestAddJitter_ZeroDelay(t *testing.T) {
	delay := time.Duration(0)
	jittered := addJitter(delay)
	assert.Equal(t, time.Duration(0), jittered)
}
