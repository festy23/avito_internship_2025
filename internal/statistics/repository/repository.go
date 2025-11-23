// Package repository provides data access layer for statistics module.
package repository

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/statistics/model"
)

// Repository defines the interface for statistics data access operations.
type Repository interface {
	// GetReviewersStatistics returns statistics for all reviewers.
	GetReviewersStatistics(ctx context.Context) ([]model.ReviewerStatistics, error)

	// GetPullRequestStatistics returns statistics for pull requests.
	GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatistics, error)
}

type repository struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new statistics repository instance.
func New(db *gorm.DB, logger *zap.SugaredLogger) Repository {
	return &repository{
		db:     db,
		logger: logger,
	}
}

// GetReviewersStatistics returns statistics for all reviewers.
func (r *repository) GetReviewersStatistics(ctx context.Context) ([]model.ReviewerStatistics, error) {
	r.logger.Debugw("GetReviewersStatistics called")

	var stats []model.ReviewerStatistics

	err := r.db.WithContext(ctx).
		Table("users").
		Select(`
			users.user_id,
			users.username,
			users.team_name,
			users.is_active,
			COALESCE(COUNT(pull_request_reviewers.user_id), 0) as assignment_count
		`).
		Joins("LEFT JOIN pull_request_reviewers ON users.user_id = pull_request_reviewers.user_id").
		Group("users.user_id, users.username, users.team_name, users.is_active").
		Order("assignment_count DESC, users.user_id ASC").
		Scan(&stats).Error

	if err != nil {
		r.logger.Errorw("GetReviewersStatistics database error", "error", err)
		return nil, err
	}

	if stats == nil {
		stats = []model.ReviewerStatistics{}
	}

	r.logger.Debugw("GetReviewersStatistics completed", "count", len(stats))
	return stats, nil
}

// GetPullRequestStatistics returns statistics for pull requests.
func (r *repository) GetPullRequestStatistics(ctx context.Context) (*model.PullRequestStatistics, error) {
	r.logger.Debugw("GetPullRequestStatistics called")

	var result struct {
		TotalPRs              int64   `gorm:"column:total_prs"`
		OpenPRs               int64   `gorm:"column:open_prs"`
		MergedPRs             int64   `gorm:"column:merged_prs"`
		AverageReviewersPerPR float64 `gorm:"column:avg_reviewers"`
		PRsWith0Reviewers     int64   `gorm:"column:prs_0_reviewers"`
		PRsWith1Reviewer      int64   `gorm:"column:prs_1_reviewer"`
		PRsWith2Reviewers     int64   `gorm:"column:prs_2_reviewers"`
	}

	err := r.db.WithContext(ctx).
		Table("pull_requests").
		Select(`
			COUNT(*) as total_prs,
			SUM(CASE WHEN status = 'OPEN' THEN 1 ELSE 0 END) as open_prs,
			SUM(CASE WHEN status = 'MERGED' THEN 1 ELSE 0 END) as merged_prs,
			COALESCE(AVG(reviewer_counts.reviewer_count), 0) as avg_reviewers,
			SUM(CASE WHEN COALESCE(reviewer_counts.reviewer_count, 0) = 0 THEN 1 ELSE 0 END) as prs_0_reviewers,
			SUM(CASE WHEN COALESCE(reviewer_counts.reviewer_count, 0) = 1 THEN 1 ELSE 0 END) as prs_1_reviewer,
			SUM(CASE WHEN COALESCE(reviewer_counts.reviewer_count, 0) = 2 THEN 1 ELSE 0 END) as prs_2_reviewers
		`).
		Joins(`
			LEFT JOIN (
				SELECT pull_request_id, CAST(COUNT(*) AS REAL) as reviewer_count
				FROM pull_request_reviewers
				GROUP BY pull_request_id
			) reviewer_counts ON pull_requests.pull_request_id = reviewer_counts.pull_request_id
		`).
		Scan(&result).Error

	if err != nil {
		r.logger.Errorw("GetPullRequestStatistics database error", "error", err)
		return nil, err
	}

	stats := &model.PullRequestStatistics{
		TotalPRs:              int(result.TotalPRs),
		OpenPRs:               int(result.OpenPRs),
		MergedPRs:             int(result.MergedPRs),
		AverageReviewersPerPR: result.AverageReviewersPerPR,
		PRsWith0Reviewers:     int(result.PRsWith0Reviewers),
		PRsWith1Reviewer:      int(result.PRsWith1Reviewer),
		PRsWith2Reviewers:     int(result.PRsWith2Reviewers),
	}

	r.logger.Debugw("GetPullRequestStatistics completed", "total_prs", stats.TotalPRs)
	return stats, nil
}
