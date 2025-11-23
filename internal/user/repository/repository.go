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
	var user model.User
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// UpdateIsActive updates user's is_active flag using RETURNING clause for atomicity.
func (r *repository) UpdateIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("user_id = ?", userID).
		Update("is_active", isActive)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, model.ErrUserNotFound
	}

	// Fetch updated user
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAssignedPullRequests returns PRs where user is reviewer.
func (r *repository) GetAssignedPullRequests(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	var prs []model.PullRequestShort

	err := r.db.WithContext(ctx).
		Table("pull_request_reviewers").
		Select("pull_requests.pull_request_id, pull_requests.pull_request_name, pull_requests.author_id, pull_requests.status").
		Joins("JOIN pull_requests ON pull_request_reviewers.pull_request_id = pull_requests.pull_request_id").
		Where("pull_request_reviewers.user_id = ?", userID).
		Order("pull_requests.created_at DESC").
		Scan(&prs).Error

	if err != nil {
		return nil, err
	}

	if prs == nil {
		return []model.PullRequestShort{}, nil
	}

	return prs, nil
}
