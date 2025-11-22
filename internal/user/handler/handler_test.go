package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/service"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) SetIsActive(
	ctx context.Context,
	req *model.SetIsActiveRequest,
) (*model.SetIsActiveResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SetIsActiveResponse), args.Error(1)
}

func (m *mockService) GetReview(ctx context.Context, userID string) (*model.GetReviewResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GetReviewResponse), args.Error(1)
}

var _ service.Service = (*mockService)(nil)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandler_SetIsActive(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/users/setIsActive", handler.SetIsActive)

		isActive := false
		reqBody := model.SetIsActiveRequest{
			UserID:   "u1",
			IsActive: &isActive,
		}
		jsonBody, _ := json.Marshal(reqBody)

		expectedResp := &model.SetIsActiveResponse{
			User: model.User{
				UserID:   "u1",
				Username: "Alice",
				TeamName: "team1",
				IsActive: false,
			},
		}

		mockSvc.On("SetIsActive", mock.Anything, &reqBody).Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.SetIsActiveResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "u1", resp.User.UserID)
		assert.False(t, resp.User.IsActive)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/users/setIsActive", handler.SetIsActive)

		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
		mockSvc.AssertNotCalled(t, "SetIsActive")
	})

	t.Run("user not found", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/users/setIsActive", handler.SetIsActive)

		isActive := false
		reqBody := model.SetIsActiveRequest{
			UserID:   "nonexistent",
			IsActive: &isActive,
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockSvc.On("SetIsActive", mock.Anything, &reqBody).Return(nil, model.ErrUserNotFound)

		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", resp.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("missing is_active field", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/users/setIsActive", handler.SetIsActive)

		reqBody := map[string]interface{}{
			"user_id": "u1",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
		mockSvc.AssertNotCalled(t, "SetIsActive")
	})
}

func TestHandler_GetReview(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/users/getReview", handler.GetReview)

		expectedResp := &model.GetReviewResponse{
			UserID: "u1",
			PullRequests: []model.PullRequestShort{
				{
					PullRequestID:   "pr-1",
					PullRequestName: "PR 1",
					AuthorID:        "u2",
					Status:          "OPEN",
				},
			},
		}

		mockSvc.On("GetReview", mock.Anything, "u1").Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=u1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.GetReviewResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "u1", resp.UserID)
		require.Len(t, resp.PullRequests, 1)
		assert.Equal(t, "pr-1", resp.PullRequests[0].PullRequestID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("missing user_id parameter", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/users/getReview", handler.GetReview)

		req := httptest.NewRequest(http.MethodGet, "/users/getReview", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
		mockSvc.AssertNotCalled(t, "GetReview")
	})

	t.Run("user not found returns empty list", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/users/getReview", handler.GetReview)

		expectedResp := &model.GetReviewResponse{
			UserID:       "nonexistent",
			PullRequests: []model.PullRequestShort{},
		}

		mockSvc.On("GetReview", mock.Anything, "nonexistent").Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.GetReviewResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "nonexistent", resp.UserID)
		assert.Empty(t, resp.PullRequests)
		mockSvc.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/users/getReview", handler.GetReview)

		expectedResp := &model.GetReviewResponse{
			UserID:       "u1",
			PullRequests: []model.PullRequestShort{},
		}

		mockSvc.On("GetReview", mock.Anything, "u1").Return(expectedResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=u1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.GetReviewResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "u1", resp.UserID)
		assert.Empty(t, resp.PullRequests)
		mockSvc.AssertExpectations(t)
	})
}

func TestHandler_EdgeCases(t *testing.T) {
	t.Run("user_id with special characters", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/users/getReview", handler.GetReview)

		specialUserID := "user'; DROP TABLE users; --"

		expectedResp := &model.GetReviewResponse{
			UserID:       specialUserID,
			PullRequests: []model.PullRequestShort{},
		}

		mockSvc.On("GetReview", mock.Anything, specialUserID).Return(expectedResp, nil)

		reqURL := "/users/getReview?user_id=" + url.QueryEscape(specialUserID)
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
