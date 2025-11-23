// Package service provides business logic layer for user module.
package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	pullrequestRepo "github.com/festy23/avito_internship/internal/pullrequest/repository"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
	teamRepo "github.com/festy23/avito_internship/internal/team/repository"
	userModel "github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/repository"
)

// Service defines the interface for user business logic operations.
type Service interface {
	// SetIsActive updates user activity status.
	SetIsActive(
		ctx context.Context,
		req *userModel.SetIsActiveRequest,
	) (*userModel.SetIsActiveResponse, error)

	// GetReview returns PRs assigned to user.
	GetReview(ctx context.Context, userID string) (*userModel.GetReviewResponse, error)

	// BulkDeactivateTeamMembers deactivates all team members and safely reassigns open PRs.
	BulkDeactivateTeamMembers(
		ctx context.Context,
		req *userModel.BulkDeactivateTeamRequest,
	) (*userModel.BulkDeactivateTeamResponse, error)
}

type service struct {
	repo            repository.Repository
	teamRepo        teamRepo.Repository
	pullrequestRepo pullrequestRepo.Repository
	db              *gorm.DB
	logger          *zap.SugaredLogger
}

// New creates a new user service instance.
func New(repo repository.Repository, logger *zap.SugaredLogger) Service {
	return &service{repo: repo, logger: logger}
}

// NewWithDependencies creates a new user service instance with additional dependencies.
func NewWithDependencies(
	repo repository.Repository,
	teamRepo teamRepo.Repository,
	pullrequestRepo pullrequestRepo.Repository,
	db *gorm.DB,
	logger *zap.SugaredLogger,
) Service {
	return &service{
		repo:            repo,
		teamRepo:        teamRepo,
		pullrequestRepo: pullrequestRepo,
		db:              db,
		logger:          logger,
	}
}

// SetIsActive updates user activity status.
func (s *service) SetIsActive(
	ctx context.Context,
	req *userModel.SetIsActiveRequest,
) (*userModel.SetIsActiveResponse, error) {
	s.logger.Debugw("SetIsActive called", "user_id", req.UserID, "is_active", req.IsActive)

	if req.UserID == "" {
		s.logger.Debugw("SetIsActive validation failed", "error", "empty user_id")
		return nil, userModel.ErrUserNotFound
	}

	user, err := s.repo.UpdateIsActive(ctx, req.UserID, req.IsActive)
	if err != nil {
		s.logger.Errorw(
			"SetIsActive failed",
			"user_id",
			req.UserID,
			"is_active",
			req.IsActive,
			"error",
			err,
		)
		return nil, err
	}

	s.logger.Infow("SetIsActive completed", "user_id", req.UserID, "new_state", req.IsActive)
	return &userModel.SetIsActiveResponse{User: *user}, nil
}

// GetReview returns PRs assigned to user.
func (s *service) GetReview(
	ctx context.Context,
	userID string,
) (*userModel.GetReviewResponse, error) {
	s.logger.Debugw("GetReview called", "user_id", userID)

	if userID == "" {
		s.logger.Debugw("GetReview validation failed", "error", "empty user_id")
		return nil, userModel.ErrUserNotFound
	}

	prs, err := s.repo.GetAssignedPullRequests(ctx, userID)
	if err != nil {
		s.logger.Errorw("GetReview failed", "user_id", userID, "error", err)
		return nil, err
	}

	s.logger.Infow("GetReview completed", "user_id", userID, "pr_count", len(prs))
	return &userModel.GetReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}

// BulkDeactivateTeamMembers deactivates all team members and safely reassigns open PRs.
//
//nolint:gocognit,funlen // Complex business logic with multiple steps
func (s *service) BulkDeactivateTeamMembers(
	ctx context.Context,
	req *userModel.BulkDeactivateTeamRequest,
) (*userModel.BulkDeactivateTeamResponse, error) {
	s.logger.Infow("BulkDeactivateTeamMembers called", "team_name", req.TeamName)

	if req.TeamName == "" {
		return nil, errors.New("team_name is required")
	}

	// Check if team exists
	_, err := s.teamRepo.GetByName(ctx, req.TeamName)
	if err != nil {
		if errors.Is(err, teamModel.ErrTeamNotFound) {
			return nil, err
		}
		s.logger.Errorw(
			"BulkDeactivateTeamMembers failed to get team",
			"team_name",
			req.TeamName,
			"error",
			err,
		)
		return nil, err
	}

	var result *userModel.BulkDeactivateTeamResponse

	// Use transaction to ensure atomicity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txUserRepo := repository.New(tx, s.logger)
		txPRRepo := pullrequestRepo.New(tx, s.logger)

		// Get active team members BEFORE deactivating (to get candidates before they're deactivated)
		activeCandidates, candidatesErr := txPRRepo.GetActiveTeamMembers(ctx, req.TeamName, "")
		if candidatesErr != nil {
			return candidatesErr
		}

		// Bulk deactivate (this also returns deactivated user IDs)
		deactivatedUserIDs, deactivateErr := txUserRepo.BulkDeactivateTeamMembers(ctx, req.TeamName)
		if deactivateErr != nil {
			return deactivateErr
		}

		if len(deactivatedUserIDs) == 0 {
			result = &userModel.BulkDeactivateTeamResponse{
				TeamName:          req.TeamName,
				DeactivatedUsers:  []string{},
				ReassignedPRs:     []string{},
				DeactivatedCount:  0,
				ReassignedPRCount: 0,
			}
			return nil
		}

		// Get open PRs with authors (optimized: single query)
		prAuthors, prsErr := txPRRepo.GetOpenPRsWithAuthors(ctx, deactivatedUserIDs)
		if prsErr != nil {
			return prsErr
		}

		if len(prAuthors) == 0 {
			result = &userModel.BulkDeactivateTeamResponse{
				TeamName:          req.TeamName,
				DeactivatedUsers:  deactivatedUserIDs,
				ReassignedPRs:     []string{},
				DeactivatedCount:  len(deactivatedUserIDs),
				ReassignedPRCount: 0,
			}
			return nil
		}

		// Reassign reviewers for each PR
		reassignedPRs := make([]string, 0)
		for prID, authorID := range prAuthors {
			reassignErr := s.reassignDeactivatedReviewersOptimized(
				ctx, txPRRepo, prID, authorID, deactivatedUserIDs, activeCandidates)
			if reassignErr != nil {
				// Log error but continue with other PRs
				s.logger.Warnw(
					"BulkDeactivateTeamMembers failed to reassign PR",
					"pr_id",
					prID,
					"error",
					reassignErr,
				)
				continue
			}
			reassignedPRs = append(reassignedPRs, prID)
		}

		result = &userModel.BulkDeactivateTeamResponse{
			TeamName:          req.TeamName,
			DeactivatedUsers:  deactivatedUserIDs,
			ReassignedPRs:     reassignedPRs,
			DeactivatedCount:  len(deactivatedUserIDs),
			ReassignedPRCount: len(reassignedPRs),
		}

		return nil
	})

	if err != nil {
		s.logger.Errorw("BulkDeactivateTeamMembers failed", "team_name", req.TeamName, "error", err)
		return nil, err
	}

	s.logger.Infow("BulkDeactivateTeamMembers completed",
		"team_name", req.TeamName,
		"deactivated_count", result.DeactivatedCount,
		"reassigned_pr_count", result.ReassignedPRCount)

	return result, nil
}

