package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create minimal tables for router test
	err = db.Exec(`
		CREATE TABLE users (
			user_id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE pull_requests (
			pull_request_id VARCHAR(255) PRIMARY KEY,
			pull_request_name VARCHAR(255) NOT NULL,
			author_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'OPEN'
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE pull_request_reviewers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestRegisterRoutes(t *testing.T) {
	t.Run("registers reviewers statistics route", func(t *testing.T) {
		db := setupTestDB(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := zap.NewNop().Sugar()

		RegisterRoutes(router, db, logger)

		req := httptest.NewRequest(http.MethodGet, "/statistics/reviewers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Route should exist and return 200 (even if empty)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("registers pull requests statistics route", func(t *testing.T) {
		db := setupTestDB(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := zap.NewNop().Sugar()

		RegisterRoutes(router, db, logger)

		req := httptest.NewRequest(http.MethodGet, "/statistics/pullrequests", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Route should exist and return 200 (even if empty)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("routes are accessible after registration", func(t *testing.T) {
		db := setupTestDB(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := zap.NewNop().Sugar()

		RegisterRoutes(router, db, logger)

		// Test reviewers route
		req1 := httptest.NewRequest(http.MethodGet, "/statistics/reviewers", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Test pull requests route
		req2 := httptest.NewRequest(http.MethodGet, "/statistics/pullrequests", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("non-existent route returns 404", func(t *testing.T) {
		db := setupTestDB(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := zap.NewNop().Sugar()

		RegisterRoutes(router, db, logger)

		req := httptest.NewRequest(http.MethodGet, "/statistics/nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("POST method not allowed on GET routes", func(t *testing.T) {
		db := setupTestDB(t)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		logger := zap.NewNop().Sugar()

		RegisterRoutes(router, db, logger)

		req := httptest.NewRequest(http.MethodPost, "/statistics/reviewers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Gin returns 404 for method not allowed
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
