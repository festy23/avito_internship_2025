// Package handler provides HTTP handlers for user endpoints.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/service"
)

// Handler handles HTTP requests for user endpoints.
type Handler struct {
	service service.Service
}

// New creates a new user handler instance.
func New(svc service.Service) *Handler {
	return &Handler{service: svc}
}

// SetIsActive handles POST /users/setIsActive request.
// @Summary Set user activity status
// @Tags Users
// @Accept json
// @Produce json
// @Param request body model.SetIsActiveRequest true "Request"
// @Success 200 {object} model.SetIsActiveResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/setIsActive [post].
func (h *Handler) SetIsActive(c *gin.Context) {
	var req model.SetIsActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
// @Router /users/getReview [get].
func (h *Handler) GetReview(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		errorResponse(c, "INVALID_REQUEST", "user_id parameter is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetReview(c.Request.Context(), userID)
	if err != nil {
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}
