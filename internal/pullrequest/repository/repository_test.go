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

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
)

type testPullRequest struct {
	PullRequestID   string     `gorm:"primaryKey;column:pull_request_id"`
	PullRequestName string     `gorm:"column:pull_request_name;not null"`
	AuthorID        string     `gorm:"column:author_id;not null"`
	Status          string     `gorm:"column:status;not null"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	MergedAt        *time.Time `gorm:"column:merged_at"`
}

func (testPullRequest) TableName() string {
	return "pull_requests"
}

type testPullRequestReviewer struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	PullRequestID string    `gorm:"column:pull_request_id;not null"`
	UserID        string    `gorm:"column:user_id;not null"`
	AssignedAt    time.Time `gorm:"column:assigned_at"`
}

func (testPullRequestReviewer) TableName() string {
	return "pull_request_reviewers"
}

type testTeam struct {
	TeamName  string    `gorm:"primaryKey;column:team_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (testTeam) TableName() string {
	return "teams"
}

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

	err = db.AutoMigrate(&testPullRequest{}, &testPullRequestReviewer{}, &testTeam{}, &testUser{})
	require.NoError(t, err)

	return db
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)

		pr, err := repo.Create(ctx, "pr-1", "Add feature", "u1")

		require.NoError(t, err)
		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "Add feature", pr.PullRequestName)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, "OPEN", pr.Status)
		assert.False(t, pr.CreatedAt.IsZero())
		assert.Nil(t, pr.MergedAt)

		var dbPR testPullRequest
		db.Where("pull_request_id = ?", "pr-1").First(&dbPR)
		assert.Equal(t, "pr-1", dbPR.PullRequestID)
		assert.Equal(t, "OPEN", dbPR.Status)
	})

	t.Run("duplicate pull request ID", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Existing PR",
			"u1",
			"OPEN",
		)

		pr, err := repo.Create(ctx, "pr-1", "New PR", "u1")

		assert.Nil(t, pr)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestExists)
	})

	t.Run("invalid author_id", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		// Repository doesn't validate author existence - that's service layer responsibility
		// In SQLite, foreign key constraints are not enforced by default
		// This test verifies that repository allows creating PR with non-existent author
		pr, err := repo.Create(ctx, "pr-1", "Add feature", "nonexistent")

		// Repository should succeed (author validation is done at service layer)
		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "nonexistent", pr.AuthorID)
	})
}

func TestRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)

		pr, err := repo.GetByID(ctx, "pr-1")

		require.NoError(t, err)
		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "Add feature", pr.PullRequestName)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, "OPEN", pr.Status)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		pr, err := repo.GetByID(ctx, "nonexistent")

		assert.Nil(t, pr)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestNotFound)
	})
}

func TestRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("success - merge", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)

		now := time.Now()
		err := repo.UpdateStatus(ctx, "pr-1", "MERGED", &now)

		require.NoError(t, err)

		var dbPR testPullRequest
		db.Where("pull_request_id = ?", "pr-1").First(&dbPR)
		assert.Equal(t, "MERGED", dbPR.Status)
		assert.NotNil(t, dbPR.MergedAt)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		now := time.Now()
		err := repo.UpdateStatus(ctx, "nonexistent", "MERGED", &now)

		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestNotFound)
	})

	t.Run("idempotent merge", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		mergedAt := time.Now()
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) VALUES (?, ?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"MERGED",
			mergedAt,
		)

		newMergedAt := time.Now()
		err := repo.UpdateStatus(ctx, "pr-1", "MERGED", &newMergedAt)

		require.NoError(t, err)

		var dbPR testPullRequest
		db.Where("pull_request_id = ?", "pr-1").First(&dbPR)
		assert.Equal(t, "MERGED", dbPR.Status)
	})
}

