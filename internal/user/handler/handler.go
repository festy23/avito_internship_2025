// Package handler provides HTTP handlers for user endpoints.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/service"
)

// Handler handles HTTP requests for user endpoints.
type Handler struct {
	service service.Service
	logger  *zap.SugaredLogger
}

// New creates a new user handler instance.
func New(svc service.Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{service: svc, logger: logger}
}

// SetIsActive handles POST /users/setIsActive request.
// @Summary Set user activity status
// @Tags Users
// @Accept json
// @Produce json
// @Param request body model.SetIsActiveRequest true "Request"
// @Success 200 {object} model.SetIsActiveResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/setIsActive [post] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) SetIsActive(c *gin.Context) {
	// Read raw body to validate required field presence
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		errorResponse(c, "INVALID_REQUEST", "failed to read request body", http.StatusBadRequest)
		return
	}

	// Validate that is_active field is present in JSON (required by OpenAPI spec)
	var rawData map[string]interface{}
	if err = json.Unmarshal(body, &rawData); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid JSON format", http.StatusBadRequest)
		return
	}

	if _, exists := rawData["is_active"]; !exists {
		errorResponse(c, "INVALID_REQUEST", "is_active field is required", http.StatusBadRequest)
		return
	}

	// Parse into struct
	var req model.SetIsActiveRequest
	if err = json.Unmarshal(body, &req); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SetIsActive(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			notFoundResponse(c, "user not found")
			return
		}
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetReview handles GET /users/getReview request.
// Returns 200 with empty list for nonexistent users rather than 404.
// @Summary Get PRs assigned to user
// @Tags Users
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} model.GetReviewResponse
// @Failure 400 {object} ErrorResponse
// @Router /users/getReview [get] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) GetReview(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		errorResponse(c, "INVALID_REQUEST", "user_id parameter is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetReview(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			c.JSON(http.StatusOK, &model.GetReviewResponse{
				UserID:       userID,
				PullRequests: []model.PullRequestShort{},
			})
			return
		}
		h.logger.Errorw("error getting review for user", "user_id", userID, "error", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}
