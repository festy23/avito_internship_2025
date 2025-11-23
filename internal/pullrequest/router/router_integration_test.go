package router

import (
	"bytes"
	"database/sql"
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

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
)

// ErrorResponse represents error response structure matching OpenAPI spec.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

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
	IsActive  bool      `gorm:"column:is_active;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (testUser) TableName() string {
	return "users"
}

func setupIntegrationDB(t *testing.T) *gorm.DB {
	// Use unique in-memory DB for each test to ensure isolation
	// Each call to Open(":memory:") creates a new in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Disable GORM logging
	})
	require.NoError(t, err)

	// Limit connection pool to 1 to ensure in-memory DB works correctly
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(&testTeam{}, &testUser{}, &testPullRequest{}, &testPullRequestReviewer{})
	require.NoError(t, err)

	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, db, zap.NewNop().Sugar())
	return r
}

func TestIntegration_CreatePullRequest(t *testing.T) {
	t.Run("success - create PR with automatic reviewer assignment", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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
		assert.Equal(t, "Add feature", response["pr"].PullRequestName)
		assert.Equal(t, "u1", response["pr"].AuthorID)
		assert.Equal(t, "OPEN", response["pr"].Status)
		assert.Len(t, response["pr"].AssignedReviewers, 2)
		assert.Contains(t, response["pr"].AssignedReviewers, "u2")
		assert.Contains(t, response["pr"].AssignedReviewers, "u3")
		assert.NotContains(
			t,
			response["pr"].AssignedReviewers,
			"u1",
		) // Author should not be reviewer
	})

	t.Run("duplicate pull request", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "New PR",
			AuthorID:        "u1",
		}

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
	})

	t.Run("author not found", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		req := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "nonexistent",
		}

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
	})

	t.Run("invalid request - missing fields", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		body := []byte(`{"pull_request_id": "pr-1"}`)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIntegration_MergePullRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

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
	})

	t.Run("idempotent merge", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		mergedAt := time.Now()
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) VALUES (?, ?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"MERGED",
			mergedAt,
		)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

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
	})

	t.Run("pull request not found", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		req := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "nonexistent",
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestIntegration_ReassignReviewer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response pullrequestModel.ReassignReviewerResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "u3", response.ReplacedBy)
		assert.Contains(t, response.PR.AssignedReviewers, "u3")
		assert.NotContains(t, response.PR.AssignedReviewers, "u2")
	})

	t.Run("pull request merged", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		mergedAt := time.Now()
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec(
			"INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, merged_at) VALUES (?, ?, ?, ?, ?)",
			"pr-1",
			"Add feature",
			"u1",
			"MERGED",
			mergedAt,
		)

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

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
	})

	t.Run("reviewer not assigned", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

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
	})

	t.Run("no candidate", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

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

		req := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

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
	})
}

func TestIntegration_FullFlow(t *testing.T) {
	t.Run("create PR then merge", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Setup test data
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)

		// Create PR
		createReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Merge PR
		mergeReq := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		body, _ = json.Marshal(mergeReq)
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "MERGED", response["pr"].Status)
	})

	t.Run("create PR then reassign then merge", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Setup test data
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u3", "Charlie", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u4", "David", "backend", true)

		// Create PR
		createReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Get created PR to find assigned reviewers
		var createResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		require.NoError(t, err)
		require.NotEmpty(t, createResponse["pr"].AssignedReviewers, "PR should have at least one reviewer assigned")
		oldReviewerID := createResponse["pr"].AssignedReviewers[0]

		// Reassign reviewer
		reassignReq := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     oldReviewerID,
		}

		body, _ = json.Marshal(reassignReq)
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var reassignResponse pullrequestModel.ReassignReviewerResponse
		err = json.Unmarshal(w.Body.Bytes(), &reassignResponse)
		require.NoError(t, err)
		// ReplacedBy should be different from the old reviewer
		assert.NotEqual(t, oldReviewerID, reassignResponse.ReplacedBy)
		// ReplacedBy should be one of the available users (u3 or u4)
		assert.Contains(t, []string{"u3", "u4"}, reassignResponse.ReplacedBy)

		// Merge PR
		mergeReq := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		body, _ = json.Marshal(mergeReq)
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
