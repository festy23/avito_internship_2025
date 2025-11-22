package model

// SetIsActiveRequest represents the request to update user activity status.
type SetIsActiveRequest struct {
	UserID   string `json:"user_id" binding:"required"`
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

