package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPullRequest_TableName(t *testing.T) {
	t.Run("returns correct table name", func(t *testing.T) {
		pr := PullRequest{}
		assert.Equal(t, "pull_requests", pr.TableName())
	})
}

func TestPullRequestReviewer_TableName(t *testing.T) {
	t.Run("returns correct table name", func(t *testing.T) {
		reviewer := PullRequestReviewer{}
		assert.Equal(t, "pull_request_reviewers", reviewer.TableName())
	})
}

func TestPullRequest_Fields(t *testing.T) {
	t.Run("pull request struct has correct fields", func(t *testing.T) {
		now := time.Now()
		mergedAt := now.Add(1 * time.Hour)

		pr := PullRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
			Status:          StatusOPEN,
			CreatedAt:       now,
			MergedAt:        &mergedAt,
		}

		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "Add feature", pr.PullRequestName)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, StatusOPEN, pr.Status)
		assert.Equal(t, now, pr.CreatedAt)
		require.NotNil(t, pr.MergedAt)
		assert.Equal(t, mergedAt, *pr.MergedAt)
	})

	t.Run("pull request without merged_at", func(t *testing.T) {
		pr := PullRequest{
			MergedAt: nil,
		}

		assert.Nil(t, pr.MergedAt)
	})

	t.Run("pull request with different statuses", func(t *testing.T) {
		openPR := PullRequest{Status: StatusOPEN}
		assert.Equal(t, StatusOPEN, openPR.Status)

		mergedPR := PullRequest{Status: StatusMERGED}
		assert.Equal(t, StatusMERGED, mergedPR.Status)
	})
}

func TestPullRequestReviewer_Fields(t *testing.T) {
	t.Run("reviewer struct has correct fields", func(t *testing.T) {
		now := time.Now()

		reviewer := PullRequestReviewer{
			ID:            1,
			PullRequestID: "pr-1",
			UserID:        "u1",
			AssignedAt:    now,
		}

		assert.Equal(t, int64(1), reviewer.ID)
		assert.Equal(t, "pr-1", reviewer.PullRequestID)
		assert.Equal(t, "u1", reviewer.UserID)
		assert.Equal(t, now, reviewer.AssignedAt)
	})

	t.Run("reviewer with zero ID", func(t *testing.T) {
		reviewer := PullRequestReviewer{}

		assert.Equal(t, int64(0), reviewer.ID)
	})
}

func TestPullRequest_GORMIntegration(t *testing.T) {
	t.Run("creates pull request with GORM", func(t *testing.T) {
		db := setupTestDB(t)

		pr := &PullRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
			Status:          StatusOPEN,
		}

		err := db.Create(pr).Error
		require.NoError(t, err)

		assert.NotZero(t, pr.CreatedAt)
		assert.Nil(t, pr.MergedAt)
	})

	t.Run("creates merged pull request", func(t *testing.T) {
		db := setupTestDB(t)

		now := time.Now()
		pr := &PullRequest{
			PullRequestID:   "pr-2",
			PullRequestName: "Fix bug",
			AuthorID:        "u2",
			Status:          StatusMERGED,
			MergedAt:        &now,
		}

		err := db.Create(pr).Error
		require.NoError(t, err)

		var retrieved PullRequest
		err = db.First(&retrieved, "pull_request_id = ?", "pr-2").Error
		require.NoError(t, err)

		assert.Equal(t, StatusMERGED, retrieved.Status)
		require.NotNil(t, retrieved.MergedAt)
	})

	t.Run("pull_request_id is primary key", func(t *testing.T) {
		db := setupTestDB(t)

		pr1 := &PullRequest{PullRequestID: "pr-1", PullRequestName: "Feature", AuthorID: "u1", Status: StatusOPEN}
		err := db.Create(pr1).Error
		require.NoError(t, err)

		// Try to create another PR with same ID
		pr2 := &PullRequest{PullRequestID: "pr-1", PullRequestName: "Another", AuthorID: "u2", Status: StatusOPEN}
		err = db.Create(pr2).Error
		assert.Error(t, err) // Should fail due to primary key constraint
	})

	t.Run("retrieves pull request by id", func(t *testing.T) {
		db := setupTestDB(t)

		original := &PullRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
			Status:          StatusOPEN,
		}
		err := db.Create(original).Error
		require.NoError(t, err)

		var retrieved PullRequest
		err = db.First(&retrieved, "pull_request_id = ?", "pr-1").Error
		require.NoError(t, err)

		assert.Equal(t, original.PullRequestID, retrieved.PullRequestID)
		assert.Equal(t, original.PullRequestName, retrieved.PullRequestName)
		assert.Equal(t, original.AuthorID, retrieved.AuthorID)
		assert.Equal(t, original.Status, retrieved.Status)
	})
}

