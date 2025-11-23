package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	"github.com/festy23/avito_internship/internal/pullrequest/service"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) CreatePullRequest(
	ctx context.Context,
	req *pullrequestModel.CreatePullRequestRequest,
) (*pullrequestModel.PullRequestResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pullrequestModel.PullRequestResponse), args.Error(1)
}

func (m *mockService) MergePullRequest(
	ctx context.Context,
	req *pullrequestModel.MergePullRequestRequest,
) (*pullrequestModel.PullRequestResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pullrequestModel.PullRequestResponse), args.Error(1)
}

func (m *mockService) ReassignReviewer(
	ctx context.Context,
	req *pullrequestModel.ReassignReviewerRequest,
) (*pullrequestModel.ReassignReviewerResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pullrequestModel.ReassignReviewerResponse), args.Error(1)
}

var _ service.Service = (*mockService)(nil)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandler_CreatePullRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/create", handler.CreatePullRequest)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}
		resp := &pullrequestModel.PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Add feature",
			AuthorID:          "u1",
			Status:            "OPEN",
			AssignedReviewers: []string{"u2", "u3"},
		}

		mockSvc.On("CreatePullRequest", mock.Anything, req).Return(resp, nil)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "pr-1", response["pr"].PullRequestID)
		assert.Len(t, response["pr"].AssignedReviewers, 2)
		mockSvc.AssertExpectations(t)
	})

	t.Run("duplicate pull request", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/create", handler.CreatePullRequest)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		mockSvc.On("CreatePullRequest", mock.Anything, req).
			Return(nil, pullrequestModel.ErrPullRequestExists)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "PR_EXISTS", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("author not found", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/create", handler.CreatePullRequest)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "nonexistent",
		}

		mockSvc.On("CreatePullRequest", mock.Anything, req).
			Return(nil, pullrequestModel.ErrAuthorNotFound)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/create", handler.CreatePullRequest)

		body := []byte(`{"invalid": "json"`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})
}

func TestHandler_MergePullRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/merge", handler.MergePullRequest)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}
		resp := &pullrequestModel.PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Add feature",
			AuthorID:          "u1",
			Status:            "MERGED",
			AssignedReviewers: []string{"u2", "u3"},
			MergedAt:          "2025-10-24T12:34:56Z",
		}

		mockSvc.On("MergePullRequest", mock.Anything, req).Return(resp, nil)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "MERGED", response["pr"].Status)
		assert.NotEmpty(t, response["pr"].MergedAt)
		mockSvc.AssertExpectations(t)
	})

	t.Run("pull request not found", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/merge", handler.MergePullRequest)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "nonexistent",
		}

		mockSvc.On("MergePullRequest", mock.Anything, req).
			Return(nil, pullrequestModel.ErrPullRequestNotFound)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/merge", handler.MergePullRequest)

		body := []byte(`{"invalid": "json"`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})
}

func TestHandler_ReassignReviewer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/reassign", handler.ReassignReviewer)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}
		resp := &pullrequestModel.ReassignReviewerResponse{
			PR: &pullrequestModel.PullRequestResponse{
				PullRequestID:     "pr-1",
				PullRequestName:   "Add feature",
				AuthorID:          "u1",
				Status:            "OPEN",
				AssignedReviewers: []string{"u3", "u5"},
			},
			ReplacedBy: "u5",
		}

		mockSvc.On("ReassignReviewer", mock.Anything, req).Return(resp, nil)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		var response pullrequestModel.ReassignReviewerResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "u5", response.ReplacedBy)
		assert.Contains(t, response.PR.AssignedReviewers, "u3")
		assert.Contains(t, response.PR.AssignedReviewers, "u5")
		mockSvc.AssertExpectations(t)
	})

	t.Run("pull request merged", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/reassign", handler.ReassignReviewer)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		mockSvc.On("ReassignReviewer", mock.Anything, req).
			Return(nil, pullrequestModel.ErrPullRequestMerged)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "PR_MERGED", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("reviewer not assigned", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/reassign", handler.ReassignReviewer)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		mockSvc.On("ReassignReviewer", mock.Anything, req).
			Return(nil, pullrequestModel.ErrReviewerNotAssigned)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NOT_ASSIGNED", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("no candidate", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/reassign", handler.ReassignReviewer)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		mockSvc.On("ReassignReviewer", mock.Anything, req).
			Return(nil, pullrequestModel.ErrNoCandidate)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NO_CANDIDATE", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc, zap.NewNop().Sugar())
		router := setupRouter()
		router.POST("/pullRequest/reassign", handler.ReassignReviewer)

		body := []byte(`{"invalid": "json"`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})
}
