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

func setupRecoveryRouter(logger *zap.SugaredLogger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery(logger))
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	return r
}

func TestRecovery_Middleware(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	router := setupRecoveryRouter(logger)

	t.Run("recovers from panic", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)

		// Should not panic, middleware should recover
		assert.NotPanics(t, func() {
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	})

	t.Run("normal request works", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})
}

func TestRecovery_ReturnsCorrectErrorFormat(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	router := setupRecoveryRouter(logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	assert.Contains(t, w.Body.String(), "code")
	assert.Contains(t, w.Body.String(), "message")
}

func TestRecovery_AbortsRequest(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	router := setupRecoveryRouter(logger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	// After panic, request should be aborted and not continue processing
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
