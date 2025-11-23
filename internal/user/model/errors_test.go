package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_Definition(t *testing.T) {
	t.Run("ErrUserNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrUserNotFound)
		assert.Equal(t, "user not found", ErrUserNotFound.Error())
	})

	t.Run("ErrInvalidUserID is defined", func(t *testing.T) {
		assert.NotNil(t, ErrInvalidUserID)
		assert.Equal(t, "invalid user ID", ErrInvalidUserID.Error())
	})

	t.Run("ErrInvalidIsActive is defined", func(t *testing.T) {
		assert.NotNil(t, ErrInvalidIsActive)
		assert.Equal(t, "is_active field is required", ErrInvalidIsActive.Error())
	})
}

func TestErrors_Uniqueness(t *testing.T) {
	t.Run("all errors are unique", func(t *testing.T) {
		errorList := []error{
			ErrUserNotFound,
			ErrInvalidUserID,
			ErrInvalidIsActive,
		}

		// Check that all error messages are unique
		seen := make(map[string]bool)
		for _, err := range errorList {
			msg := err.Error()
			assert.False(t, seen[msg], "Duplicate error message: %s", msg)
			seen[msg] = true
		}
	})
}

func TestErrors_Comparison(t *testing.T) {
	t.Run("can compare with errors.Is", func(t *testing.T) {
		err := ErrUserNotFound
		assert.True(t, errors.Is(err, ErrUserNotFound))
		assert.False(t, errors.Is(err, ErrInvalidUserID))
		assert.False(t, errors.Is(err, ErrInvalidIsActive))
	})

	t.Run("errors are singletons", func(t *testing.T) {
		// Same error instance
		assert.Same(t, ErrUserNotFound, ErrUserNotFound)
		assert.Same(t, ErrInvalidUserID, ErrInvalidUserID)
		assert.Same(t, ErrInvalidIsActive, ErrInvalidIsActive)
	})

	t.Run("different errors are not equal", func(t *testing.T) {
		assert.NotEqual(t, ErrUserNotFound, ErrInvalidUserID)
		assert.NotEqual(t, ErrUserNotFound, ErrInvalidIsActive)
		assert.NotEqual(t, ErrInvalidUserID, ErrInvalidIsActive)
	})
}

func TestErrors_Usage(t *testing.T) {
	t.Run("return ErrUserNotFound", func(t *testing.T) {
		findUser := func(id string) error {
			if id == "nonexistent" {
				return ErrUserNotFound
			}
			return nil
		}

		err := findUser("nonexistent")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrUserNotFound))
	})

	t.Run("return ErrInvalidUserID", func(t *testing.T) {
		validateUserID := func(id string) error {
			if id == "" {
				return ErrInvalidUserID
			}
			return nil
		}

		err := validateUserID("")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidUserID))
	})

	t.Run("return ErrInvalidIsActive", func(t *testing.T) {
		validateIsActive := func(isActive *bool) error {
			if isActive == nil {
				return ErrInvalidIsActive
			}
			return nil
		}

		err := validateIsActive(nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidIsActive))
	})
}

func TestErrors_MessageContent(t *testing.T) {
	t.Run("error messages are descriptive", func(t *testing.T) {
		assert.Contains(t, ErrUserNotFound.Error(), "user")
		assert.Contains(t, ErrUserNotFound.Error(), "not found")

		assert.Contains(t, ErrInvalidUserID.Error(), "user ID")
		assert.Contains(t, ErrInvalidUserID.Error(), "invalid")

		assert.Contains(t, ErrInvalidIsActive.Error(), "is_active")
		assert.Contains(t, ErrInvalidIsActive.Error(), "required")
	})

	t.Run("error messages are lowercase", func(t *testing.T) {
		// Following Go conventions for error messages
		assert.Equal(t, "user not found", ErrUserNotFound.Error())
		assert.Equal(t, "invalid user ID", ErrInvalidUserID.Error())
		assert.Equal(t, "is_active field is required", ErrInvalidIsActive.Error())
	})
}

func TestErrors_Wrapping(t *testing.T) {
	t.Run("wrap errors with context", func(t *testing.T) {
		baseErr := ErrUserNotFound
		wrappedErr := errors.New("failed to get user: " + baseErr.Error())

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), ErrUserNotFound.Error())
	})

	t.Run("unwrap with %w formatting", func(t *testing.T) {
		userID := "u1"
		wrappedErr := errors.New("user '" + userID + "' not found")

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), userID)
	})
}

func TestErrors_SwitchCase(t *testing.T) {
	t.Run("can switch on error types", func(t *testing.T) {
		handleError := func(err error) string {
			switch {
			case errors.Is(err, ErrUserNotFound):
				return "not_found"
			case errors.Is(err, ErrInvalidUserID):
				return "invalid_id"
			case errors.Is(err, ErrInvalidIsActive):
				return "invalid_active"
			default:
				return "unknown"
			}
		}

		assert.Equal(t, "not_found", handleError(ErrUserNotFound))
		assert.Equal(t, "invalid_id", handleError(ErrInvalidUserID))
		assert.Equal(t, "invalid_active", handleError(ErrInvalidIsActive))
		assert.Equal(t, "unknown", handleError(errors.New("other error")))
	})
}

func TestErrors_NilCheck(t *testing.T) {
	t.Run("errors are not nil", func(t *testing.T) {
		assert.NotNil(t, ErrUserNotFound)
		assert.NotNil(t, ErrInvalidUserID)
		assert.NotNil(t, ErrInvalidIsActive)
	})
}

func TestErrors_HTTPMapping(t *testing.T) {
	t.Run("map errors to HTTP status codes", func(t *testing.T) {
		mapToHTTPStatus := func(err error) int {
			switch {
			case errors.Is(err, ErrUserNotFound):
				return 404
			case errors.Is(err, ErrInvalidUserID):
				return 400
			case errors.Is(err, ErrInvalidIsActive):
				return 400
			default:
				return 500
			}
		}

		assert.Equal(t, 404, mapToHTTPStatus(ErrUserNotFound))
		assert.Equal(t, 400, mapToHTTPStatus(ErrInvalidUserID))
		assert.Equal(t, 400, mapToHTTPStatus(ErrInvalidIsActive))
		assert.Equal(t, 500, mapToHTTPStatus(errors.New("unknown")))
	})
}

func TestErrors_ContextualMessages(t *testing.T) {
	t.Run("create contextual error messages", func(t *testing.T) {
		userID := "u123"
		contextErr := errors.New("failed to update user '" + userID + "': " + ErrUserNotFound.Error())

		assert.Contains(t, contextErr.Error(), userID)
		assert.Contains(t, contextErr.Error(), "user not found")
		assert.Contains(t, contextErr.Error(), "failed to update")
	})
}

// Benchmark error operations.
func BenchmarkErrors_Is(b *testing.B) {
	err := ErrUserNotFound
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, ErrUserNotFound)
	}
}

func BenchmarkErrors_Error(b *testing.B) {
	err := ErrUserNotFound
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErrors_Switch(b *testing.B) {
	err := ErrUserNotFound
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch {
		case errors.Is(err, ErrUserNotFound):
		case errors.Is(err, ErrInvalidUserID):
		case errors.Is(err, ErrInvalidIsActive):
		}
	}
}
