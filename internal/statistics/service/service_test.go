package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/statistics/model"
)

// mockRepository is a mock implementation of repository.Repository for unit tests.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetReviewersStatistics(ctx context.Context) ([]model.ReviewerStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ReviewerStatistics), args.Error(1)
}

func (m *mockRepository) GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PullRequestStatistics), args.Error(1)
}

func TestService_GetReviewersStatistics(t *testing.T) {
	ctx := context.Background()

	t.Run("success with reviewers", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		expectedReviewers := []model.ReviewerStatistics{
			{
				UserID:          "u1",
				Username:        "Alice",
				TeamName:        "backend",
				AssignmentCount: 5,
				IsActive:        true,
			},
			{
				UserID:          "u2",
				Username:        "Bob",
				TeamName:        "backend",
				AssignmentCount: 3,
				IsActive:        true,
			},
		}

		mockRepo.On("GetReviewersStatistics", ctx).Return(expectedReviewers, nil)

		resp, err := svc.GetReviewersStatistics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, resp.Total)
		require.Len(t, resp.Reviewers, 2)
		assert.Equal(t, "u1", resp.Reviewers[0].UserID)
		assert.Equal(t, 5, resp.Reviewers[0].AssignmentCount)
		assert.Equal(t, "u2", resp.Reviewers[1].UserID)
		assert.Equal(t, 3, resp.Reviewers[1].AssignmentCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success empty list", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		mockRepo.On("GetReviewersStatistics", ctx).Return([]model.ReviewerStatistics{}, nil)

		resp, err := svc.GetReviewersStatistics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Reviewers)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success with nil list", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		mockRepo.On("GetReviewersStatistics", ctx).Return(nil, nil)

		resp, err := svc.GetReviewersStatistics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Reviewers)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		repoErr := errors.New("database error")
		mockRepo.On("GetReviewersStatistics", ctx).Return(nil, repoErr)

		resp, err := svc.GetReviewersStatistics(ctx)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestService_GetPullRequestStatistics(t *testing.T) {
	ctx := context.Background()

	t.Run("success with statistics", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		expectedStats := &model.PullRequestStatistics{
			TotalPRs:              10,
			OpenPRs:               7,
			MergedPRs:             3,
			AverageReviewersPerPR: 1.5,
			PRsWith0Reviewers:     2,
			PRsWith1Reviewer:      3,
			PRsWith2Reviewers:     5,
		}

		mockRepo.On("GetPullRequestStatistics", ctx).Return(expectedStats, nil)

		resp, err := svc.GetPullRequestStatistics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 10, resp.Statistics.TotalPRs)
		assert.Equal(t, 7, resp.Statistics.OpenPRs)
		assert.Equal(t, 3, resp.Statistics.MergedPRs)
		assert.Equal(t, 1.5, resp.Statistics.AverageReviewersPerPR)
		assert.Equal(t, 2, resp.Statistics.PRsWith0Reviewers)
		assert.Equal(t, 3, resp.Statistics.PRsWith1Reviewer)
		assert.Equal(t, 5, resp.Statistics.PRsWith2Reviewers)
		mockRepo.AssertExpectations(t)
	})

	t.Run("success with zero statistics", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		expectedStats := &model.PullRequestStatistics{
			TotalPRs:              0,
			OpenPRs:               0,
			MergedPRs:             0,
			AverageReviewersPerPR: 0,
			PRsWith0Reviewers:     0,
			PRsWith1Reviewer:      0,
			PRsWith2Reviewers:     0,
		}

		mockRepo.On("GetPullRequestStatistics", ctx).Return(expectedStats, nil)

		resp, err := svc.GetPullRequestStatistics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Statistics.TotalPRs)
		assert.Equal(t, 0, resp.Statistics.OpenPRs)
		assert.Equal(t, 0, resp.Statistics.MergedPRs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockRepository)
		svc := New(mockRepo, zap.NewNop().Sugar())

		repoErr := errors.New("database error")
		mockRepo.On("GetPullRequestStatistics", ctx).Return(nil, repoErr)

		resp, err := svc.GetPullRequestStatistics(ctx)

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, repoErr)
		mockRepo.AssertExpectations(t)
	})
}
