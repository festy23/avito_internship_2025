// Package service provides business logic layer for pullrequest module.
package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	"github.com/festy23/avito_internship/internal/pullrequest/repository"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

// Service defines the interface for pullrequest business logic operations.
type Service interface {
	// CreatePullRequest creates a new pull request with automatic reviewer assignment.
	CreatePullRequest(
		ctx context.Context,
		req *pullrequestModel.CreatePullRequestRequest,
	) (*pullrequestModel.PullRequestResponse, error)

	// MergePullRequest marks a pull request as MERGED (idempotent operation).
	MergePullRequest(
		ctx context.Context,
		req *pullrequestModel.MergePullRequestRequest,
	) (*pullrequestModel.PullRequestResponse, error)

	// ReassignReviewer reassigns a reviewer to another from the same team.
	ReassignReviewer(
		ctx context.Context,
		req *pullrequestModel.ReassignReviewerRequest,
	) (*pullrequestModel.ReassignReviewerResponse, error)
}

type service struct {
	repo   repository.Repository
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new pullrequest service instance.
func New(repo repository.Repository, db *gorm.DB, logger *zap.SugaredLogger) Service {
	return &service{
		repo:   repo,
		db:     db,
		logger: logger,
	}
}

// CreatePullRequest creates a new pull request with automatic reviewer assignment.
func (s *service) CreatePullRequest(
	ctx context.Context,
	req *pullrequestModel.CreatePullRequestRequest,
) (*pullrequestModel.PullRequestResponse, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Get author's team (before transaction to fail fast if author doesn't exist)
	teamName, err := s.repo.GetUserTeam(ctx, req.AuthorID)
	if err != nil {
		return nil, err
	}

	// Get active team members excluding author (before transaction to fail fast)
	candidates, err := s.repo.GetActiveTeamMembers(ctx, teamName, req.AuthorID)
	if err != nil {
		return nil, err
	}

	// Select up to 2 random reviewers
	selectedReviewers := selectRandomReviewers(candidates, 2)

	// Use transaction to ensure atomicity
	// Check for existing PR inside transaction to prevent race condition
	var result *pullrequestModel.PullRequestResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = s.createPRInTransaction(ctx, tx, req, selectedReviewers)
		return txErr
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// validateCreateRequest validates the create pull request request.
func (s *service) validateCreateRequest(req *pullrequestModel.CreatePullRequestRequest) error {
	if req.PullRequestID == "" {
		return pullrequestModel.ErrInvalidPullRequestID
	}
	if req.PullRequestName == "" {
		return errors.New("pull_request_name is required")
	}

	// Validate string lengths (CHECK constraints: 1-255)
	if len(req.PullRequestID) == 0 || len(req.PullRequestID) > 255 {
		return pullrequestModel.ErrInvalidPullRequestID
	}
	if len(req.PullRequestName) == 0 || len(req.PullRequestName) > 255 {
		return errors.New("pull_request_name must be between 1 and 255 characters")
	}
	if len(req.AuthorID) == 0 || len(req.AuthorID) > 255 {
		return pullrequestModel.ErrInvalidAuthorID
	}

	return nil
}

// createPRInTransaction creates PR and assigns reviewers within a transaction.
func (s *service) createPRInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	req *pullrequestModel.CreatePullRequestRequest,
	selectedReviewers []userModel.User,
) (*pullrequestModel.PullRequestResponse, error) {
	txRepo := repository.New(tx, s.logger)

	// Check if PR already exists (inside transaction to prevent race condition)
	existingPR, checkErr := txRepo.GetByID(ctx, req.PullRequestID)
	if checkErr != nil && !errors.Is(checkErr, pullrequestModel.ErrPullRequestNotFound) {
		return nil, checkErr
	}
	if existingPR != nil {
		return nil, pullrequestModel.ErrPullRequestExists
	}

	// Create PR
	pr, createErr := txRepo.Create(ctx, req.PullRequestID, req.PullRequestName, req.AuthorID)
	if createErr != nil {
		return nil, createErr
	}

	// Assign reviewers
	for _, reviewer := range selectedReviewers {
		if assignErr := txRepo.AssignReviewer(ctx, req.PullRequestID, reviewer.UserID); assignErr != nil {
			return nil, assignErr
		}
	}

	// Get assigned reviewers
	reviewerIDs, getErr := txRepo.GetReviewers(ctx, req.PullRequestID)
	if getErr != nil {
		return nil, getErr
	}

	return &pullrequestModel.PullRequestResponse{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: reviewerIDs,
		CreatedAt:         pr.CreatedAt.Format(time.RFC3339),
	}, nil
}

