package handler

import (
	"bytes"
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

	teamModel "github.com/festy23/avito_internship/internal/team/model"
	"github.com/festy23/avito_internship/internal/team/service"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) AddTeam(ctx context.Context, req *teamModel.AddTeamRequest) (*teamModel.TeamResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*teamModel.TeamResponse), args.Error(1)
}

func (m *mockService) GetTeam(ctx context.Context, teamName string) (*teamModel.TeamResponse, error) {
	args := m.Called(ctx, teamName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*teamModel.TeamResponse), args.Error(1)
}

var _ service.Service = (*mockService)(nil)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandler_AddTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}
		resp := &teamModel.TeamResponse{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		mockSvc.On("AddTeam", mock.Anything, req).Return(resp, nil)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "backend", response["team"].TeamName)
		assert.Len(t, response["team"].Members, 2)
		mockSvc.AssertExpectations(t)
	})

	t.Run("duplicate team", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		mockSvc.On("AddTeam", mock.Anything, req).Return(nil, teamModel.ErrTeamExists)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "TEAM_EXISTS", response.Error.Code)
		assert.Equal(t, "team_name already exists", response.Error.Message)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer([]byte("invalid json")))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})

	t.Run("empty team name", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		req := &teamModel.AddTeamRequest{
			TeamName: "",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		// Don't set up mock expectation because Gin binding will reject this before service is called

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})

	t.Run("empty members", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members:  []teamModel.TeamMember{},
		}

		mockSvc.On("AddTeam", mock.Anything, req).Return(nil, teamModel.ErrEmptyMembers)

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.POST("/team/add", handler.AddTeam)

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		mockSvc.On("AddTeam", mock.Anything, req).Return(nil, errors.New("database error"))

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestHandler_GetTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/team/get", handler.GetTeam)

		resp := &teamModel.TeamResponse{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		mockSvc.On("GetTeam", mock.Anything, "backend").Return(resp, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		var response teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "backend", response.TeamName)
		assert.Len(t, response.Members, 2)
		mockSvc.AssertExpectations(t)
	})

	t.Run("team not found", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/team/get", handler.GetTeam)

		mockSvc.On("GetTeam", mock.Anything, "nonexistent").Return(nil, teamModel.ErrTeamNotFound)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=nonexistent", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", response.Error.Code)
		assert.Equal(t, "team not found", response.Error.Message)
		mockSvc.AssertExpectations(t)
	})

	t.Run("missing team_name parameter", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/team/get", handler.GetTeam)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/team/get", handler.GetTeam)

		mockSvc.On("GetTeam", mock.Anything, "backend").Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("empty team (no members)", func(t *testing.T) {
		mockSvc := new(mockService)
		handler := New(mockSvc)
		router := setupRouter()
		router.GET("/team/get", handler.GetTeam)

		resp := &teamModel.TeamResponse{
			TeamName: "backend",
			Members:  []teamModel.TeamMember{},
		}

		mockSvc.On("GetTeam", mock.Anything, "backend").Return(resp, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		var response teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "backend", response.TeamName)
		assert.Empty(t, response.Members)
		mockSvc.AssertExpectations(t)
	})
}

