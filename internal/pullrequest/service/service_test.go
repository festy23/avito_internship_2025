package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	"github.com/festy23/avito_internship/internal/pullrequest/repository"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Create(ctx context.Context, prID, prName, authorID string) (*pullrequestModel.PullRequest, error) {
	args := m.Called(ctx, prID, prName, authorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pullrequestModel.PullRequest), args.Error(1)
}

func (m *mockRepository) GetByID(ctx context.Context, prID string) (*pullrequestModel.PullRequest, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pullrequestModel.PullRequest), args.Error(1)
}

func (m *mockRepository) UpdateStatus(ctx context.Context, prID string, status string, mergedAt *time.Time) error {
	args := m.Called(ctx, prID, status, mergedAt)
	return args.Error(0)
}

func (m *mockRepository) AssignReviewer(ctx context.Context, prID, userID string) error {
	args := m.Called(ctx, prID, userID)
	return args.Error(0)
}

func (m *mockRepository) RemoveReviewer(ctx context.Context, prID, userID string) error {
	args := m.Called(ctx, prID, userID)
	return args.Error(0)
}

func (m *mockRepository) GetReviewers(ctx context.Context, prID string) ([]string, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]userModel.User, error) {
	args := m.Called(ctx, teamName, excludeUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]userModel.User), args.Error(1)
}

func (m *mockRepository) GetUserTeam(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestService_CreatePullRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success with 2 reviewers", func(t *testing.T) {
		db := setupTestDB(t)

		// Create tables
		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		// Setup test data
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		resp, err := svc.CreatePullRequest(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "pr-1", resp.PullRequestID)
		assert.Equal(t, "Add feature", resp.PullRequestName)
		assert.Equal(t, "u1", resp.AuthorID)
		assert.Equal(t, "OPEN", resp.Status)
		assert.Len(t, resp.AssignedReviewers, 2)
		assert.Contains(t, resp.AssignedReviewers, "u2")
		assert.Contains(t, resp.AssignedReviewers, "u3")
		assert.NotContains(t, resp.AssignedReviewers, "u1") // Author should not be reviewer
	})

	t.Run("success with 1 reviewer", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		resp, err := svc.CreatePullRequest(ctx, req)

		require.NoError(t, err)
		assert.Len(t, resp.AssignedReviewers, 1)
		assert.Equal(t, "u2", resp.AssignedReviewers[0])
	})

	t.Run("success without reviewers", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		resp, err := svc.CreatePullRequest(ctx, req)

		require.NoError(t, err)
		assert.Empty(t, resp.AssignedReviewers)
	})

	t.Run("duplicate pull request", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Existing PR", "u1", "OPEN")

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "New PR",
			AuthorID:        "u1",
		}

		resp, err := svc.CreatePullRequest(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestExists)
	})

	t.Run("author not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := repository.New(db)
		svc := New(repo, db)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "nonexistent",
		}

		resp, err := svc.CreatePullRequest(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrAuthorNotFound)
	})
}

func TestService_MergePullRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN")

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		resp, err := svc.MergePullRequest(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "MERGED", resp.Status)
		assert.NotEmpty(t, resp.MergedAt)
	})

	t.Run("idempotent merge", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		mergedAt := time.Now()
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) VALUES (?, ?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "MERGED", mergedAt)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		resp, err := svc.MergePullRequest(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "MERGED", resp.Status)
		assert.NotEmpty(t, resp.MergedAt)
	})

	t.Run("pull request not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := repository.New(db)
		svc := New(repo, db)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "nonexistent",
		}

		resp, err := svc.MergePullRequest(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestNotFound)
	})
}

func TestService_ReassignReviewer(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		resp, err := svc.ReassignReviewer(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "u3", resp.ReplacedBy)
		assert.Contains(t, resp.PR.AssignedReviewers, "u3")
		assert.NotContains(t, resp.PR.AssignedReviewers, "u2")
	})

	t.Run("pull request merged", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		mergedAt := time.Now()
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) VALUES (?, ?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "MERGED", mergedAt)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		resp, err := svc.ReassignReviewer(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestMerged)
	})

	t.Run("reviewer not assigned", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN")

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		resp, err := svc.ReassignReviewer(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrReviewerNotAssigned)
	})

	t.Run("no candidate", func(t *testing.T) {
		db := setupTestDB(t)

		type Team struct {
			TeamName  string    `gorm:"primaryKey;column:team_name"`
			CreatedAt time.Time `gorm:"column:created_at"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		type User struct {
			UserID    string    `gorm:"primaryKey;column:user_id"`
			Username  string    `gorm:"column:username"`
			TeamName  string    `gorm:"column:team_name"`
			IsActive  bool      `gorm:"column:is_active;not null"`
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
			ID            int64     `gorm:"primaryKey;column:id"`
			PullRequestID string    `gorm:"column:pull_request_id;not null"`
			UserID        string    `gorm:"column:user_id;not null"`
			AssignedAt    time.Time `gorm:"column:assigned_at"`
		}
		err := db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
		require.NoError(t, err)

		repo := repository.New(db)
		svc := New(repo, db)

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
			"pr-1", "Add feature", "u1", "OPEN")
		db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u2")

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		resp, err := svc.ReassignReviewer(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrNoCandidate)
	})

	t.Run("pull request not found", func(t *testing.T) {
		db := setupTestDB(t)
		repo := repository.New(db)
		svc := New(repo, db)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "nonexistent",
			OldUserID:     "u2",
		}

		resp, err := svc.ReassignReviewer(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, pullrequestModel.ErrPullRequestNotFound)
	})
}

