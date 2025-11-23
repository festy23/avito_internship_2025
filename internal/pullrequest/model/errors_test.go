package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_Definition(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrPullRequestExists", ErrPullRequestExists, "pull request already exists"},
		{"ErrPullRequestNotFound", ErrPullRequestNotFound, "pull request not found"},
		{"ErrPullRequestMerged", ErrPullRequestMerged, "pull request is merged"},
		{"ErrReviewerNotAssigned", ErrReviewerNotAssigned, "reviewer is not assigned to this PR"},
		{"ErrNoCandidate", ErrNoCandidate, "no active replacement candidate in team"},
		{"ErrAuthorNotFound", ErrAuthorNotFound, "author not found"},
		{"ErrInvalidPullRequestID", ErrInvalidPullRequestID, "invalid pull request ID"},
		{"ErrInvalidAuthorID", ErrInvalidAuthorID, "author_id must be between 1 and 255 characters"},
		{"ErrMaxReviewersExceeded", ErrMaxReviewersExceeded, "maximum 2 reviewers allowed per pull request"},
		{"ErrReviewerAlreadyAssigned", ErrReviewerAlreadyAssigned, "reviewer already assigned to this pull request"},
		{"ErrAuthorCannotBeReviewer", ErrAuthorCannotBeReviewer, "author cannot be assigned as reviewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrors_Uniqueness(t *testing.T) {
	t.Run("all errors are unique", func(t *testing.T) {
		errorList := []error{
			ErrPullRequestExists,
			ErrPullRequestNotFound,
			ErrPullRequestMerged,
			ErrReviewerNotAssigned,
			ErrNoCandidate,
			ErrAuthorNotFound,
			ErrInvalidPullRequestID,
			ErrInvalidAuthorID,
			ErrMaxReviewersExceeded,
			ErrReviewerAlreadyAssigned,
			ErrAuthorCannotBeReviewer,
		}

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
		err := ErrPullRequestNotFound
		assert.True(t, errors.Is(err, ErrPullRequestNotFound))
		assert.False(t, errors.Is(err, ErrPullRequestExists))
		assert.False(t, errors.Is(err, ErrPullRequestMerged))
	})

	t.Run("errors are singletons", func(t *testing.T) {
		assert.Same(t, ErrPullRequestExists, ErrPullRequestExists)
		assert.Same(t, ErrPullRequestNotFound, ErrPullRequestNotFound)
		assert.Same(t, ErrPullRequestMerged, ErrPullRequestMerged)
	})

	t.Run("different errors are not equal", func(t *testing.T) {
		assert.NotEqual(t, ErrPullRequestExists, ErrPullRequestNotFound)
		assert.NotEqual(t, ErrPullRequestMerged, ErrReviewerNotAssigned)
		assert.NotEqual(t, ErrNoCandidate, ErrAuthorNotFound)
	})
}

func TestErrors_Usage_NotFound(t *testing.T) {
	findPR := func(id string) error {
		if id == "nonexistent" {
			return ErrPullRequestNotFound
		}
		return nil
	}

	err := findPR("nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPullRequestNotFound))
}

func TestErrors_Usage_Merged(t *testing.T) {
	reassignReviewer := func(prID string, merged bool) error {
		if merged {
			return ErrPullRequestMerged
		}
		return nil
	}

	err := reassignReviewer("pr-1", true)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPullRequestMerged))
}

func TestErrors_Usage_ReviewerNotAssigned(t *testing.T) {
	removeReviewer := func(reviewers []string, userID string) error {
		for _, r := range reviewers {
			if r == userID {
				return nil
			}
		}
		return ErrReviewerNotAssigned
	}

	err := removeReviewer([]string{"u1", "u2"}, "u3")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrReviewerNotAssigned))
}

func TestErrors_Usage_MaxReviewersExceeded(t *testing.T) {
	addReviewer := func(current int) error {
		if current >= 2 {
			return ErrMaxReviewersExceeded
		}
		return nil
	}

	err := addReviewer(2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrMaxReviewersExceeded))
}

