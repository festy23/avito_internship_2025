package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/user/model"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockRepository) UpdateIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	args := m.Called(ctx, userID, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockRepository) GetAssignedPullRequests(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.PullRequestShort), args.Error(1)
}

func TestService_SetIsActive(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		isActive := false
		req := &model.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: &isActive,
		}

		expectedUser := &model.User{
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

		isActive := false
		req := &model.SetIsActiveRequest{
			UserID:   "nonexistent",
			IsActive: &isActive,
		}

		mockRepo.On("UpdateIsActive", ctx, "nonexistent", false).Return(nil, model.ErrUserNotFound)

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty user_id", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		isActive := false
		req := &model.SetIsActiveRequest{
			UserID:   "",
			IsActive: &isActive,
		}

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		mockRepo.AssertNotCalled(t, "UpdateIsActive")
	})

	t.Run("missing is_active", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		req := &model.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: nil,
		}

		resp, err := svc.SetIsActive(ctx, req)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, model.ErrInvalidIsActive)
		mockRepo.AssertNotCalled(t, "UpdateIsActive")
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		isActive := false
		req := &model.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: &isActive,
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

		expectedPRs := []model.PullRequestShort{
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

		mockRepo.On("GetAssignedPullRequests", ctx, "u1").Return([]model.PullRequestShort{}, nil)

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
		assert.ErrorIs(t, err, model.ErrUserNotFound)
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
