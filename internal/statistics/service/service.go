// Package service provides business logic layer for statistics module.
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/festy23/avito_internship/internal/statistics/model"
	"github.com/festy23/avito_internship/internal/statistics/repository"
)

// Service defines the interface for statistics business logic operations.
type Service interface {
	// GetReviewersStatistics returns statistics for all reviewers.
	GetReviewersStatistics(ctx context.Context) (*model.ReviewersStatisticsResponse, error)

	// GetPullRequestStatistics returns statistics for pull requests.
	GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatisticsResponse, error)
}

type service struct {
	repo   repository.Repository
	logger *zap.SugaredLogger
}

// New creates a new statistics service instance.
func New(repo repository.Repository, logger *zap.SugaredLogger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// GetReviewersStatistics returns statistics for all reviewers.
func (s *service) GetReviewersStatistics(ctx context.Context) (*model.ReviewersStatisticsResponse, error) {
	s.logger.Debugw("GetReviewersStatistics called")

	reviewers, err := s.repo.GetReviewersStatistics(ctx)
	if err != nil {
		s.logger.Errorw("GetReviewersStatistics failed", "error", err)
		return nil, err
	}

	if reviewers == nil {
		reviewers = []model.ReviewerStatistics{}
	}

	s.logger.Infow("GetReviewersStatistics completed", "count", len(reviewers))
	return &model.ReviewersStatisticsResponse{
		Reviewers: reviewers,
		Total:     len(reviewers),
	}, nil
}

// GetPullRequestStatistics returns statistics for pull requests.
func (s *service) GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatisticsResponse, error) {
	s.logger.Debugw("GetPullRequestStatistics called")

	stats, err := s.repo.GetPullRequestStatistics(ctx)
	if err != nil {
		s.logger.Errorw("GetPullRequestStatistics failed", "error", err)
		return nil, err
	}

	s.logger.Infow("GetPullRequestStatistics completed", "total_prs", stats.TotalPRs)
	return &model.PullRequestStatisticsResponse{
		Statistics: *stats,
	}, nil
}