func TestPullRequestReviewer_GORMIntegration(t *testing.T) {
	t.Run("creates reviewer assignment", func(t *testing.T) {
		db := setupTestDB(t)

		reviewer := &PullRequestReviewer{
			PullRequestID: "pr-1",
			UserID:        "u1",
		}

		err := db.Create(reviewer).Error
		require.NoError(t, err)

		assert.NotZero(t, reviewer.ID)
		assert.NotZero(t, reviewer.AssignedAt)
	})

	t.Run("retrieves reviewers for PR", func(t *testing.T) {
		db := setupTestDB(t)

		// Create multiple reviewers for same PR
		reviewer1 := &PullRequestReviewer{PullRequestID: "pr-1", UserID: "u1"}
		reviewer2 := &PullRequestReviewer{PullRequestID: "pr-1", UserID: "u2"}
		reviewer3 := &PullRequestReviewer{PullRequestID: "pr-2", UserID: "u3"}

		err := db.Create(reviewer1).Error
		require.NoError(t, err)
		err = db.Create(reviewer2).Error
		require.NoError(t, err)
		err = db.Create(reviewer3).Error
		require.NoError(t, err)

		// Query reviewers for pr-1
		var reviewers []PullRequestReviewer
		err = db.Where("pull_request_id = ?", "pr-1").Find(&reviewers).Error
		require.NoError(t, err)

		assert.Len(t, reviewers, 2)
	})

	t.Run("ID is auto-incremented", func(t *testing.T) {
		db := setupTestDB(t)

		reviewer1 := &PullRequestReviewer{PullRequestID: "pr-1", UserID: "u1"}
		err := db.Create(reviewer1).Error
		require.NoError(t, err)

		reviewer2 := &PullRequestReviewer{PullRequestID: "pr-1", UserID: "u2"}
		err = db.Create(reviewer2).Error
		require.NoError(t, err)

		assert.NotEqual(t, reviewer1.ID, reviewer2.ID)
		assert.Greater(t, reviewer2.ID, reviewer1.ID)
	})
}

func TestPullRequest_ZeroValue(t *testing.T) {
	t.Run("zero value pull request", func(t *testing.T) {
		var pr PullRequest

		assert.Empty(t, pr.PullRequestID)
		assert.Empty(t, pr.PullRequestName)
		assert.Empty(t, pr.AuthorID)
		assert.Empty(t, pr.Status)
		assert.True(t, pr.CreatedAt.IsZero())
		assert.Nil(t, pr.MergedAt)
	})

	t.Run("zero value reviewer", func(t *testing.T) {
		var reviewer PullRequestReviewer

		assert.Zero(t, reviewer.ID)
		assert.Empty(t, reviewer.PullRequestID)
		assert.Empty(t, reviewer.UserID)
		assert.True(t, reviewer.AssignedAt.IsZero())
	})
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	// Create pull_requests table (SQLite compatible)
	err = db.Exec(`
		CREATE TABLE pull_requests (
			pull_request_id VARCHAR(255) PRIMARY KEY,
			pull_request_name VARCHAR(255) NOT NULL,
			author_id VARCHAR(255) NOT NULL,
			status VARCHAR(10) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			merged_at TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Create pull_request_reviewers table
	err = db.Exec(`
		CREATE TABLE pull_request_reviewers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Create indexes
	err = db.Exec(`CREATE INDEX idx_pull_requests_author_id ON pull_requests(author_id)`).Error
	require.NoError(t, err)
	err = db.Exec(`CREATE INDEX idx_pull_requests_status ON pull_requests(status)`).Error
	require.NoError(t, err)
	err = db.Exec(`CREATE INDEX idx_reviewers_pull_request_id ON pull_request_reviewers(pull_request_id)`).Error
	require.NoError(t, err)
	err = db.Exec(`CREATE INDEX idx_reviewers_user_id ON pull_request_reviewers(user_id)`).Error
	require.NoError(t, err)

	return db
}

// Benchmark tests.
func BenchmarkPullRequest_TableName(b *testing.B) {
	pr := PullRequest{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pr.TableName()
	}
}

func BenchmarkPullRequestReviewer_TableName(b *testing.B) {
	reviewer := PullRequestReviewer{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reviewer.TableName()
	}
}

func TestValidateStatus(t *testing.T) {
	t.Run("valid OPEN status", func(t *testing.T) {
		err := ValidateStatus(StatusOPEN)
		assert.NoError(t, err)
	})

	t.Run("valid MERGED status", func(t *testing.T) {
		err := ValidateStatus(StatusMERGED)
		assert.NoError(t, err)
	})

	t.Run("invalid status - empty string", func(t *testing.T) {
		err := ValidateStatus("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid status - random string", func(t *testing.T) {
		err := ValidateStatus("INVALID")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid status - lowercase", func(t *testing.T) {
		err := ValidateStatus("open")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid status - merged lowercase", func(t *testing.T) {
		err := ValidateStatus("merged")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid status - partial match", func(t *testing.T) {
		err := ValidateStatus("OPENED")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("invalid status - special characters", func(t *testing.T) {
		err := ValidateStatus("OPEN'; DROP TABLE")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}
