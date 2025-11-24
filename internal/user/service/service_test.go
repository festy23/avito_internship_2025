package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	pullrequestRepo "github.com/festy23/avito_internship/internal/pullrequest/repository"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
	teamRepo "github.com/festy23/avito_internship/internal/team/repository"
	userModel "github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/repository"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetByID(ctx context.Context, userID string) (*userModel.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userModel.User), args.Error(1)
}

func (m *mockRepository) UpdateIsActive(ctx context.Context, userID string, isActive bool) (*userModel.User, error) {
	args := m.Called(ctx, userID, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userModel.User), args.Error(1)
}

func (m *mockRepository) GetAssignedPullRequests(ctx context.Context, userID string) ([]userModel.PullRequestShort, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]userModel.PullRequestShort), args.Error(1)
}

func (m *mockRepository) BulkDeactivateTeamMembers(ctx context.Context, teamName string) ([]string, error) {
	args := m.Called(ctx, teamName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRepository) GetTeamMemberIDs(ctx context.Context, teamName string) ([]string, error) {
	args := m.Called(ctx, teamName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestService_SetIsActive(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		req := &userModel.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: false,
		}

		expectedUser := &userModel.User{
			UserID:   "u1",
			Username: "Alice",
			TeamName: "team1",
			IsActive: false,
		}

		mockRepo.On("UpdateIsActive", ctx, "u1", false).Return(expectedUser, nil)

		resp, err := svc.SetIsActive(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.UserID, resp.User.UserID)
		assert.Equal(t, expectedUser.Username, resp.User.Username)
		assert.False(t, resp.User.IsActive)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		req := &userModel.SetIsActiveRequest{
			UserID:   "nonexistent",
			IsActive: false,
		}

		mockRepo.On("UpdateIsActive", ctx, "nonexistent", false).Return(nil, userModel.ErrUserNotFound)

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, userModel.ErrUserNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty user_id", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		req := &userModel.SetIsActiveRequest{
			UserID:   "",
			IsActive: false,
		}

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, userModel.ErrUserNotFound)
		mockRepo.AssertNotCalled(t, "UpdateIsActive")
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		req := &userModel.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: false,
		}

		repoErr := errors.New("database error")
		mockRepo.On("UpdateIsActive", ctx, "u1", false).Return(nil, repoErr)

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetReview(t *testing.T) {
	ctx := context.Background()

	t.Run("success with PRs", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		expectedPRs := []userModel.PullRequestShort{
			{
				PullRequestID:   "pr-1",
				PullRequestName: "PR 1",
				AuthorID:        "u2",
				Status:          "OPEN",
			},
			{
				PullRequestID:   "pr-2",
				PullRequestName: "PR 2",
				AuthorID:        "u2",
				Status:          "MERGED",
			},
		}

		mockRepo.On("GetAssignedPullRequests", ctx, "u1").Return(expectedPRs, nil)

		resp, err := svc.GetReview(ctx, "u1")

		require.NoError(t, err)
		assert.Equal(t, "u1", resp.UserID)
		require.Len(t, resp.PullRequests, 2)
		assert.Equal(t, "pr-1", resp.PullRequests[0].PullRequestID)
		assert.Equal(t, "OPEN", resp.PullRequests[0].Status)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success empty list", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		mockRepo.On("GetAssignedPullRequests", ctx, "u1").Return([]userModel.PullRequestShort{}, nil)

		resp, err := svc.GetReview(ctx, "u1")

		require.NoError(t, err)
		assert.Equal(t, "u1", resp.UserID)
		assert.Empty(t, resp.PullRequests)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty user_id", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		resp, err := svc.GetReview(ctx, "")

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, userModel.ErrUserNotFound)
		mockRepo.AssertNotCalled(t, "GetAssignedPullRequests")
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		repoErr := errors.New("database error")
		mockRepo.On("GetAssignedPullRequests", ctx, "u1").Return(nil, repoErr)

		resp, err := svc.GetReview(ctx, "u1")

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}

func setupTestDBForBulkDeactivate(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	type Team struct {
		TeamName string `gorm:"primaryKey;column:team_name"`
	}

	type User struct {
		UserID   string `gorm:"primaryKey;column:user_id"`
		Username string `gorm:"column:username;not null"`
		TeamName string `gorm:"column:team_name;not null"`
		IsActive bool   `gorm:"column:is_active;not null;default:true"`
	}

	type PullRequest struct {
		PullRequestID   string `gorm:"primaryKey;column:pull_request_id"`
		PullRequestName string `gorm:"column:pull_request_name;not null"`
		AuthorID        string `gorm:"column:author_id;not null"`
		Status          string `gorm:"column:status;not null"`
	}

	type PullRequestReviewer struct {
		ID            int    `gorm:"primaryKey;autoIncrement"`
		PullRequestID string `gorm:"column:pull_request_id;not null"`
		UserID        string `gorm:"column:user_id;not null"`
	}

	err = db.AutoMigrate(&Team{}, &User{}, &PullRequest{}, &PullRequestReviewer{})
	require.NoError(t, err)

	return db
}

