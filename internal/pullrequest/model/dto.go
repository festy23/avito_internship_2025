// Package model provides data transfer objects and domain models for the pullrequest module.
package model

// CreatePullRequestRequest represents the request to create a pull request.
type CreatePullRequestRequest struct {
	PullRequestID   string `json:"pull_request_id"   binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id"         binding:"required"`
}

// MergePullRequestRequest represents the request to merge a pull request.
type MergePullRequestRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

// ReassignReviewerRequest represents the request to reassign a reviewer.
type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldUserID     string `json:"old_user_id"     binding:"required"`
}

// PullRequestResponse represents the response after creating or merging a pull request.
type PullRequestResponse struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         string   `json:"createdAt,omitempty"`
	MergedAt          string   `json:"mergedAt,omitempty"`
}

// ReassignReviewerResponse represents the response after reassigning a reviewer.
type ReassignReviewerResponse struct {
	PR         *PullRequestResponse `json:"pr"`
	ReplacedBy string               `json:"replaced_by"`
}
