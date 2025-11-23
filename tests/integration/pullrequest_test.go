//go:build integration
// +build integration

package integration

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
	pullrequestRouter "github.com/festy23/avito_internship/internal/pullrequest/router"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
	teamRouter "github.com/festy23/avito_internship/internal/team/router"
)

type prTestTeam struct {
	TeamName  string    `gorm:"primaryKey;column:team_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (prTestTeam) TableName() string {
	return "teams"
}

type prTestUser struct {
	UserID    string    `gorm:"primaryKey;column:user_id"`
	Username  string    `gorm:"column:username;not null"`
	TeamName  string    `gorm:"column:team_name;not null"`
	IsActive  bool      `gorm:"column:is_active;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (prTestUser) TableName() string {
	return "users"
}

type prTestPullRequest struct {
	PullRequestID   string     `gorm:"primaryKey;column:pull_request_id"`
	PullRequestName string     `gorm:"column:pull_request_name;not null"`
	AuthorID        string     `gorm:"column:author_id;not null"`
	Status          string     `gorm:"column:status;not null"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	MergedAt        *time.Time `gorm:"column:merged_at"`
}

func (prTestPullRequest) TableName() string {
	return "pull_requests"
}

type prTestPullRequestReviewer struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	PullRequestID string    `gorm:"column:pull_request_id;not null"`
	UserID        string    `gorm:"column:user_id;not null"`
	AssignedAt    time.Time `gorm:"column:assigned_at"`
}

func (prTestPullRequestReviewer) TableName() string {
	return "pull_request_reviewers"
}

func setupDB(t *testing.T) *gorm.DB {
	dbName := ":memory:"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	err = db.AutoMigrate(&prTestTeam{}, &prTestUser{}, &prTestPullRequest{}, &prTestPullRequestReviewer{})
	require.NoError(t, err)

	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	teamRouter.RegisterRoutes(r, db, zap.NewNop().Sugar())
	pullrequestRouter.RegisterRoutes(r, db, zap.NewNop().Sugar())
	return r
}

func TestPullRequestLifecycle(t *testing.T) {
	t.Run("create team then create PR with automatic reviewer assignment", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Step 1: Create team
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Step 2: Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add search feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var prResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &prResponse)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", prResponse["pr"].PullRequestID)
		assert.Equal(t, "OPEN", prResponse["pr"].Status)
		assert.Len(t, prResponse["pr"].AssignedReviewers, 2)
		assert.Contains(t, prResponse["pr"].AssignedReviewers, "u2")
		assert.Contains(t, prResponse["pr"].AssignedReviewers, "u3")
		assert.NotContains(t, prResponse["pr"].AssignedReviewers, "u1") // Author should not be reviewer
	})

	t.Run("create PR then merge", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Merge PR
		mergeReq := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		body, _ = json.Marshal(mergeReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var mergeResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &mergeResponse)
		require.NoError(t, err)

		assert.Equal(t, "MERGED", mergeResponse["pr"].Status)
		assert.NotEmpty(t, mergeResponse["pr"].MergedAt)
	})

	t.Run("create PR then reassign then merge", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
				{UserID: "u4", Username: "David", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		// Get one of the assigned reviewers (must have at least one)
		require.NotEmpty(t, createResponse["pr"].AssignedReviewers, "PR should have at least one reviewer assigned")
		oldReviewer := createResponse["pr"].AssignedReviewers[0]

		// Reassign reviewer
		reassignReq := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     oldReviewer,
		}

		body, _ = json.Marshal(reassignReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var reassignResponse pullrequestModel.ReassignReviewerResponse
		err = json.Unmarshal(w.Body.Bytes(), &reassignResponse)
		require.NoError(t, err)

		assert.NotEqual(t, oldReviewer, reassignResponse.ReplacedBy)
		assert.Contains(t, reassignResponse.PR.AssignedReviewers, reassignResponse.ReplacedBy)
		assert.NotContains(t, reassignResponse.PR.AssignedReviewers, oldReviewer)

		// Merge PR
		mergeReq := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		body, _ = json.Marshal(mergeReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("attempt reassign after merge - error", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Merge PR
		mergeReq := &pullrequestModel.MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		body, _ = json.Marshal(mergeReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Attempt reassign after merge
		reassignReq := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		body, _ = json.Marshal(reassignReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "PR_MERGED", errorResponse.Error.Code)
	})

	t.Run("attempt reassign non-existent reviewer - error", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team with 3 members: u1 (author), u2 (active), u3 (inactive)
		// u3 is inactive so won't be assigned, making it perfect for testing NOT_ASSIGNED
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: false}, // Inactive, won't be assigned
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR (will assign u2, but not u1 as author or u3 as inactive)
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "PR should be created successfully")

		// Attempt reassign u3 (exists in team but is not assigned as reviewer)
		// Should return 409 NOT_ASSIGNED
		reassignReq := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u3",
		}

		body, _ = json.Marshal(reassignReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "NOT_ASSIGNED", errorResponse.Error.Code)
	})

	t.Run("create PR when only author in team - 0 reviewers", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team with only author
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var prResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &prResponse)
		require.NoError(t, err)

		assert.Empty(t, prResponse["pr"].AssignedReviewers)
	})

	t.Run("create PR when 2 people including author - 1 reviewer", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team with 2 people
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var prResponse map[string]pullrequestModel.PullRequestResponse
		err := json.Unmarshal(w.Body.Bytes(), &prResponse)
		require.NoError(t, err)

		assert.Len(t, prResponse["pr"].AssignedReviewers, 1)
		assert.Equal(t, "u2", prResponse["pr"].AssignedReviewers[0])
	})

	t.Run("reassign when no candidate - error", func(t *testing.T) {
		db := setupDB(t)
		router := setupRouter(db)

		// Setup team with only 2 people (author + 1 reviewer)
		createTeamReq := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		body, _ := json.Marshal(createTeamReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code, "team should be created successfully")

		// Create PR
		createPRReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		body, _ = json.Marshal(createPRReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Attempt reassign (no other candidates in team)
		reassignReq := &pullrequestModel.ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u2",
		}

		body, _ = json.Marshal(reassignReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "NO_CANDIDATE", errorResponse.Error.Code)
	})
}

// ErrorResponse represents error response structure matching OpenAPI spec.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
