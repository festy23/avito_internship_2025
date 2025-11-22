package router

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
)

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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&testTeam{}, &testUser{})
	require.NoError(t, err)

	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, db)
	return r
}

func TestIntegration_AddTeam(t *testing.T) {
	t.Run("success - create team with multiple members", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: false},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
		}

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
		require.Len(t, response["team"].Members, 3)
		assert.Equal(t, "u1", response["team"].Members[0].UserID)
		assert.Equal(t, "Alice", response["team"].Members[0].Username)
		assert.True(t, response["team"].Members[0].IsActive)
		assert.Equal(t, "u2", response["team"].Members[1].UserID)
		assert.Equal(t, "Bob", response["team"].Members[1].Username)
		assert.False(t, response["team"].Members[1].IsActive)

		// Verify in database
		var dbTeam testTeam
		db.Where("team_name = ?", "backend").First(&dbTeam)
		assert.Equal(t, "backend", dbTeam.TeamName)

		var dbUsers []testUser
		db.Where("team_name = ?", "backend").Order("user_id ASC").Find(&dbUsers)
		require.Len(t, dbUsers, 3)
		assert.Equal(t, "u1", dbUsers[0].UserID)
		assert.Equal(t, "Alice", dbUsers[0].Username)
	})

	t.Run("duplicate team returns error", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Pre-create team
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		req := &teamModel.AddTeamRequest{
			TeamName: "backend",
			Members: []teamModel.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "TEAM_EXISTS", response["error"]["code"])
	})

	t.Run("invalid request - missing team_name", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		req := map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": "u1", "username": "Alice", "is_active": true},
			},
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid request - missing members", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		req := map[string]interface{}{
			"team_name": "backend",
		}

		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIntegration_GetTeam(t *testing.T) {
	t.Run("success - get team with members", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Setup data
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u1", "Alice", "backend", true)
		db.Exec("INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			"u2", "Bob", "backend", false)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "backend", response.TeamName)
		require.Len(t, response.Members, 2)
		assert.Equal(t, "u1", response.Members[0].UserID)
		assert.Equal(t, "Alice", response.Members[0].Username)
		assert.True(t, response.Members[0].IsActive)
		assert.Equal(t, "u2", response.Members[1].UserID)
		assert.Equal(t, "Bob", response.Members[1].Username)
		assert.False(t, response.Members[1].IsActive)
	})

	t.Run("success - get team with no members", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Setup data
		db.Exec("INSERT INTO teams (team_name) VALUES (?)", "backend")

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=backend", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "backend", response.TeamName)
		assert.Empty(t, response.Members)
	})

	t.Run("team not found", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get?team_name=nonexistent", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NOT_FOUND", response["error"]["code"])
	})

	t.Run("missing team_name parameter", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/team/get", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", response["error"]["code"])
	})
}

func TestIntegration_FullFlow(t *testing.T) {
	t.Run("create team then get team", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Create team
		createReq := &teamModel.AddTeamRequest{
			TeamName: "payments",
			Members: []teamModel.TeamMember{
				{UserID: "p1", Username: "Peter", IsActive: true},
				{UserID: "p2", Username: "Paul", IsActive: true},
			},
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Get team
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("GET", "/team/get?team_name=payments", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response teamModel.TeamResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "payments", response.TeamName)
		require.Len(t, response.Members, 2)
		assert.Equal(t, "p1", response.Members[0].UserID)
		assert.Equal(t, "Peter", response.Members[0].Username)
	})

	t.Run("update user info via team add", func(t *testing.T) {
		db := setupIntegrationDB(t)
		router := setupRouter(db)

		// Create initial team
		createReq := &teamModel.AddTeamRequest{
			TeamName: "devops",
			Members: []teamModel.TeamMember{
				{UserID: "d1", Username: "Dave", IsActive: true},
			},
		}

		body, _ := json.Marshal(createReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/team/add", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Note: Team add should fail for duplicate team, so we can't test user update this way
		// User updates would come through a different endpoint (users/setIsActive)
	})
}