// MergePullRequest marks a pull request as MERGED (idempotent operation).
func (s *service) MergePullRequest(
	ctx context.Context,
	req *pullrequestModel.MergePullRequestRequest,
) (*pullrequestModel.PullRequestResponse, error) {
	// Validate input
	if req.PullRequestID == "" {
		return nil, pullrequestModel.ErrInvalidPullRequestID
	}

	// Get PR
	pr, err := s.repo.GetByID(ctx, req.PullRequestID)
	if err != nil {
		return nil, err
	}

	// If already MERGED, return current state (idempotent)
	if pr.Status == "MERGED" {
		reviewerIDs, getErr := s.repo.GetReviewers(ctx, req.PullRequestID)
		if getErr != nil {
			return nil, getErr
		}

		mergedAt := ""
		if pr.MergedAt != nil {
			mergedAt = pr.MergedAt.Format(time.RFC3339)
		}

		return &pullrequestModel.PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewerIDs,
			CreatedAt:         pr.CreatedAt.Format(time.RFC3339),
			MergedAt:          mergedAt,
		}, nil
	}

	// Update status to MERGED
	now := time.Now()
	err = s.repo.UpdateStatus(ctx, req.PullRequestID, "MERGED", &now)
	if err != nil {
		return nil, err
	}

	// Get updated PR
	mergedPR, err := s.repo.GetByID(ctx, req.PullRequestID)
	if err != nil {
		return nil, err
	}

	// Get reviewers
	reviewerIDs, err := s.repo.GetReviewers(ctx, req.PullRequestID)
	if err != nil {
		return nil, err
	}

	mergedAt := ""
	if mergedPR.MergedAt != nil {
		mergedAt = mergedPR.MergedAt.Format(time.RFC3339)
	}

	return &pullrequestModel.PullRequestResponse{
		PullRequestID:     mergedPR.PullRequestID,
		PullRequestName:   mergedPR.PullRequestName,
		AuthorID:          mergedPR.AuthorID,
		Status:            mergedPR.Status,
		AssignedReviewers: reviewerIDs,
		CreatedAt:         mergedPR.CreatedAt.Format(time.RFC3339),
		MergedAt:          mergedAt,
	}, nil
}

