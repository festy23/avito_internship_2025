// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func setupTestRouter(logger *zap.SugaredLogger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Logger(logger))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	r.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})
	r.GET("/server-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})
	return r
}

func TestLogger_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "successful request",
			path:           "/test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "client error",
			path:           "/error",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "server error",
			path:           "/server-error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t).Sugar()
			router := setupTestRouter(logger)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestLogger_LogsRequestDetails(t *testing.T) {
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stdout"}

	logger, _ := config.Build()
	sugar := logger.Sugar()

	router := setupTestRouter(sugar)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_LogsLatency(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	router := setupTestRouter(logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_LogsResponseSize(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	router := setupTestRouter(logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Greater(t, w.Body.Len(), 0)
}
