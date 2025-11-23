// Package handler provides HTTP handlers for statistics endpoints.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/statistics/service"
)

// Handler handles HTTP requests for statistics endpoints.
type Handler struct {
	service service.Service
	logger  *zap.SugaredLogger
}

// New creates a new statistics handler instance.
func New(svc service.Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{service: svc, logger: logger}
}

// GetReviewersStatistics handles GET /statistics/reviewers request.
// @Summary Get statistics for reviewers
// @Tags Statistics
// @Produce json
// @Success 200 {object} model.ReviewersStatisticsResponse
// @Failure 500 {object} ErrorResponse
// @Router /statistics/reviewers [get] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) GetReviewersStatistics(c *gin.Context) {
	resp, err := h.service.GetReviewersStatistics(c.Request.Context())
	if err != nil {
		h.logger.Errorw("error getting reviewers statistics", "error", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetPullRequestStatistics handles GET /statistics/pullrequests request.
// @Summary Get statistics for pull requests
// @Tags Statistics
// @Produce json
// @Success 200 {object} model.PullRequestStatisticsResponse
// @Failure 500 {object} ErrorResponse
// @Router /statistics/pullrequests [get] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) GetPullRequestStatistics(c *gin.Context) {
	resp, err := h.service.GetPullRequestStatistics(c.Request.Context())
	if err != nil {
		h.logger.Errorw("error getting pull request statistics", "error", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}
