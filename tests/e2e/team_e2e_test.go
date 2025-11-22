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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	teamModel "github.com/festy23/avito_internship/internal/team/model"
	teamRouter "github.com/festy23/avito_internship/internal/team/router"
)

type teamTestTeam struct {
	TeamName  string    `gorm:"primaryKey;column:team_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (teamTestTeam) TableName() string {
	return "teams"
}

type teamTestUser struct {
	UserID    string    `gorm:"primaryKey;column:user_id"`
	Username  string    `gorm:"column:username;not null"`
	TeamName  string    `gorm:"column:team_name;not null"`
	IsActive  bool      `gorm:"column:is_active;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (teamTestUser) TableName() string {
	return "users"
}

func teamSetupE2EDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

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

	err = db.AutoMigrate(&teamTestTeam{}, &teamTestUser{}, &PullRequest{}, &PullRequestReviewer{})
	require.NoError(t, err)

	return db
}

func teamSetupE2ERouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	teamRouter.RegisterRoutes(r, db)
	return r
}

func TestE2E_TeamLifecycle(t *testing.T) {
	t.Run("complete team lifecycle", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		// Step 1: Create team
		createReq := &teamModel.AddTeamRequest{
			TeamName: "engineering",
			Members: []teamModel.TeamMember{
				{UserID: "eng1", Username: "Alice", IsActive: true},
				{UserID: "eng2", Username: "Bob", IsActive: true},
				{UserID: "eng3", Username: "Charlie", IsActive: false},
			},
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResp map[string]teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)
		assert.Equal(t, "engineering", createResp["team"].TeamName)
		assert.Len(t, createResp["team"].Members, 3)

		// Step 2: Get team and verify
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/team/get?team_name=engineering", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var getResp teamModel.TeamResponse
		err = json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)
		assert.Equal(t, "engineering", getResp.TeamName)
		assert.Len(t, getResp.Members, 3)
		assert.Equal(t, "eng1", getResp.Members[0].UserID)
		assert.Equal(t, "Alice", getResp.Members[0].Username)
		assert.True(t, getResp.Members[0].IsActive)

		// Step 3: Try to create duplicate team
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp map[string]map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "TEAM_EXISTS", errResp["error"]["code"])
	})
}

func TestE2E_MultipleTeams(t *testing.T) {
	t.Run("create and retrieve multiple teams", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		// Create first team
		team1Req := &teamModel.AddTeamRequest{
			TeamName: "frontend",
			Members: []teamModel.TeamMember{
				{UserID: "fe1", Username: "Frank", IsActive: true},
				{UserID: "fe2", Username: "Grace", IsActive: true},
			},
		}

		body, _ := json.Marshal(team1Req)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Create second team
		team2Req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "be1", Username: "Henry", IsActive: true},
				{UserID: "be2", Username: "Iris", IsActive: false},
			},
		}

		body, _ = json.Marshal(team2Req)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Get first team
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/team/get?team_name=frontend", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp1 teamModel.TeamResponse
		json.Unmarshal(w.Body.Bytes(), &resp1)
		assert.Equal(t, "frontend", resp1.TeamName)
		assert.Len(t, resp1.Members, 2)

		// Get second team
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp2 teamModel.TeamResponse
		json.Unmarshal(w.Body.Bytes(), &resp2)
		assert.Equal(t, "backend", resp2.TeamName)
		assert.Len(t, resp2.Members, 2)
	})
}

func TestE2E_TeamWithSpecialCharacters(t *testing.T) {
	t.Run("team name with special characters", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		createReq := &teamModel.AddTeamRequest{
			TeamName: "team-with-dashes_and_underscores.123",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "User One", IsActive: true},
			},
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Get team
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/team/get?team_name=team-with-dashes_and_underscores.123", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp teamModel.TeamResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "team-with-dashes_and_underscores.123", resp.TeamName)
	})

	t.Run("user with unicode characters", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		createReq := &teamModel.AddTeamRequest{
			TeamName: "international",
			Members: []teamModel.TeamMember{
				{UserID: "u_тест", Username: "Тестовый пользователь", IsActive: true},
				{UserID: "u_日本", Username: "テストユーザー", IsActive: true},
			},
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Get team
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/team/get?team_name=international", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp teamModel.TeamResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "international", resp.TeamName)
		assert.Len(t, resp.Members, 2)
	})
}

func TestE2E_ErrorCases(t *testing.T) {
	t.Run("get non-existent team", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/team/get?team_name=nonexistent", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var resp map[string]map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "NOT_FOUND", resp["error"]["code"])
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty team name parameter", func(t *testing.T) {
		db := teamSetupE2EDB(t)
		router := teamSetupE2ERouter(db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/team/get?team_name=", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