func TestErrors_Usage_AuthorCannotBeReviewer(t *testing.T) {
	validateReviewer := func(authorID, reviewerID string) error {
		if authorID == reviewerID {
			return ErrAuthorCannotBeReviewer
		}
		return nil
	}

	err := validateReviewer("u1", "u1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAuthorCannotBeReviewer))
}

func TestErrors_MessageContent(t *testing.T) {
	t.Run("error messages are descriptive", func(t *testing.T) {
		assert.Contains(t, ErrPullRequestExists.Error(), "pull request")
		assert.Contains(t, ErrPullRequestExists.Error(), "exists")

		assert.Contains(t, ErrPullRequestNotFound.Error(), "pull request")
		assert.Contains(t, ErrPullRequestNotFound.Error(), "not found")

		assert.Contains(t, ErrPullRequestMerged.Error(), "merged")

		assert.Contains(t, ErrReviewerNotAssigned.Error(), "reviewer")
		assert.Contains(t, ErrReviewerNotAssigned.Error(), "not assigned")

		assert.Contains(t, ErrNoCandidate.Error(), "candidate")

		assert.Contains(t, ErrMaxReviewersExceeded.Error(), "2 reviewers")
		assert.Contains(t, ErrMaxReviewersExceeded.Error(), "maximum")
	})
}

func TestErrors_Wrapping(t *testing.T) {
	t.Run("wrap errors with context", func(t *testing.T) {
		baseErr := ErrPullRequestNotFound
		wrappedErr := errors.New("failed to get PR: " + baseErr.Error())

		assert.Error(t, wrappedErr)
		assert.Contains(t, wrappedErr.Error(), ErrPullRequestNotFound.Error())
	})

	t.Run("contextual error messages", func(t *testing.T) {
		prID := "pr-123"
		contextErr := errors.New("PR '" + prID + "' " + ErrPullRequestNotFound.Error())

		assert.Contains(t, contextErr.Error(), prID)
		assert.Contains(t, contextErr.Error(), "not found")
	})
}

func TestErrors_SwitchCase(t *testing.T) {
	t.Run("can switch on error types", func(t *testing.T) {
		handleError := func(err error) string {
			switch {
			case errors.Is(err, ErrPullRequestExists):
				return "duplicate"
			case errors.Is(err, ErrPullRequestNotFound):
				return "not_found"
			case errors.Is(err, ErrPullRequestMerged):
				return "merged"
			case errors.Is(err, ErrReviewerNotAssigned):
				return "not_assigned"
			case errors.Is(err, ErrNoCandidate):
				return "no_candidate"
			case errors.Is(err, ErrAuthorNotFound):
				return "author_not_found"
			case errors.Is(err, ErrInvalidPullRequestID):
				return "invalid_id"
			case errors.Is(err, ErrMaxReviewersExceeded):
				return "max_reviewers"
			case errors.Is(err, ErrAuthorCannotBeReviewer):
				return "author_reviewer"
			default:
				return "unknown"
			}
		}

		assert.Equal(t, "duplicate", handleError(ErrPullRequestExists))
		assert.Equal(t, "not_found", handleError(ErrPullRequestNotFound))
		assert.Equal(t, "merged", handleError(ErrPullRequestMerged))
		assert.Equal(t, "not_assigned", handleError(ErrReviewerNotAssigned))
		assert.Equal(t, "no_candidate", handleError(ErrNoCandidate))
		assert.Equal(t, "unknown", handleError(errors.New("other")))
	})
}

func TestErrors_HTTPMapping(t *testing.T) {
	t.Run("map errors to HTTP status codes", func(t *testing.T) {
		mapToHTTPStatus := func(err error) int {
			switch {
			case errors.Is(err, ErrPullRequestExists):
				return 409
			case errors.Is(err, ErrPullRequestNotFound), errors.Is(err, ErrAuthorNotFound):
				return 404
			case errors.Is(err, ErrPullRequestMerged):
				return 409
			case errors.Is(err, ErrReviewerNotAssigned):
				return 409
			case errors.Is(err, ErrNoCandidate):
				return 409
			case errors.Is(err, ErrInvalidPullRequestID), errors.Is(err, ErrInvalidAuthorID):
				return 400
			case errors.Is(err, ErrMaxReviewersExceeded):
				return 400
			case errors.Is(err, ErrReviewerAlreadyAssigned):
				return 409
			case errors.Is(err, ErrAuthorCannotBeReviewer):
				return 400
			default:
				return 500
			}
		}

		assert.Equal(t, 409, mapToHTTPStatus(ErrPullRequestExists))
		assert.Equal(t, 404, mapToHTTPStatus(ErrPullRequestNotFound))
		assert.Equal(t, 409, mapToHTTPStatus(ErrPullRequestMerged))
		assert.Equal(t, 400, mapToHTTPStatus(ErrInvalidPullRequestID))
		assert.Equal(t, 500, mapToHTTPStatus(errors.New("unknown")))
	})
}

func TestErrors_NilCheck(t *testing.T) {
	t.Run("errors are not nil", func(t *testing.T) {
		errorList := []error{
			ErrPullRequestExists,
			ErrPullRequestNotFound,
			ErrPullRequestMerged,
			ErrReviewerNotAssigned,
			ErrNoCandidate,
			ErrAuthorNotFound,
			ErrInvalidPullRequestID,
			ErrInvalidAuthorID,
			ErrMaxReviewersExceeded,
			ErrReviewerAlreadyAssigned,
			ErrAuthorCannotBeReviewer,
		}

		for _, err := range errorList {
			assert.NotNil(t, err)
		}
	})
}

// Benchmark error operations.
func BenchmarkErrors_Is(b *testing.B) {
	err := ErrPullRequestNotFound
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, ErrPullRequestNotFound)
	}
}

func BenchmarkErrors_Switch(b *testing.B) {
	err := ErrPullRequestNotFound
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch {
		case errors.Is(err, ErrPullRequestExists):
		case errors.Is(err, ErrPullRequestNotFound):
		case errors.Is(err, ErrPullRequestMerged):
		case errors.Is(err, ErrReviewerNotAssigned):
		}
	}
}