// reassignDeactivatedReviewersOptimized reassigns deactivated reviewers in a PR (optimized version).
//
//nolint:gocognit,gocyclo // Complex business logic with multiple steps and filtering
func (s *service) reassignDeactivatedReviewersOptimized(
	ctx context.Context,
	prRepo pullrequestRepo.Repository,
	prID string,
	authorID string,
	deactivatedUserIDs []string,
	activeCandidates []userModel.User,
) error {
	// Get current reviewers
	reviewers, err := prRepo.GetReviewers(ctx, prID)
	if err != nil {
		return err
	}

	// Find deactivated reviewers in this PR
	deactivatedInPR := make([]string, 0)
	deactivatedSet := make(map[string]bool, len(deactivatedUserIDs))
	for _, id := range deactivatedUserIDs {
		deactivatedSet[id] = true
	}

	for _, reviewerID := range reviewers {
		if deactivatedSet[reviewerID] {
			deactivatedInPR = append(deactivatedInPR, reviewerID)
		}
	}

	if len(deactivatedInPR) == 0 {
		return nil // No deactivated reviewers in this PR
	}

	// Filter candidates: exclude author, already assigned reviewers, and deactivated users
	reviewerSet := make(map[string]bool, len(reviewers))
	for _, reviewerID := range reviewers {
		reviewerSet[reviewerID] = true
	}

	filteredCandidates := make([]userModel.User, 0, len(activeCandidates))
	for _, candidate := range activeCandidates {
		// Skip author
		if candidate.UserID == authorID {
			continue
		}
		// Skip already assigned reviewers
		if reviewerSet[candidate.UserID] {
			continue
		}
		// Skip deactivated users (they were in activeCandidates before deactivation)
		if deactivatedSet[candidate.UserID] {
			continue
		}
		filteredCandidates = append(filteredCandidates, candidate)
	}

	// Reassign each deactivated reviewer
	for _, deactivatedID := range deactivatedInPR {
		if len(filteredCandidates) == 0 {
			// No candidates available, just remove the reviewer
			if removeErr := prRepo.RemoveReviewer(ctx, prID, deactivatedID); removeErr != nil {
				return removeErr
			}
			continue
		}

		// Select random replacement
		selected := selectRandomReviewers(filteredCandidates, 1)
		if len(selected) == 0 {
			// No candidates, just remove
			if removeErr := prRepo.RemoveReviewer(ctx, prID, deactivatedID); removeErr != nil {
				return removeErr
			}
			continue
		}

		newReviewerID := selected[0].UserID

		// Remove old and assign new
		if removeErr := prRepo.RemoveReviewer(ctx, prID, deactivatedID); removeErr != nil {
			return removeErr
		}

		if assignErr := prRepo.AssignReviewer(ctx, prID, newReviewerID); assignErr != nil {
			// If assignment fails (e.g., max reviewers), just continue
			s.logger.Debugw(
				"BulkDeactivateTeamMembers failed to assign reviewer",
				"pr_id",
				prID,
				"user_id",
				newReviewerID,
				"error",
				assignErr,
			)
		}

		// Remove assigned reviewer from candidates for next iteration
		for i, candidate := range filteredCandidates {
			if candidate.UserID == newReviewerID {
				filteredCandidates = append(filteredCandidates[:i], filteredCandidates[i+1:]...)
				break
			}
		}
	}

	return nil
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

	candidatesCopy := make([]userModel.User, len(candidates))
	copy(candidatesCopy, candidates)

	//nolint:gosec // G404: math/rand is sufficient for reviewer selection
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(candidatesCopy) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		candidatesCopy[i], candidatesCopy[j] = candidatesCopy[j], candidatesCopy[i]
	}

	return candidatesCopy[:count]
}