func TestRepository_AssignReviewer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)

		err := repo.AssignReviewer(ctx, "pr-1", "u2")

		require.NoError(t, err)

		var reviewer testPullRequestReviewer
		db.Where("pull_request_id = ? AND user_id = ?", "pr-1", "u2").First(&reviewer)
		assert.Equal(t, "pr-1", reviewer.PullRequestID)
		assert.Equal(t, "u2", reviewer.UserID)
	})

	t.Run("exceed limit of 2", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u4", "David", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u2",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u3",
		)

		err := repo.AssignReviewer(ctx, "pr-1", "u4")

		assert.Error(t, err)
		assert.ErrorIs(t, err, pullrequestModel.ErrMaxReviewersExceeded)
	})

	t.Run("duplicate reviewer", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u2",
		)

		err := repo.AssignReviewer(ctx, "pr-1", "u2")

		assert.Error(t, err)
		assert.ErrorIs(t, err, pullrequestModel.ErrReviewerAlreadyAssigned)
	})

	t.Run("assign second reviewer", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u2",
		)

		err := repo.AssignReviewer(ctx, "pr-1", "u3")
		require.NoError(t, err)

		reviewers, err := repo.GetReviewers(ctx, "pr-1")
		require.NoError(t, err)
		assert.Len(t, reviewers, 2)
		assert.Contains(t, reviewers, "u2")
		assert.Contains(t, reviewers, "u3")
	})

	t.Run("error getting reviewers count", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		// Close DB to simulate error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		err := repo.AssignReviewer(ctx, "pr-1", "u2")
		assert.Error(t, err)
	})

	// Note: Author cannot be reviewer check is enforced by PostgreSQL trigger
	// This is tested in E2E tests with real PostgreSQL database

	t.Run("database error on create", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		sqlDB, _ := db.DB()
		sqlDB.Close()

		err := repo.AssignReviewer(ctx, "pr-1", "u2")
		assert.Error(t, err)
	})
}

func TestRepository_RemoveReviewer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u2",
		)

		err := repo.RemoveReviewer(ctx, "pr-1", "u2")

		require.NoError(t, err)

		var count int64
		db.Model(&testPullRequestReviewer{}).
			Where("pull_request_id = ? AND user_id = ?", "pr-1", "u2").
			Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)

		err := repo.RemoveReviewer(ctx, "pr-1", "u2")

		assert.ErrorIs(t, err, pullrequestModel.ErrReviewerNotAssigned)
	})
}

func TestRepository_GetReviewers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u2",
		)
		db.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)",
			"pr-1",
			"u3",
		)

		reviewers, err := repo.GetReviewers(ctx, "pr-1")

		require.NoError(t, err)
		assert.Len(t, reviewers, 2)
		assert.Contains(t, reviewers, "u2")
		assert.Contains(t, reviewers, "u3")
	})

	t.Run("empty list", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"OPEN",
		)

		reviewers, err := repo.GetReviewers(ctx, "pr-1")

		require.NoError(t, err)
		assert.Empty(t, reviewers)
	})
}

func TestRepository_GetActiveTeamMembers(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", false)

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "")

		require.NoError(t, err)
		assert.Len(t, members, 2)
		userIDs := []string{members[0].UserID, members[1].UserID}
		assert.Contains(t, userIDs, "u1")
		assert.Contains(t, userIDs, "u2")
		assert.NotContains(t, userIDs, "u3")
	})

	t.Run("exclude user", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "u1")

		require.NoError(t, err)
		assert.Len(t, members, 1)
		assert.Equal(t, "u2", members[0].UserID)
	})

	t.Run("only active members", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", false)

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "")

		require.NoError(t, err)
		assert.Len(t, members, 1)
		assert.Equal(t, "u1", members[0].UserID)
	})
}

func TestRepository_GetUserTeam(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)

		teamName, err := repo.GetUserTeam(ctx, "u1")

		require.NoError(t, err)
		assert.Equal(t, "backend", teamName)
	})

	t.Run("user not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		teamName, err := repo.GetUserTeam(ctx, "nonexistent")

		assert.Empty(t, teamName)
		assert.ErrorIs(t, err, pullrequestModel.ErrAuthorNotFound)
	})
}

// Extended tests to increase coverage to 85%+

func TestRepository_Create_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("database error handling", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		// Close DB to simulate error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		pr, err := repo.Create(ctx, "pr-1", "Add feature", "u1")
		assert.Nil(t, pr)
		assert.Error(t, err)
	})

	t.Run("isDuplicateError with UNIQUE constraint", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Existing", "u1", "OPEN",
		)

		pr, err := repo.Create(ctx, "pr-1", "Duplicate", "u1")
		assert.Nil(t, pr)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestExists)
	})
}

