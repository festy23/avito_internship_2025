// Package repository provides data access layer for pullrequest module.
package repository

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

// Repository defines the interface for pullrequest data access operations.
type Repository interface {
	// Create creates a new pull request.
	Create(
		ctx context.Context,
		prID, prName, authorID string,
	) (*pullrequestModel.PullRequest, error)

	// GetByID finds pull request by pull_request_id.
	GetByID(ctx context.Context, prID string) (*pullrequestModel.PullRequest, error)

	// UpdateStatus updates pull request status and merged_at timestamp.
	UpdateStatus(ctx context.Context, prID string, status string, mergedAt *time.Time) error

	// AssignReviewer assigns a reviewer to a pull request.
	AssignReviewer(ctx context.Context, prID, userID string) error

	// RemoveReviewer removes a reviewer from a pull request.
	RemoveReviewer(ctx context.Context, prID, userID string) error

	// GetReviewers returns list of user_id reviewers for a pull request.
	GetReviewers(ctx context.Context, prID string) ([]string, error)

	// GetActiveTeamMembers returns active team members excluding specified user.
	GetActiveTeamMembers(
		ctx context.Context,
		teamName string,
		excludeUserID string,
	) ([]userModel.User, error)

	// GetUserTeam returns team name for a user.
	GetUserTeam(ctx context.Context, userID string) (string, error)
}

type repository struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new pullrequest repository instance.
func New(db *gorm.DB, logger *zap.SugaredLogger) Repository {
	return &repository{db: db, logger: logger}
}

// Create creates a new pull request.
func (r *repository) Create(
	ctx context.Context,
	prID, prName, authorID string,
) (*pullrequestModel.PullRequest, error) {
	r.logger.Infow("Creating pull request", "pull_request_id", prID, "author_id", authorID)

	now := time.Now()
	pr := &pullrequestModel.PullRequest{
		PullRequestID:   prID,
		PullRequestName: prName,
		AuthorID:        authorID,
		Status:          "OPEN",
		CreatedAt:       now,
		MergedAt:        nil,
	}

	err := r.db.WithContext(ctx).Create(pr).Error
	if err != nil {
		// Check for unique constraint violation
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateError(err) {
			r.logger.Debugw("Create pull request duplicate key", "pull_request_id", prID)
			return nil, pullrequestModel.ErrPullRequestExists
		}
		r.logger.Errorw("Failed to create pull request", "pull_request_id", prID, "error", err)
		return nil, err
	}

	r.logger.Infow("Pull request created", "pull_request_id", prID, "author_id", authorID)
	return pr, nil
}

// isDuplicateError checks if error is a duplicate key error.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		(errStr != "" && (contains(errStr, "duplicate key") ||
			contains(errStr, "UNIQUE constraint")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetByID finds pull request by pull_request_id.
func (r *repository) GetByID(
	ctx context.Context,
	prID string,
) (*pullrequestModel.PullRequest, error) {
	r.logger.Debugw("GetByID called", "pull_request_id", prID)

	var pr pullrequestModel.PullRequest
	err := r.db.WithContext(ctx).
		Where("pull_request_id = ?", prID).
		First(&pr).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debugw("GetByID pull request not found", "pull_request_id", prID)
			return nil, pullrequestModel.ErrPullRequestNotFound
		}
		r.logger.Errorw("GetByID database error", "pull_request_id", prID, "error", err)
		return nil, err
	}

	return &pr, nil
}

