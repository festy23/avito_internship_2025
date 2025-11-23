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
	if req.UserID == "" {
		return nil, model.ErrUserNotFound
	}

	if req.IsActive == nil {
		return nil, model.ErrInvalidIsActive
	}

	user, err := s.repo.UpdateIsActive(ctx, req.UserID, *req.IsActive)
	if err != nil {
		return nil, err
	}

	return &model.SetIsActiveResponse{User: *user}, nil
}

// GetReview returns PRs assigned to user.
func (s *service) GetReview(ctx context.Context, userID string) (*model.GetReviewResponse, error) {
	if userID == "" {
		return nil, model.ErrUserNotFound
	}

	prs, err := s.repo.GetAssignedPullRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.GetReviewResponse{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}