func TestService_BulkDeactivateTeamMembers(t *testing.T) {
	ctx := context.Background()

	t.Run("empty team_name", func(t *testing.T) {
		db := setupTestDBForBulkDeactivate(t)
		userRepo := repository.New(db, zap.NewNop().Sugar())
		teamRepoInstance := teamRepo.New(db, zap.NewNop().Sugar())
		prRepo := pullrequestRepo.New(db, zap.NewNop().Sugar())
		svc := NewWithDependencies(userRepo, teamRepoInstance, prRepo, db, zap.NewNop().Sugar())

		req := &userModel.BulkDeactivateTeamRequest{
			TeamName: "",
		}

		resp, err := svc.BulkDeactivateTeamMembers(ctx, req)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team_name is required")
	})

	t.Run("team not found", func(t *testing.T) {
		db := setupTestDBForBulkDeactivate(t)
		userRepo := repository.New(db, zap.NewNop().Sugar())
		teamRepoInstance := teamRepo.New(db, zap.NewNop().Sugar())
		prRepo := pullrequestRepo.New(db, zap.NewNop().Sugar())
		svc := NewWithDependencies(userRepo, teamRepoInstance, prRepo, db, zap.NewNop().Sugar())

		req := &userModel.BulkDeactivateTeamRequest{
			TeamName: "nonexistent",
		}

		resp, err := svc.BulkDeactivateTeamMembers(ctx, req)

		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, teamModel.ErrTeamNotFound))
	})

	t.Run("success - no active members", func(t *testing.T) {
		db := setupTestDBForBulkDeactivate(t)
		userRepo := repository.New(db, zap.NewNop().Sugar())
		teamRepoInstance := teamRepo.New(db, zap.NewNop().Sugar())
		prRepo := pullrequestRepo.New(db, zap.NewNop().Sugar())
		svc := NewWithDependencies(userRepo, teamRepoInstance, prRepo, db, zap.NewNop().Sugar())

		// Setup: create team and inactive users
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", false)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", false)

		req := &userModel.BulkDeactivateTeamRequest{
			TeamName: "backend",
		}

		resp, err := svc.BulkDeactivateTeamMembers(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "backend", resp.TeamName)
		assert.Empty(t, resp.DeactivatedUsers)
		assert.Equal(t, 0, resp.DeactivatedCount)
		assert.Equal(t, 0, resp.ReassignedPRCount)
	})

	t.Run("success - deactivate members with no PRs", func(t *testing.T) {
		db := setupTestDBForBulkDeactivate(t)
		userRepo := repository.New(db, zap.NewNop().Sugar())
		teamRepoInstance := teamRepo.New(db, zap.NewNop().Sugar())
		prRepo := pullrequestRepo.New(db, zap.NewNop().Sugar())
		svc := NewWithDependencies(userRepo, teamRepoInstance, prRepo, db, zap.NewNop().Sugar())

		// Setup: create team and active users
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)

		req := &userModel.BulkDeactivateTeamRequest{
			TeamName: "backend",
		}

		resp, err := svc.BulkDeactivateTeamMembers(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "backend", resp.TeamName)
		assert.Len(t, resp.DeactivatedUsers, 2)
		assert.Contains(t, resp.DeactivatedUsers, "u1")
		assert.Contains(t, resp.DeactivatedUsers, "u2")
		assert.Equal(t, 2, resp.DeactivatedCount)
		assert.Equal(t, 0, resp.ReassignedPRCount)
		assert.Empty(t, resp.ReassignedPRs)

		// Verify users are deactivated
		var user1, user2 userModel.User
		db.Where("user_id = ?", "u1").First(&user1)
		db.Where("user_id = ?", "u2").First(&user2)
		assert.False(t, user1.IsActive)
		assert.False(t, user2.IsActive)
	})

	t.Run("success - empty team", func(t *testing.T) {
		db := setupTestDBForBulkDeactivate(t)
		userRepo := repository.New(db, zap.NewNop().Sugar())
		teamRepoInstance := teamRepo.New(db, zap.NewNop().Sugar())
		prRepo := pullrequestRepo.New(db, zap.NewNop().Sugar())
		svc := NewWithDependencies(userRepo, teamRepoInstance, prRepo, db, zap.NewNop().Sugar())

		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		req := &userModel.BulkDeactivateTeamRequest{
			TeamName: "backend",
		}

		resp, err := svc.BulkDeactivateTeamMembers(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "backend", resp.TeamName)
		assert.Empty(t, resp.DeactivatedUsers)
		assert.Equal(t, 0, resp.DeactivatedCount)
		assert.Equal(t, 0, resp.ReassignedPRCount)
	})
}
