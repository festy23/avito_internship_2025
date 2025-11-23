//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/festy23/avito_internship/internal/user/model"
	userRouter "github.com/festy23/avito_internship/internal/user/router"
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

func setupE2EDB(t *testing.T) *gorm.DB {
	// Disable GORM logging in tests to reduce noise (expected "record not found" errors)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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

func TestE2E_SetUserActive(t *testing.T) {
	db := setupE2EDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userRouter.RegisterRoutes(router, db, zap.NewNop().Sugar())

	db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
	db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
		"u1", "Alice", "team1", true)

	isActive := false
	reqBody := model.SetIsActiveRequest{
		UserID:   "u1",
		IsActive: &isActive,
	}
	jsonBody, _ := json.Marshal(reqBody)

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
}

func TestE2E_SetUserInactive(t *testing.T) {
	db := setupE2EDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userRouter.RegisterRoutes(router, db, zap.NewNop().Sugar())

	db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
	db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
		"u1", "Alice", "team1", false)

	isActive := true
	reqBody := model.SetIsActiveRequest{
		UserID:   "u1",
		IsActive: &isActive,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.SetIsActiveResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.User.UserID)
	assert.True(t, resp.User.IsActive)
}

func TestE2E_GetReviewWithPRs(t *testing.T) {
	db := setupE2EDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userRouter.RegisterRoutes(router, db, zap.NewNop().Sugar())

	db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
	db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
		"u1", "Alice", "team1", true)
	db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
		"u2", "Bob", "team1", true)
	db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
		"pr-1", "PR 1", "u2", "OPEN")
	db.Exec("INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) VALUES (?, ?, ?, ?)",
		"pr-2", "PR 2", "u2", "MERGED")
	db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-1", "u1")
	db.Exec("INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES (?, ?)", "pr-2", "u1")

	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=u1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.GetReviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.UserID)
	require.Len(t, resp.PullRequests, 2)
	assert.Equal(t, "pr-1", resp.PullRequests[0].PullRequestID)
	assert.Equal(t, "OPEN", resp.PullRequests[0].Status)
	assert.Equal(t, "pr-2", resp.PullRequests[1].PullRequestID)
	assert.Equal(t, "MERGED", resp.PullRequests[1].Status)
}

func TestE2E_GetReviewWithoutPRs(t *testing.T) {
	db := setupE2EDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userRouter.RegisterRoutes(router, db, zap.NewNop().Sugar())

	db.Exec("INSERT INTO teams (team_name) VALUES (?)", "team1")
	db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
		"u1", "Alice", "team1", true)

	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=u1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.GetReviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.UserID)
	assert.Empty(t, resp.PullRequests)
}

func TestE2E_GetReviewUserNotFound(t *testing.T) {
	db := setupE2EDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userRouter.RegisterRoutes(router, db, zap.NewNop().Sugar())

	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.GetReviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "nonexistent", resp.UserID)
	assert.Empty(t, resp.PullRequests)
}
