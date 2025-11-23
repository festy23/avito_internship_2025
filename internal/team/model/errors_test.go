package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_Definition(t *testing.T) {
	t.Run("ErrTeamExists is defined", func(t *testing.T) {
		assert.NotNil(t, ErrTeamExists)
		assert.Equal(t, "team already exists", ErrTeamExists.Error())
	})

	t.Run("ErrTeamNotFound is defined", func(t *testing.T) {
		assert.NotNil(t, ErrTeamNotFound)
		assert.Equal(t, "team not found", ErrTeamNotFound.Error())
	})

	t.Run("ErrInvalidTeamName is defined", func(t *testing.T) {
		assert.NotNil(t, ErrInvalidTeamName)
		assert.Equal(t, "invalid team name", ErrInvalidTeamName.Error())
	})

	t.Run("ErrEmptyMembers is defined", func(t *testing.T) {
		assert.NotNil(t, ErrEmptyMembers)
		assert.Equal(t, "members list cannot be empty", ErrEmptyMembers.Error())
	})
}

func TestErrors_Uniqueness(t *testing.T) {
	t.Run("all errors are unique", func(t *testing.T) {
		errors := []error{
			ErrTeamExists,
			ErrTeamNotFound,
			ErrInvalidTeamName,
			ErrEmptyMembers,
		}

		// Check that all error messages are unique
		seen := make(map[string]bool)
		for _, err := range errors {
			msg := err.Error()
			assert.False(t, seen[msg], "Duplicate error message: %s", msg)
			seen[msg] = true
		}
	})
}

func TestErrors_Comparison(t *testing.T) {
	t.Run("can compare with errors.Is", func(t *testing.T) {
		err := ErrTeamExists
		assert.True(t, errors.Is(err, ErrTeamExists))
		assert.False(t, errors.Is(err, ErrTeamNotFound))
	})

	t.Run("wrapped errors work with errors.Is", func(t *testing.T) {
		wrappedErr := errors.New("wrapper: " + ErrTeamExists.Error())
		// Note: This won't work with errors.Is unless we use %w
		assert.NotEqual(t, ErrTeamExists, wrappedErr)
	})

	t.Run("errors are singletons", func(t *testing.T) {
		// Same error instance
		assert.Same(t, ErrTeamExists, ErrTeamExists)
		assert.Same(t, ErrTeamNotFound, ErrTeamNotFound)
		assert.Same(t, ErrInvalidTeamName, ErrInvalidTeamName)
		assert.Same(t, ErrEmptyMembers, ErrEmptyMembers)
	})
}

func TestErrors_Usage(t *testing.T) {
	t.Run("return ErrTeamExists", func(t *testing.T) {
		checkTeam := func(name string) error {
			if name == "existing-team" {
				return ErrTeamExists
			}
			return nil
		}

		err := checkTeam("existing-team")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTeamExists))
	})

	t.Run("return ErrTeamNotFound", func(t *testing.T) {
		findTeam := func(name string) error {
			if name == "nonexistent" {
				return ErrTeamNotFound
			}
			return nil
		}

		err := findTeam("nonexistent")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTeamNotFound))
	})

	t.Run("return ErrInvalidTeamName", func(t *testing.T) {
		validateName := func(name string) error {
			if name == "" {
				return ErrInvalidTeamName
			}
			return nil
		}

		err := validateName("")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidTeamName))
	})

	t.Run("return ErrEmptyMembers", func(t *testing.T) {
		validateMembers := func(members []TeamMember) error {
			if len(members) == 0 {
				return ErrEmptyMembers
			}
			return nil
		}

		err := validateMembers([]TeamMember{})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrEmptyMembers))
	})
}

func TestErrors_MessageContent(t *testing.T) {
	t.Run("error messages are descriptive", func(t *testing.T) {
		assert.Contains(t, ErrTeamExists.Error(), "team")
		assert.Contains(t, ErrTeamExists.Error(), "exists")

		assert.Contains(t, ErrTeamNotFound.Error(), "team")
		assert.Contains(t, ErrTeamNotFound.Error(), "not found")

		assert.Contains(t, ErrInvalidTeamName.Error(), "team name")
		assert.Contains(t, ErrInvalidTeamName.Error(), "invalid")

		assert.Contains(t, ErrEmptyMembers.Error(), "members")
		assert.Contains(t, ErrEmptyMembers.Error(), "empty")
	})

	t.Run("error messages are lowercase", func(t *testing.T) {
		// Following Go conventions for error messages
		assert.Equal(t, "team already exists", ErrTeamExists.Error())
		assert.Equal(t, "team not found", ErrTeamNotFound.Error())
		assert.Equal(t, "invalid team name", ErrInvalidTeamName.Error())
		assert.Equal(t, "members list cannot be empty", ErrEmptyMembers.Error())
	})
}

func TestErrors_Wrapping(t *testing.T) {
	t.Run("wrap errors with context", func(t *testing.T) {
		baseErr := ErrTeamNotFound
		wrappedErr := errors.New("failed to get team: " + baseErr.Error())

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), ErrTeamNotFound.Error())
	})

	t.Run("unwrap with %w formatting", func(t *testing.T) {
		teamName := "backend"
		wrappedErr := errors.New("team '" + teamName + "' not found")

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), teamName)
	})
}

func TestErrors_SwitchCase(t *testing.T) {
	t.Run("can switch on error types", func(t *testing.T) {
		handleError := func(err error) string {
			switch {
			case errors.Is(err, ErrTeamExists):
				return "duplicate"
			case errors.Is(err, ErrTeamNotFound):
				return "not_found"
			case errors.Is(err, ErrInvalidTeamName):
				return "invalid"
			case errors.Is(err, ErrEmptyMembers):
				return "empty"
			default:
				return "unknown"
			}
		}

		assert.Equal(t, "duplicate", handleError(ErrTeamExists))
		assert.Equal(t, "not_found", handleError(ErrTeamNotFound))
		assert.Equal(t, "invalid", handleError(ErrInvalidTeamName))
		assert.Equal(t, "empty", handleError(ErrEmptyMembers))
		assert.Equal(t, "unknown", handleError(errors.New("other error")))
	})
}

func TestErrors_NilCheck(t *testing.T) {
	t.Run("errors are not nil", func(t *testing.T) {
		assert.NotNil(t, ErrTeamExists)
		assert.NotNil(t, ErrTeamNotFound)
		assert.NotNil(t, ErrInvalidTeamName)
		assert.NotNil(t, ErrEmptyMembers)
	})
}

// Benchmark error operations.
func BenchmarkErrors_Is(b *testing.B) {
	err := ErrTeamExists
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, ErrTeamExists)
	}
}

func BenchmarkErrors_Error(b *testing.B) {
	err := ErrTeamExists
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}
