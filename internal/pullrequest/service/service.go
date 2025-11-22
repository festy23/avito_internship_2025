// Package service provides business logic layer for pullrequest module.
package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	"github.com/festy23/avito_internship/internal/pullrequest/repository"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

// Service defines the interface for pullrequest business logic operations.
type Service interface {
	// CreatePullRequest creates a new pull request with automatic reviewer assignment.
	CreatePullRequest(ctx context.Context, req *pullrequestModel.CreatePullRequestRequest) (*pullrequestModel.PullRequestResponse, error)

	// MergePullRequest marks a pull request as MERGED (idempotent operation).
	MergePullRequest(ctx context.Context, req *pullrequestModel.MergePullRequestRequest) (*pullrequestModel.PullRequestResponse, error)

	// ReassignReviewer reassigns a reviewer to another from the same team.
	ReassignReviewer(ctx context.Context, req *pullrequestModel.ReassignReviewerRequest) (*pullrequestModel.ReassignReviewerResponse, error)
}

type service struct {
	repo repository.Repository
	db   *gorm.DB
}

// New creates a new pullrequest service instance.
func New(repo repository.Repository, db *gorm.DB) Service {
	return &service{
		repo: repo,
		db:   db,
	}
}

// CreatePullRequest creates a new pull request with automatic reviewer assignment.
func (s *service) CreatePullRequest(ctx context.Context, req *pullrequestModel.CreatePullRequestRequest) (*pullrequestModel.PullRequestResponse, error) {
	// Validate input
	if req.PullRequestID == "" {
		return nil, pullrequestModel.ErrInvalidPullRequestID
	}
	if req.PullRequestName == "" {
		return nil, errors.New("pull_request_name is required")
	}
	if req.AuthorID == "" {
		return nil, pullrequestModel.ErrAuthorNotFound
	}

	// Check if PR already exists
	existingPR, err := s.repo.GetByID(ctx, req.PullRequestID)
	if err != nil && !errors.Is(err, pullrequestModel.ErrPullRequestNotFound) {
		return nil, err
	}
	if existingPR != nil {
		return nil, pullrequestModel.ErrPullRequestExists
	}

	// Get author's team
	teamName, err := s.repo.GetUserTeam(ctx, req.AuthorID)
	if err != nil {
		return nil, err
	}

	// Get active team members excluding author
	candidates, err := s.repo.GetActiveTeamMembers(ctx, teamName, req.AuthorID)
	if err != nil {
		return nil, err
	}

	// Select up to 2 random reviewers
	selectedReviewers := selectRandomReviewers(candidates, 2)

	// Use transaction to ensure atomicity
	var result *pullrequestModel.PullRequestResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create repository with transaction
		txRepo := repository.New(tx)

		// Create PR
		pr, err := txRepo.Create(ctx, req.PullRequestID, req.PullRequestName, req.AuthorID)
		if err != nil {
			return err
		}

		// Assign reviewers
		for _, reviewer := range selectedReviewers {
			err = txRepo.AssignReviewer(ctx, req.PullRequestID, reviewer.UserID)
			if err != nil {
				return err
			}
		}

		// Get assigned reviewers
		reviewerIDs, err := txRepo.GetReviewers(ctx, req.PullRequestID)
		if err != nil {
			return err
		}

		result = &pullrequestModel.PullRequestResponse{
			PullRequestID:    pr.PullRequestID,
			PullRequestName:  pr.PullRequestName,
			AuthorID:         pr.AuthorID,
			Status:           pr.Status,
			AssignedReviewers: reviewerIDs,
			CreatedAt:        pr.CreatedAt.Format(time.RFC3339),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// MergePullRequest marks a pull request as MERGED (idempotent operation).
func (s *service) MergePullRequest(ctx context.Context, req *pullrequestModel.MergePullRequestRequest) (*pullrequestModel.PullRequestResponse, error) {
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
		reviewerIDs, err := s.repo.GetReviewers(ctx, req.PullRequestID)
		if err != nil {
			return nil, err
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
func (s *service) ReassignReviewer(ctx context.Context, req *pullrequestModel.ReassignReviewerRequest) (*pullrequestModel.ReassignReviewerResponse, error) {
	// Validate input
	if req.PullRequestID == "" {
		return nil, pullrequestModel.ErrInvalidPullRequestID
	}
	if req.OldUserID == "" {
		return nil, errors.New("old_user_id is required")
	}

	// Get PR
	pr, err := s.repo.GetByID(ctx, req.PullRequestID)
	if err != nil {
		return nil, err
	}

	// Check if PR is already merged
	if pr.Status == "MERGED" {
		return nil, pullrequestModel.ErrPullRequestMerged
	}

	// Check if old_user_id is assigned as reviewer
	reviewers, err := s.repo.GetReviewers(ctx, req.PullRequestID)
	if err != nil {
		return nil, err
	}

	isAssigned := false
	for _, reviewerID := range reviewers {
		if reviewerID == req.OldUserID {
			isAssigned = true
			break
		}
	}

	if !isAssigned {
		return nil, pullrequestModel.ErrReviewerNotAssigned
	}

	// Get old reviewer's team
	teamName, err := s.repo.GetUserTeam(ctx, req.OldUserID)
	if err != nil {
		return nil, err
	}

	// Get active team members excluding old reviewer and PR author
	candidates, err := s.repo.GetActiveTeamMembers(ctx, teamName, req.OldUserID)
	if err != nil {
		return nil, err
	}

	// Exclude PR author from candidates
	filteredCandidates := make([]userModel.User, 0)
	for _, candidate := range candidates {
		if candidate.UserID != pr.AuthorID {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	if len(filteredCandidates) == 0 {
		return nil, pullrequestModel.ErrNoCandidate
	}

	// Select random replacement
	selected := selectRandomReviewers(filteredCandidates, 1)
	if len(selected) == 0 {
		return nil, pullrequestModel.ErrNoCandidate
	}
	newReviewerID := selected[0].UserID

	// Use transaction to ensure atomicity
	var result *pullrequestModel.ReassignReviewerResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create repository with transaction
		txRepo := repository.New(tx)

		// Remove old reviewer
		err = txRepo.RemoveReviewer(ctx, req.PullRequestID, req.OldUserID)
		if err != nil {
			return err
		}

		// Assign new reviewer
		err = txRepo.AssignReviewer(ctx, req.PullRequestID, newReviewerID)
		if err != nil {
			return err
		}

		// Get updated reviewers list
		reviewerIDs, err := txRepo.GetReviewers(ctx, req.PullRequestID)
		if err != nil {
			return err
		}

		// Get updated PR
		updatedPR, err := txRepo.GetByID(ctx, req.PullRequestID)
		if err != nil {
			return err
		}

		mergedAt := ""
		if updatedPR.MergedAt != nil {
			mergedAt = updatedPR.MergedAt.Format(time.RFC3339)
		}

		result = &pullrequestModel.ReassignReviewerResponse{
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
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(candidatesCopy) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		candidatesCopy[i], candidatesCopy[j] = candidatesCopy[j], candidatesCopy[i]
	}

	return candidatesCopy[:count]
}

