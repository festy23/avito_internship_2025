package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/user/model"
)

type testUser struct {
	UserID    string    `gorm:"primaryKey;column:user_id"`
	Username  string    `gorm:"column:username;not null"`
	TeamName  string    `gorm:"column:team_name;not null"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (testUser) TableName() string {
	return "users"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Team struct {
		TeamName  string    `gorm:"primaryKey;column:team_name"`
		CreatedAt time.Time `gorm:"column:created_at"`
		UpdatedAt time.Time `gorm:"column:updated_at"`
	}

	type PullRequest struct {
		PullRequestID   string     `gorm:"primaryKey;column:pull_request_id"`
		PullRequestName string     `gorm:"column:pull_request_name;not null"`
		AuthorID        string     `gorm:"column:author_id;not null"`
		Status          string     `gorm:"column:status;not null"`
		CreatedAt       time.Time  `gorm:"column:created_at"`
		MergedAt        *time.Time `gorm:"column:merged_at"`
	}

	type PullRequestReviewer struct {
		ID            int       `gorm:"primaryKey;autoIncrement"`
		PullRequestID string    `gorm:"column:pull_request_id;not null"`
		UserID        string    `gorm:"column:user_id;not null"`
		AssignedAt    time.Time `gorm:"column:assigned_at"`
	}

	err = db.AutoMigrate(&Team{}, &testUser{}, &PullRequest{}, &PullRequestReviewer{})
	require.NoError(t, err)

	return db
}

func TestRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)

		user, err := repo.GetByID(ctx, "u1")

		require.NoError(t, err)
		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "Alice", user.Username)
		assert.Equal(t, "team1", user.TeamName)
		assert.True(t, user.IsActive)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		user, err := repo.GetByID(ctx, "nonexistent")

		assert.Nil(t, user)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})
}

func TestRepository_UpdateIsActive(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)

		user, err := repo.UpdateIsActive(ctx, "u1", false)

		require.NoError(t, err)
		assert.Equal(t, "u1", user.UserID)
		assert.False(t, user.IsActive)

		var updatedUser model.User
		db.Where("user_id = ?", "u1").First(&updatedUser)
		assert.False(t, updatedUser.IsActive)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		user, err := repo.UpdateIsActive(ctx, "nonexistent", false)

		assert.Nil(t, user)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})
}

func TestRepository_GetAssignedPullRequests(t *testing.T) {
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")

		require.NoError(t, err)
		assert.Empty(t, prs)
	})

	t.Run("multiple PRs sorted by created_at DESC", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "team1", true)
		// Create PRs with different timestamps (pr-2 is newer)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) VALUES (?, ?, ?, ?, datetime('now', '-1 day'))",
			"pr-1",
			"PR 1",
			"u2",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
			"pr-2",
			"PR 2",
			"u2",
			"MERGED",
		)
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u1")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-2", "u1")

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")

		require.NoError(t, err)
		require.Len(t, prs, 2)
		// Newer PR should be first (pr-2)
		assert.Equal(t, "pr-2", prs[0].PullRequestID)
		assert.Equal(t, "MERGED", prs[0].Status)
		assert.Equal(t, "pr-1", prs[1].PullRequestID)
		assert.Equal(t, "PR 1", prs[1].PullRequestName)
		assert.Equal(t, "u2", prs[1].AuthorID)
		assert.Equal(t, "OPEN", prs[1].Status)
	})
}

func TestRepository_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("user_id with max length (255 chars)", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		longUserID := string(make([]byte, 255))
		for i := range longUserID {
			longUserID = longUserID[:i] + "a" + longUserID[i+1:]
		}

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			longUserID, "Alice", "team1", true)

		user, err := repo.GetByID(ctx, longUserID)
		require.NoError(t, err)
		assert.Equal(t, longUserID, user.UserID)
	})

	t.Run("user_id with SQL special characters", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		specialUserID := "user'; DROP TABLE users; --"

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			specialUserID, "Alice", "team1", true)

		user, err := repo.GetByID(ctx, specialUserID)
		require.NoError(t, err)
		assert.Equal(t, specialUserID, user.UserID)
	})

	t.Run("user_id with unicode characters", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		unicodeUserID := "user_😀_тест_ユーザー"

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			unicodeUserID, "Alice", "team1", true)

		user, err := repo.GetByID(ctx, unicodeUserID)
		require.NoError(t, err)
		assert.Equal(t, unicodeUserID, user.UserID)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		sqlDB, _ := db.DB()
		sqlDB.Close()

		user, err := repo.GetByID(ctx, "u1")
		assert.Nil(t, user)
		assert.Error(t, err)
	})
}

func TestRepository_UpdateIsActive_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("activate inactive user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", 0)

		user, err := repo.UpdateIsActive(ctx, "u1", true)
		require.NoError(t, err)
		assert.True(t, user.IsActive)
		assert.Equal(t, "u1", user.UserID)
	})

	t.Run("deactivate active user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", 1)

		user, err := repo.UpdateIsActive(ctx, "u1", false)
		require.NoError(t, err)
		assert.False(t, user.IsActive)
	})

	t.Run("update non-existent user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		user, err := repo.UpdateIsActive(ctx, "nonexistent", true)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})

	t.Run("database error on update", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)
		sqlDB, _ := db.DB()
		sqlDB.Close()

		user, err := repo.UpdateIsActive(ctx, "u1", false)
		assert.Nil(t, user)
		assert.Error(t, err)
	})

	t.Run("update is_active and fetch updated user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)

		// Update is_active and verify the updated user is returned
		user, err := repo.UpdateIsActive(ctx, "u1", false)
		require.NoError(t, err)
		assert.False(t, user.IsActive, "is_active should be updated to false")
		assert.Equal(t, "u1", user.UserID)
		assert.Equal(t, "Alice", user.Username)
	})
}

func TestRepository_GetAssignedPullRequests_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("user with multiple assigned PRs", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "team1", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "PR 1", "u2", "OPEN")
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-2", "PR 2", "u2", "OPEN")
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-3", "PR 3", "u2", "MERGED")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u1")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-2", "u1")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-3", "u1")

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")
		require.NoError(t, err)
		assert.Len(t, prs, 3)
	})

	t.Run("user with no assigned PRs", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")
		require.NoError(t, err)
		assert.Empty(t, prs)
		assert.NotNil(t, prs)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		sqlDB, _ := db.DB()
		sqlDB.Close()

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")
		assert.Nil(t, prs)
		assert.Error(t, err)
	})

	t.Run("PRs ordered by created_at DESC", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "team1", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "team1", true)
		// Create PRs with different timestamps
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) "+
			"VALUES (?, ?, ?, ?, ?)",
			"pr-1", "PR 1", "u2", "OPEN", time.Now().Add(-2*time.Hour))
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) "+
			"VALUES (?, ?, ?, ?, ?)",
			"pr-2", "PR 2", "u2", "OPEN", time.Now().Add(-1*time.Hour))
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) "+
			"VALUES (?, ?, ?, ?, ?)",
			"pr-3", "PR 3", "u2", "OPEN", time.Now())
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u1")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-2", "u1")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-3", "u1")

		prs, err := repo.GetAssignedPullRequests(ctx, "u1")
		require.NoError(t, err)
		assert.Len(t, prs, 3)
		// Should be ordered DESC by created_at
		assert.Equal(t, "pr-3", prs[0].PullRequestID)
		assert.Equal(t, "pr-2", prs[1].PullRequestID)
		assert.Equal(t, "pr-1", prs[2].PullRequestID)
	})
}