// UpdateStatus updates pull request status and merged_at timestamp.
func (r *repository) UpdateStatus(
	ctx context.Context,
	prID string,
	status string,
	mergedAt *time.Time,
) error {
	r.logger.Infow("UpdateStatus called", "pull_request_id", prID, "new_status", status)

	updates := map[string]interface{}{
		"status": status,
	}

	if mergedAt != nil {
		updates["merged_at"] = *mergedAt
	} else {
		updates["merged_at"] = nil
	}

	result := r.db.WithContext(ctx).
		Model(&pullrequestModel.PullRequest{}).
		Where("pull_request_id = ?", prID).
		Updates(updates)

	if result.Error != nil {
		r.logger.Errorw("UpdateStatus database error", "pull_request_id", prID, "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Debugw("UpdateStatus pull request not found", "pull_request_id", prID)
		return pullrequestModel.ErrPullRequestNotFound
	}

	r.logger.Infow("UpdateStatus completed", "pull_request_id", prID, "new_status", status)
	return nil
}

// AssignReviewer assigns a reviewer to a pull request.
// Checks max reviewers limit (2) before assignment.
// Note: In production with PostgreSQL, this is enforced by database trigger.
func (r *repository) AssignReviewer(ctx context.Context, prID, userID string) error {
	r.logger.Infow("AssignReviewer called", "pull_request_id", prID, "user_id", userID)

	// Check current reviewer count to enforce max 2 reviewers limit
	// This is needed for SQLite compatibility (PostgreSQL uses trigger)
	reviewers, countErr := r.GetReviewers(ctx, prID)
	if countErr != nil {
		r.logger.Errorw("AssignReviewer failed to get reviewers", "pull_request_id", prID, "error", countErr)
		return countErr
	}
	// Check for duplicate reviewer
	for _, reviewerID := range reviewers {
		if reviewerID == userID {
			r.logger.Debugw("AssignReviewer duplicate reviewer", "pull_request_id", prID, "user_id", userID)
			return pullrequestModel.ErrReviewerAlreadyAssigned
		}
	}
	if len(reviewers) >= 2 {
		r.logger.Debugw(
			"AssignReviewer max reviewers exceeded",
			"pull_request_id", prID,
			"current_count", len(reviewers),
		)
		return pullrequestModel.ErrMaxReviewersExceeded
	}

	reviewer := &pullrequestModel.PullRequestReviewer{
		PullRequestID: prID,
		UserID:        userID,
		AssignedAt:    time.Now(),
	}

	err := r.db.WithContext(ctx).Create(reviewer).Error
	if err != nil {
		// Check for unique constraint violation (same reviewer already assigned)
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateError(err) {
			r.logger.Debugw("AssignReviewer duplicate key constraint", "pull_request_id", prID, "user_id", userID)
			return pullrequestModel.ErrReviewerAlreadyAssigned
		}
		// Check for max reviewers constraint from trigger (atomic protection)
		if err.Error() != "" && contains(err.Error(), "Maximum 2 reviewers") {
			r.logger.Debugw("AssignReviewer max reviewers constraint", "pull_request_id", prID)
			return pullrequestModel.ErrMaxReviewersExceeded
		}
		// Check for author constraint from trigger
		if err.Error() != "" && contains(err.Error(), "Author cannot be assigned") {
			r.logger.Debugw("AssignReviewer author constraint", "pull_request_id", prID, "user_id", userID)
			return pullrequestModel.ErrAuthorCannotBeReviewer
		}
		r.logger.Errorw("AssignReviewer database error", "pull_request_id", prID, "user_id", userID, "error", err)
		return err
	}

	r.logger.Infow("AssignReviewer completed", "pull_request_id", prID, "user_id", userID)
	return nil
}

// RemoveReviewer removes a reviewer from a pull request.
func (r *repository) RemoveReviewer(ctx context.Context, prID, userID string) error {
	r.logger.Infow("RemoveReviewer called", "pull_request_id", prID, "user_id", userID)

	result := r.db.WithContext(ctx).
		Where("pull_request_id = ? AND user_id = ?", prID, userID).
		Delete(&pullrequestModel.PullRequestReviewer{})

	if result.Error != nil {
		r.logger.Errorw(
			"RemoveReviewer database error",
			"pull_request_id",
			prID,
			"user_id",
			userID,
			"error",
			result.Error,
		)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Debugw("RemoveReviewer reviewer not assigned", "pull_request_id", prID, "user_id", userID)
		return pullrequestModel.ErrReviewerNotAssigned
	}

	r.logger.Infow("RemoveReviewer completed", "pull_request_id", prID, "user_id", userID)
	return nil
}

// GetReviewers returns list of user_id reviewers for a pull request.
func (r *repository) GetReviewers(ctx context.Context, prID string) ([]string, error) {
	r.logger.Debugw("GetReviewers called", "pull_request_id", prID)

	var reviewers []pullrequestModel.PullRequestReviewer
	err := r.db.WithContext(ctx).
		Where("pull_request_id = ?", prID).
		Order("assigned_at ASC").
		Find(&reviewers).Error

	if err != nil {
		r.logger.Errorw("GetReviewers database error", "pull_request_id", prID, "error", err)
		return nil, err
	}

	userIDs := make([]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		userIDs = append(userIDs, reviewer.UserID)
	}

	if userIDs == nil {
		userIDs = []string{}
	}

	r.logger.Debugw("GetReviewers completed", "pull_request_id", prID, "reviewer_count", len(userIDs))
	return userIDs, nil
}

// GetActiveTeamMembers returns active team members excluding specified user.
func (r *repository) GetActiveTeamMembers(
	ctx context.Context,
	teamName string,
	excludeUserID string,
) ([]userModel.User, error) {
	r.logger.Debugw("GetActiveTeamMembers called", "team_name", teamName, "exclude_user_id", excludeUserID)

	var users []userModel.User
	query := r.db.WithContext(ctx).
		Where("team_name = ? AND is_active = ?", teamName, true)

	if excludeUserID != "" {
		query = query.Where("user_id != ?", excludeUserID)
	}

	err := query.Order("user_id ASC").Find(&users).Error

	if err != nil {
		r.logger.Errorw("GetActiveTeamMembers database error", "team_name", teamName, "error", err)
		return nil, err
	}

	if users == nil {
		users = []userModel.User{}
	}

	r.logger.Debugw("GetActiveTeamMembers completed", "team_name", teamName, "member_count", len(users))
	return users, nil
}

// GetUserTeam returns team name for a user.
func (r *repository) GetUserTeam(ctx context.Context, userID string) (string, error) {
	r.logger.Debugw("GetUserTeam called", "user_id", userID)

	var user userModel.User
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debugw("GetUserTeam user not found", "user_id", userID)
			return "", pullrequestModel.ErrAuthorNotFound
		}
		r.logger.Errorw("GetUserTeam database error", "user_id", userID, "error", err)
		return "", err
	}

	r.logger.Debugw("GetUserTeam completed", "user_id", userID, "team_name", user.TeamName)
	return user.TeamName, nil
}