func TestRepository_GetByID_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		sqlDB, _ := db.DB()
		sqlDB.Close()

		pr, err := repo.GetByID(ctx, "pr-1")
		assert.Nil(t, pr)
		assert.Error(t, err)
	})

	t.Run("get merged PR with merged_at", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		mergedAt := time.Now()
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) "+
				"VALUES (?, ?, ?, ?, ?)",
			"pr-1", "Merged PR", "u1", "MERGED", mergedAt,
		)

		pr, err := repo.GetByID(ctx, "pr-1")
		require.NoError(t, err)
		assert.Equal(t, "MERGED", pr.Status)
		require.NotNil(t, pr.MergedAt)
	})
}

func TestRepository_UpdateStatus_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("update to OPEN status", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)

		err := repo.UpdateStatus(ctx, "pr-1", "OPEN", nil)
		require.NoError(t, err)

		var dbPR testPullRequest
		db.Where("pull_request_id = ?", "pr-1").First(&dbPR)
		assert.Equal(t, "OPEN", dbPR.Status)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)
		sqlDB, _ := db.DB()
		sqlDB.Close()

		err := repo.UpdateStatus(ctx, "pr-1", "MERGED", nil)
		assert.Error(t, err)
	})
}

func TestRepository_AssignReviewer_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("assign when already at limit", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u4", "David", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u3")

		err := repo.AssignReviewer(ctx, "pr-1", "u4")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pullrequestModel.ErrMaxReviewersExceeded)
	})

	t.Run("duplicate reviewer via database constraint", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)
		// Create unique constraint manually
		db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_reviewer " +
			"ON pull_request_reviewers(pull_request_id, user_id)")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")

		err := repo.AssignReviewer(ctx, "pr-1", "u2")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pullrequestModel.ErrReviewerAlreadyAssigned)
	})
}

func TestRepository_RemoveReviewer_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("remove non-existent reviewer", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)

		// Removing non-existent reviewer returns error
		err := repo.RemoveReviewer(ctx, "pr-1", "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pullrequestModel.ErrReviewerNotAssigned)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")
		sqlDB, _ := db.DB()
		sqlDB.Close()

		err := repo.RemoveReviewer(ctx, "pr-1", "u2")
		assert.Error(t, err)
	})
}

func TestRepository_GetReviewers_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("get reviewers for PR with multiple reviewers", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN",
		)
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u3")

		reviewers, err := repo.GetReviewers(ctx, "pr-1")
		require.NoError(t, err)
		assert.Len(t, reviewers, 2)
		assert.Contains(t, reviewers, "u2")
		assert.Contains(t, reviewers, "u3")
	})

	t.Run("get reviewers for non-existent PR", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())

		reviewers, err := repo.GetReviewers(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, reviewers)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		sqlDB, _ := db.DB()
		sqlDB.Close()

		reviewers, err := repo.GetReviewers(ctx, "pr-1")
		assert.Nil(t, reviewers)
		assert.Error(t, err)
	})
}

func TestRepository_GetActiveTeamMembers_Extended(t *testing.T) {
	ctx := context.Background()

	t.Run("get active members excluding multiple users", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", 1)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", 1)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", 1)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u4", "David", "backend", 0)

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "u1")
		require.NoError(t, err)
		assert.Len(t, members, 2)
		assert.NotContains(t, []string{members[0].UserID, members[1].UserID}, "u1")
		assert.NotContains(t, []string{members[0].UserID, members[1].UserID}, "u4")
	})

	t.Run("team with no active members", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", 0)

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "u1")
		require.NoError(t, err)
		assert.Empty(t, members)
	})

	t.Run("database error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := New(db, zap.NewNop().Sugar())
		sqlDB, _ := db.DB()
		sqlDB.Close()

		members, err := repo.GetActiveTeamMembers(ctx, "backend", "u1")
		assert.Nil(t, members)
		assert.Error(t, err)
	})
}
