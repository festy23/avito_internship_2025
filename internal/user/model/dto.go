// Package model provides domain models and DTOs for user module.
package model

// SetIsActiveRequest represents the request to update user activity status.
// Note: IsActive doesn't use binding:"required" because Gin treats false as zero value
// and fails validation. The field is required by OpenAPI spec and is validated in handler
// to ensure it's present in the JSON request body.
type SetIsActiveRequest struct {
	UserID   string `json:"user_id"   binding:"required"`
	IsActive bool   `json:"is_active"`
}

// SetIsActiveResponse represents the response after updating user activity.
type SetIsActiveResponse struct {
	User User `json:"user"`
}

// PullRequestShort represents a shortened pull request information.
// Used in GetReviewResponse.
type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"` // OPEN or MERGED
}

// GetReviewResponse represents the response for getting user's assigned PRs.
type GetReviewResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

// BulkDeactivateTeamRequest represents the request to bulk deactivate team members.
type BulkDeactivateTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"`
}

// BulkDeactivateTeamResponse represents the response after bulk deactivation.
type BulkDeactivateTeamResponse struct {
	TeamName          string   `json:"team_name"`
	DeactivatedUsers  []string `json:"deactivated_users"`
	ReassignedPRs     []string `json:"reassigned_prs"`
	DeactivatedCount  int      `json:"deactivated_count"`
	ReassignedPRCount int      `json:"reassigned_pr_count"`
}