// ReassignReviewer reassigns a reviewer to another from the same team.
func (s *service) ReassignReviewer(
	ctx context.Context,
	req *pullrequestModel.ReassignReviewerRequest,
) (*pullrequestModel.ReassignReviewerResponse, error) {
	if err := s.validateReassignRequest(req); err != nil {
		return nil, err
	}

	// Use transaction to ensure atomicity
	// All checks and operations inside transaction to prevent race conditions
	var result *pullrequestModel.ReassignReviewerResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = s.reassignInTransaction(ctx, tx, req)
		return txErr
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// validateReassignRequest validates the reassign reviewer request.
func (s *service) validateReassignRequest(req *pullrequestModel.ReassignReviewerRequest) error {
	if req.PullRequestID == "" {
		return pullrequestModel.ErrInvalidPullRequestID
	}
	if req.OldUserID == "" {
		return errors.New("old_user_id is required")
	}

	// Validate string lengths (CHECK constraints: 1-255)
	if len(req.PullRequestID) == 0 || len(req.PullRequestID) > 255 {
		return pullrequestModel.ErrInvalidPullRequestID
	}
	if len(req.OldUserID) == 0 || len(req.OldUserID) > 255 {
		return errors.New("old_user_id must be between 1 and 255 characters")
	}

	return nil
}

// reassignInTransaction performs reassignment within a transaction.
//
//nolint:gocognit,gocyclo // Complex business logic with multiple validation steps
func (s *service) reassignInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	req *pullrequestModel.ReassignReviewerRequest,
) (*pullrequestModel.ReassignReviewerResponse, error) {
	txRepo := repository.New(tx, s.logger)

	// Get PR (inside transaction)
	pr, txErr := txRepo.GetByID(ctx, req.PullRequestID)
	if txErr != nil {
		return nil, txErr
	}

	// Check if PR is already merged (inside transaction)
	if pr.Status == "MERGED" {
		return nil, pullrequestModel.ErrPullRequestMerged
	}

	// Get old reviewer's team first to check if user exists
	teamName, teamErr := txRepo.GetUserTeam(ctx, req.OldUserID)
	if teamErr != nil {
		// If user doesn't exist, return NOT_FOUND error (404)
		if errors.Is(teamErr, pullrequestModel.ErrAuthorNotFound) {
			return nil, pullrequestModel.ErrAuthorNotFound // User not found should return 404
		}
		return nil, teamErr
	}

	// Check if old_user_id is assigned as reviewer (inside transaction)
	reviewers, getErr := txRepo.GetReviewers(ctx, req.PullRequestID)
	if getErr != nil {
		return nil, getErr
	}

	if !isReviewerAssigned(reviewers, req.OldUserID) {
		return nil, pullrequestModel.ErrReviewerNotAssigned
	}

	// Get active team members excluding old reviewer
	candidates, candidatesErr := txRepo.GetActiveTeamMembers(ctx, teamName, req.OldUserID)
	if candidatesErr != nil {
		return nil, candidatesErr
	}

	// Filter candidates (exclude PR author and already assigned reviewers)
	filteredCandidates := filterCandidates(candidates, pr.AuthorID)
	// Also exclude already assigned reviewers (except the one being replaced)
	finalCandidates := make([]userModel.User, 0, len(filteredCandidates))
	for _, candidate := range filteredCandidates {
		isAssigned := false
		for _, reviewerID := range reviewers {
			if reviewerID == candidate.UserID && reviewerID != req.OldUserID {
				isAssigned = true
				break
			}
		}
		if !isAssigned {
			finalCandidates = append(finalCandidates, candidate)
		}
	}
	if len(finalCandidates) == 0 {
		return nil, pullrequestModel.ErrNoCandidate
	}

	// Select random replacement
	selected := selectRandomReviewers(finalCandidates, 1)
	if len(selected) == 0 {
		return nil, pullrequestModel.ErrNoCandidate
	}
	newReviewerID := selected[0].UserID

	// Remove old reviewer and assign new one
	if removeErr := txRepo.RemoveReviewer(ctx, req.PullRequestID, req.OldUserID); removeErr != nil {
		return nil, removeErr
	}

	if assignErr := txRepo.AssignReviewer(ctx, req.PullRequestID, newReviewerID); assignErr != nil {
		return nil, assignErr
	}

	// Get updated reviewers list
	reviewerIDs, reviewersErr := txRepo.GetReviewers(ctx, req.PullRequestID)
	if reviewersErr != nil {
		return nil, reviewersErr
	}

	// Get updated PR
	updatedPR, updatedErr := txRepo.GetByID(ctx, req.PullRequestID)
	if updatedErr != nil {
		return nil, updatedErr
	}

	mergedAt := ""
	if updatedPR.MergedAt != nil {
		mergedAt = updatedPR.MergedAt.Format(time.RFC3339)
	}

	return &pullrequestModel.ReassignReviewerResponse{
		PR: &pullrequestModel.PullRequestResponse{
			PullRequestID:     updatedPR.PullRequestID,
			PullRequestName:   updatedPR.PullRequestName,
			AuthorID:          updatedPR.AuthorID,
			Status:            updatedPR.Status,
			AssignedReviewers: reviewerIDs,
			CreatedAt:         updatedPR.CreatedAt.Format(time.RFC3339),
			MergedAt:          mergedAt,
		},
		ReplacedBy: newReviewerID,
	}, nil
}

// isReviewerAssigned checks if a user is assigned as reviewer.
func isReviewerAssigned(reviewers []string, userID string) bool {
	for _, reviewerID := range reviewers {
		if reviewerID == userID {
			return true
		}
	}
	return false
}

// filterCandidates filters out the PR author from candidate list.
func filterCandidates(candidates []userModel.User, authorID string) []userModel.User {
	filtered := make([]userModel.User, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.UserID != authorID {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

// selectRandomReviewers selects up to maxCount random reviewers from candidates.
func selectRandomReviewers(candidates []userModel.User, maxCount int) []userModel.User {
	if len(candidates) == 0 {
		return []userModel.User{}
	}

	count := maxCount
	if len(candidates) < maxCount {
		count = len(candidates)
	}

	// Create a copy to avoid modifying original slice
	candidatesCopy := make([]userModel.User, len(candidates))
	copy(candidatesCopy, candidates)

	// Shuffle using Fisher-Yates algorithm
	// Using math/rand is acceptable for this use case (non-cryptographic, low RPS)
	//nolint:gosec // G404: math/rand is sufficient for reviewer selection
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(candidatesCopy) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		candidatesCopy[i], candidatesCopy[j] = candidatesCopy[j], candidatesCopy[i]
	}

	return candidatesCopy[:count]
}
