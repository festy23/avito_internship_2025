// Package handler provides HTTP handlers for pullrequest endpoints.
package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	"github.com/festy23/avito_internship/internal/pullrequest/service"
)

// Handler handles HTTP requests for pullrequest endpoints.
type Handler struct {
	service service.Service
}

// New creates a new pullrequest handler instance.
func New(svc service.Service) *Handler {
	return &Handler{service: svc}
}

// CreatePullRequest handles POST /pullRequest/create request.
// @Summary Create a pull request with automatic reviewer assignment
// @Tags PullRequests
// @Accept json
// @Produce json
// @Param request body pullrequestModel.CreatePullRequestRequest true "Request"
// @Success 201 {object} map[string]pullrequestModel.PullRequestResponse "Response wrapped in pr object"
// @Failure 400 {object} ErrorResponse "Bad request (INVALID_REQUEST)"
// @Failure 404 {object} ErrorResponse "Author/team not found"
// @Failure 409 {object} ErrorResponse "PR already exists (PR_EXISTS)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pullRequest/create [post] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) CreatePullRequest(c *gin.Context) {
	var req pullrequestModel.CreatePullRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.CreatePullRequest(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, pullrequestModel.ErrPullRequestExists) {
			errorResponse(c, "PR_EXISTS", "PR id already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, pullrequestModel.ErrAuthorNotFound) {
			notFoundResponse(c, "author not found")
			return
		}
		if errors.Is(err, pullrequestModel.ErrInvalidPullRequestID) {
			errorResponse(c, "INVALID_REQUEST", "pull_request_id is required", http.StatusBadRequest)
			return
		}
		log.Printf("error creating pull request: %v", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"pr": resp,
	})
}

// MergePullRequest handles POST /pullRequest/merge request.
// @Summary Mark a pull request as MERGED (idempotent operation)
// @Tags PullRequests
// @Accept json
// @Produce json
// @Param request body pullrequestModel.MergePullRequestRequest true "Request"
// @Success 200 {object} map[string]pullrequestModel.PullRequestResponse "Response wrapped in pr object"
// @Failure 400 {object} ErrorResponse "Bad request (INVALID_REQUEST)"
// @Failure 404 {object} ErrorResponse "PR not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pullRequest/merge [post] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) MergePullRequest(c *gin.Context) {
	var req pullrequestModel.MergePullRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.MergePullRequest(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, pullrequestModel.ErrPullRequestNotFound) {
			notFoundResponse(c, "pull request not found")
			return
		}
		if errors.Is(err, pullrequestModel.ErrInvalidPullRequestID) {
			errorResponse(c, "INVALID_REQUEST", "pull_request_id is required", http.StatusBadRequest)
			return
		}
		log.Printf("error merging pull request: %v", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"pr": resp,
	})
}

// ReassignReviewer handles POST /pullRequest/reassign request.
// @Summary Reassign a reviewer to another from the same team
// @Tags PullRequests
// @Accept json
// @Produce json
// @Param request body pullrequestModel.ReassignReviewerRequest true "Request"
// @Success 200 {object} pullrequestModel.ReassignReviewerResponse "Response with pr and replaced_by"
// @Failure 400 {object} ErrorResponse "Bad request (INVALID_REQUEST)"
// @Failure 404 {object} ErrorResponse "PR or user not found"
// @Failure 409 {object} ErrorResponse "Domain rule violation (PR_MERGED, NOT_ASSIGNED, NO_CANDIDATE)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pullRequest/reassign [post] //nolint:godot // Swagger annotation should not end with period
func (h *Handler) ReassignReviewer(c *gin.Context) {
	var req pullrequestModel.ReassignReviewerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, "INVALID_REQUEST", "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ReassignReviewer(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, pullrequestModel.ErrPullRequestNotFound) {
			notFoundResponse(c, "pull request not found")
			return
		}
		if errors.Is(err, pullrequestModel.ErrPullRequestMerged) {
			errorResponse(c, "PR_MERGED", "cannot reassign on merged PR", http.StatusConflict)
			return
		}
		if errors.Is(err, pullrequestModel.ErrReviewerNotAssigned) {
			errorResponse(c, "NOT_ASSIGNED", "reviewer is not assigned to this PR", http.StatusConflict)
			return
		}
		if errors.Is(err, pullrequestModel.ErrNoCandidate) {
			errorResponse(c, "NO_CANDIDATE", "no active replacement candidate in team", http.StatusConflict)
			return
		}
		if errors.Is(err, pullrequestModel.ErrInvalidPullRequestID) {
			errorResponse(c, "INVALID_REQUEST", "pull_request_id is required", http.StatusBadRequest)
			return
		}
		log.Printf("error reassigning reviewer: %v", err)
		errorResponse(c, "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}

