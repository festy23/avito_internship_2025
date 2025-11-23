package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/statistics/model"
	"github.com/festy23/avito_internship/internal/statistics/service"
)

// mockService is a mock implementation of service.Service for unit tests.
type mockService struct {
	mock.Mock
}

func (m *mockService) GetReviewersStatistics(ctx context.Context) (*model.ReviewersStatisticsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ReviewersStatisticsResponse), args.Error(1)
}

func (m *mockService) GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatisticsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PullRequestStatisticsResponse), args.Error(1)
}

var _ service.Service = (*mockService)(nil)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandler_GetReviewersStatistics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/reviewers", handler.GetReviewersStatistics)

		expectedResp := &model.ReviewersStatisticsResponse{
			Reviewers: []model.ReviewerStatistics{
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
			},
			Total: 2,
		}

		mockSvc.On("GetReviewersStatistics", mock.Anything).Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/statistics/reviewers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.ReviewersStatisticsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		require.Len(t, resp.Reviewers, 2)
		assert.Equal(t, "u1", resp.Reviewers[0].UserID)
		assert.Equal(t, 5, resp.Reviewers[0].AssignmentCount)
		mockSvc.AssertExpectations(t)
	})

	t.Run("success empty list", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/reviewers", handler.GetReviewersStatistics)

		expectedResp := &model.ReviewersStatisticsResponse{
			Reviewers: []model.ReviewerStatistics{},
			Total:     0,
		}

		mockSvc.On("GetReviewersStatistics", mock.Anything).Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/statistics/reviewers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.ReviewersStatisticsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Reviewers)
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/reviewers", handler.GetReviewersStatistics)

		svcErr := errors.New("database error")
		mockSvc.On("GetReviewersStatistics", mock.Anything).Return(nil, svcErr)

		req := httptest.NewRequest(http.MethodGet, "/statistics/reviewers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_ERROR", errorResp.Error.Code)
		assert.Equal(t, "internal server error", errorResp.Error.Message)
		mockSvc.AssertExpectations(t)
	})
}

func TestHandler_GetPullRequestStatistics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/pullrequests", handler.GetPullRequestStatistics)

		expectedResp := &model.PullRequestStatisticsResponse{
			Statistics: model.PullRequestStatistics{
				TotalPRs:              10,
				OpenPRs:               7,
				MergedPRs:             3,
				AverageReviewersPerPR: 1.5,
				PRsWith0Reviewers:     2,
				PRsWith1Reviewer:      3,
				PRsWith2Reviewers:     5,
			},
		}

		mockSvc.On("GetPullRequestStatistics", mock.Anything).Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/statistics/pullrequests", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.PullRequestStatisticsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 10, resp.Statistics.TotalPRs)
		assert.Equal(t, 7, resp.Statistics.OpenPRs)
		assert.Equal(t, 3, resp.Statistics.MergedPRs)
		assert.Equal(t, 1.5, resp.Statistics.AverageReviewersPerPR)
		mockSvc.AssertExpectations(t)
	})

	t.Run("success with zero statistics", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/pullrequests", handler.GetPullRequestStatistics)

		expectedResp := &model.PullRequestStatisticsResponse{
			Statistics: model.PullRequestStatistics{
				TotalPRs:              0,
				OpenPRs:               0,
				MergedPRs:             0,
				AverageReviewersPerPR: 0,
				PRsWith0Reviewers:     0,
				PRsWith1Reviewer:      0,
				PRsWith2Reviewers:     0,
			},
		}

		mockSvc.On("GetPullRequestStatistics", mock.Anything).Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/statistics/pullrequests", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.PullRequestStatisticsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Statistics.TotalPRs)
		assert.Equal(t, 0, resp.Statistics.OpenPRs)
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.GET("/statistics/pullrequests", handler.GetPullRequestStatistics)

		svcErr := errors.New("database error")
		mockSvc.On("GetPullRequestStatistics", mock.Anything).Return(nil, svcErr)

		req := httptest.NewRequest(http.MethodGet, "/statistics/pullrequests", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_ERROR", errorResp.Error.Code)
		assert.Equal(t, "internal server error", errorResp.Error.Message)
		mockSvc.AssertExpectations(t)
	})
}
