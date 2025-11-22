// Package model provides domain models and DTOs for team module.
package model

// TeamMember represents a team member in API responses.
// Used in team creation and retrieval.
type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// AddTeamRequest represents the request to create a team with members.
type AddTeamRequest struct {
	TeamName string       `json:"team_name" binding:"required"`
	Members  []TeamMember `json:"members" binding:"required,dive"`
}

// TeamResponse represents the response after creating or getting a team.
type TeamResponse struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

