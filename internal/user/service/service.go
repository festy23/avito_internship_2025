// Package service provides business logic layer for user module.
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/user/model"
	"github.com/festy23/avito_internship/internal/user/repository"
)

// Service defines the interface for user business logic operations.
type Service interface {
	// SetIsActive updates user activity status.
	SetIsActive(ctx context.Context, req *model.SetIsActiveRequest) (*model.SetIsActiveResponse, error)

	// GetReview returns PRs assigned to user.
	GetReview(ctx context.Context, userID string) (*model.GetReviewResponse, error)
}

type service struct {
	repo   repository.Repository
	logger *zap.SugaredLogger
}

// New creates a new user service instance.
func New(repo repository.Repository, logger *zap.SugaredLogger) Service {
	return &service{repo: repo, logger: logger}
}

// SetIsActive updates user activity status.
func (s *service) SetIsActive(ctx context.Context, req *model.SetIsActiveRequest) (*model.SetIsActiveResponse, error) {
	s.logger.Debugw("SetIsActive called", "user_id", req.UserID, "is_active", req.IsActive)

	if req.UserID == "" {
		s.logger.Debugw("SetIsActive validation failed", "error", "empty user_id")
		return nil, model.ErrUserNotFound
	}

	if req.IsActive == nil {
		s.logger.Debugw("SetIsActive validation failed", "error", "is_active is nil")
		return nil, model.ErrInvalidIsActive
	}

	user, err := s.repo.UpdateIsActive(ctx, req.UserID, *req.IsActive)
	if err != nil {
		s.logger.Errorw("SetIsActive failed", "user_id", req.UserID, "is_active", *req.IsActive, "error", err)
		return nil, err
	}

	s.logger.Infow("SetIsActive completed", "user_id", req.UserID, "new_state", *req.IsActive)
	return &model.SetIsActiveResponse{User: *user}, nil
}

// GetReview returns PRs assigned to user.
func (s *service) GetReview(ctx context.Context, userID string) (*model.GetReviewResponse, error) {
	s.logger.Debugw("GetReview called", "user_id", userID)

	if userID == "" {
		s.logger.Debugw("GetReview validation failed", "error", "empty user_id")
		return nil, model.ErrUserNotFound
	}

	prs, err := s.repo.GetAssignedPullRequests(ctx, userID)
	if err != nil {
		s.logger.Errorw("GetReview failed", "user_id", userID, "error", err)
		return nil, err
	}

	s.logger.Infow("GetReview completed", "user_id", userID, "pr_count", len(prs))
	return &model.GetReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}
