// Package handler provides HTTP handlers for team endpoints.
package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	teamModel "github.com/festy23/avito_internship/internal/team/model"
	"github.com/festy23/avito_internship/internal/team/service"
)

// Handler handles HTTP requests for team endpoints.
type Handler struct {
	service service.Service
}

// New creates a new team handler instance.
func New(svc service.Service) *Handler {
	return &Handler{service: svc}
}

// AddTeam handles POST /team/add request.
// @Summary Create a team with members
// @Tags Teams
// @Accept json
// @Produce json
// @Param request body teamModel.AddTeamRequest true "Request"
// @Success 201 {object} map[string]teamModel.TeamResponse "Response wrapped in team object"
// @Failure 400 {object} ErrorResponse "Bad request (TEAM_EXISTS, INVALID_REQUEST)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /team/add [post] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) AddTeam(c *gin.Context) {
	var req teamModel.AddTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.AddTeam(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, teamModel.ErrTeamExists) {
			errorResponse(c, "TEAM_EXISTS", "team_name already exists", http.StatusBadRequest)
			return
		}
		if errors.Is(err, teamModel.ErrInvalidTeamName) {
			errorResponse(c, "INVALID_REQUEST", "team_name is required", http.StatusBadRequest)
			return
		}
		if errors.Is(err, teamModel.ErrEmptyMembers) {
			errorResponse(c, "INVALID_REQUEST", "members list cannot be empty", http.StatusBadRequest)
			return
		}
		log.Printf("error adding team: %v", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"team": resp,
	})
}

// GetTeam handles GET /team/get request.
// @Summary Get a team with members
// @Tags Teams
// @Produce json
// @Param team_name query string true "Team Name"
// @Success 200 {object} teamModel.TeamResponse "Team response"
// @Failure 400 {object} ErrorResponse "Bad request (missing team_name parameter)"
// @Failure 404 {object} ErrorResponse "Team not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /team/get [get] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		errorResponse(c, "INVALID_REQUEST", "team_name parameter is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetTeam(c.Request.Context(), teamName)
	if err != nil {
		if errors.Is(err, teamModel.ErrTeamNotFound) {
			notFoundResponse(c, "team not found")
			return
		}
		log.Printf("error getting team %s: %v", teamName, err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}
