// Package health provides health check endpoint handler.
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/database/database"
)

// Handler handles health check requests.
type Handler struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new health handler instance.
func New(db *gorm.DB, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

// Response represents health check response.
type Response struct {
	Status string `json:"status"`
}

// Check handles GET /health request.
func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Check database connection
	if err := database.HealthCheck(ctx, h.db); err != nil {
		h.logger.Warnw("health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, Response{
			Status: "unhealthy",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status: "ok",
	})
}
