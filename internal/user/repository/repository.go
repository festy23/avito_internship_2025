// Package repository provides data access layer for user module.
package repository

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/user/model"
)

// Repository defines the interface for user data access operations.
type Repository interface {
	// GetByID finds user by user_id.
	GetByID(ctx context.Context, userID string) (*model.User, error)

	// UpdateIsActive updates user's is_active flag.
	UpdateIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error)

	// GetAssignedPullRequests returns PRs where user is reviewer.
	GetAssignedPullRequests(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

type repository struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new user repository instance.
func New(db *gorm.DB, logger *zap.SugaredLogger) Repository {
	return &repository{db: db, logger: logger}
}

// GetByID finds user by user_id.
func (r *repository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	r.logger.Debugw("GetByID called", "user_id", userID)

	var user model.User
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debugw("GetByID user not found", "user_id", userID)
			return nil, model.ErrUserNotFound
		}
		r.logger.Errorw("GetByID database error", "user_id", userID, "error", err)
		return nil, err
	}

	return &user, nil
}

// UpdateIsActive updates user's is_active flag using RETURNING clause for atomicity.
func (r *repository) UpdateIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	r.logger.Infow("UpdateIsActive called", "user_id", userID, "new_state", isActive)

	var user model.User
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("user_id = ?", userID).
		Update("is_active", isActive)

	if result.Error != nil {
		r.logger.Errorw("UpdateIsActive database error", "user_id", userID, "error", result.Error)
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Debugw("UpdateIsActive user not found", "user_id", userID)
		return nil, model.ErrUserNotFound
	}

	// Fetch updated user
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		r.logger.Errorw("UpdateIsActive failed to fetch updated user", "user_id", userID, "error", err)
		return nil, err
	}

	r.logger.Infow("UpdateIsActive completed", "user_id", userID, "new_state", isActive)
	return &user, nil
}

// GetAssignedPullRequests returns PRs where user is reviewer.
func (r *repository) GetAssignedPullRequests(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	r.logger.Debugw("GetAssignedPullRequests called", "user_id", userID)

	var prs []model.PullRequestShort

	err := r.db.WithContext(ctx).
		Table("pull_request_reviewers").
		Select("pull_requests.pull_request_id, pull_requests.pull_request_name, pull_requests.author_id, pull_requests.status").
		Joins("JOIN pull_requests ON pull_request_reviewers.pull_request_id = pull_requests.pull_request_id").
		Where("pull_request_reviewers.user_id = ?", userID).
		Order("pull_requests.created_at DESC").
		Scan(&prs).Error

	if err != nil {
		r.logger.Errorw("GetAssignedPullRequests database error", "user_id", userID, "error", err)
		return nil, err
	}

	if prs == nil {
		prs = []model.PullRequestShort{}
	}

	r.logger.Debugw("GetAssignedPullRequests completed", "user_id", userID, "pr_count", len(prs))
	return prs, nil
}
