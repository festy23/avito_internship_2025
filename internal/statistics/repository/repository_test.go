//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/statistics/model"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	err = db.Exec(`
		CREATE TABLE users (
			user_id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE teams (
			team_name VARCHAR(255) PRIMARY KEY
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE pull_requests (
			pull_request_id VARCHAR(255) PRIMARY KEY,
			pull_request_name VARCHAR(255) NOT NULL,
			author_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'OPEN'
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE pull_request_reviewers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestGetReviewersStatistics(t *testing.T) {
	db := setupTestDB(t)
	logger := zap.NewNop().Sugar()
	repo := New(db, logger)
	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		stats, err := repo.GetReviewersStatistics(ctx)
		require.NoError(t, err)
		assert.Empty(t, stats)
	})

	t.Run("with users and assignments", func(t *testing.T) {
		// Insert test data
		err := db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true).Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true).Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr1", "PR1", "u1", "OPEN").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr1", "u2").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr1", "u1").Error
		require.NoError(t, err)

		stats, err := repo.GetReviewersStatistics(ctx)
		require.NoError(t, err)
		assert.Len(t, stats, 2)

		// Find u2 (should have 1 assignment)
		var u2Stat *model.ReviewerStatistics
		for i := range stats {
			if stats[i].UserID == "u2" {
				u2Stat = &stats[i]
				break
			}
		}
		require.NotNil(t, u2Stat)
		assert.Equal(t, 1, u2Stat.AssignmentCount)
	})
}

func TestGetPullRequestStatistics(t *testing.T) {
	db := setupTestDB(t)
	logger := zap.NewNop().Sugar()
	repo := New(db, logger)
	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		stats, err := repo.GetPullRequestStatistics(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.TotalPRs)
		assert.Equal(t, 0, stats.OpenPRs)
		assert.Equal(t, 0, stats.MergedPRs)
	})

	t.Run("with PRs", func(t *testing.T) {
		// Insert test data
		err := db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr1", "PR1", "u1", "OPEN").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr2", "PR2", "u1", "MERGED").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr1", "u2").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr1", "u3").Error
		require.NoError(t, err)

		stats, err := repo.GetPullRequestStatistics(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 2, stats.TotalPRs)
		assert.Equal(t, 1, stats.OpenPRs)
		assert.Equal(t, 1, stats.MergedPRs)
		assert.Equal(t, 1, stats.PRsWith2Reviewers)
		assert.Equal(t, 1, stats.PRsWith0Reviewers)
	})
}
