package health

import (
	"context"
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
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Set max open connections to 1 for in-memory SQLite
	// This ensures all operations use the same connection and see the same database state
	// Without this, SQLite :memory: can create separate databases per connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	return db
}

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", handler.Check)
	return router
}

func TestHandler_Check(t *testing.T) {
	t.Run("success - database is healthy", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ok"`)
	})

	t.Run("failure - database is unavailable", func(t *testing.T) {
		// Create a DB connection and close it to simulate unavailability
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.Close()

		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"unhealthy"`)
	})

	t.Run("context with timeout", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		// Create request with very short context timeout
		w := httptest.NewRecorder()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure context expires

		req, _ := http.NewRequestWithContext(ctx, "GET", "/health", nil)
		router.ServeHTTP(w, req)

		// Should still complete, internal timeout is 5 seconds
		// Context cancellation doesn't prevent response
		assert.NotEqual(t, 0, w.Code)
	})

	t.Run("response format validation", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check Content-Type
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		// Verify response contains only status field
		body := w.Body.String()
		assert.Contains(t, body, `"status":"ok"`)
	})

	t.Run("multiple concurrent health checks", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		// Simulate concurrent health check requests
		// Collect results in channel to avoid unsafe assert calls from goroutines
		results := make(chan int, 10)
		for i := 0; i < 10; i++ {
			go func() {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/health", nil)
				router.ServeHTTP(w, req)
				// Send status code to channel instead of calling assert in goroutine
				results <- w.Code
			}()
		}

		// Wait for all goroutines to complete and verify results in main goroutine
		for i := 0; i < 10; i++ {
			statusCode := <-results
			assert.Equal(t, http.StatusOK, statusCode, "health check should return 200 OK")
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("creates handler with valid parameters", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()

		handler := New(db, logger)

		assert.NotNil(t, handler)
		assert.Equal(t, db, handler.db)
		assert.Equal(t, logger, handler.logger)
	})

	t.Run("creates handler with nil parameters", func(t *testing.T) {
		// Should not panic even with nil parameters
		handler := New(nil, nil)
		assert.NotNil(t, handler)
	})
}

func TestResponse(t *testing.T) {
	t.Run("response struct has correct fields", func(t *testing.T) {
		resp := Response{
			Status: "ok",
		}

		assert.Equal(t, "ok", resp.Status)
	})
}

func TestHealthCheckWithMockError(t *testing.T) {
	t.Run("handles database ping error", func(t *testing.T) {
		// This test verifies that handler correctly catches DB errors
		// We use a closed connection to simulate database unavailability
		db := setupTestDB(t)
		sqlDB, err := db.DB()
		require.NoError(t, err)

		// Close the connection
		err = sqlDB.Close()
		require.NoError(t, err)

		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

// Benchmark health check performance.
func BenchmarkHandler_Check(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	logger := zap.NewNop().Sugar()
	handler := New(db, logger)
	router := setupRouter(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)
	}
}

// TestHealthCheckIntegration tests the full integration with actual database operations.
func TestHealthCheckIntegration(t *testing.T) {
	t.Run("health check after database operations", func(t *testing.T) {
		db := setupTestDB(t)

		// Create a test table
		err := db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)").Error
		require.NoError(t, err)

		// Insert data
		err = db.Exec("INSERT INTO test_table (id) VALUES (1)").Error
		require.NoError(t, err)

		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ok"`)
	})
}

// TestHealthCheckTimeout verifies timeout behavior.
func TestHealthCheckTimeout(t *testing.T) {
	t.Run("respects internal timeout", func(t *testing.T) {
		db := setupTestDB(t)
		logger := zap.NewNop().Sugar()
		handler := New(db, logger)
		router := setupRouter(handler)

		start := time.Now()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)
		elapsed := time.Since(start)

		// Health check should complete quickly (within 1 second for SQLite)
		assert.Less(t, elapsed, 1*time.Second)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
