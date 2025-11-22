// Package repository provides data access layer for team module.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	teamModel "github.com/festy23/avito_internship/internal/team/model"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

// Repository defines the interface for team data access operations.
type Repository interface {
	// Create creates a new team.
	Create(ctx context.Context, teamName string) (*teamModel.Team, error)

	// GetByName finds team by team_name.
	GetByName(ctx context.Context, teamName string) (*teamModel.Team, error)

	// CreateOrUpdateUser creates or updates a user in the team.
	CreateOrUpdateUser(ctx context.Context, teamName, userID, username string, isActive bool) (*userModel.User, error)

	// GetTeamMembers returns all members of a team.
	GetTeamMembers(ctx context.Context, teamName string) ([]teamModel.TeamMember, error)
}

type repository struct {
	db *gorm.DB
}

// New creates a new team repository instance.
func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create creates a new team.
func (r *repository) Create(ctx context.Context, teamName string) (*teamModel.Team, error) {
	now := time.Now()
	team := &teamModel.Team{
		TeamName:  teamName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := r.db.WithContext(ctx).Create(team).Error
	if err != nil {
		// Check for unique constraint violation
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateError(err) {
			return nil, teamModel.ErrTeamExists
		}
		return nil, err
	}

	return team, nil
}

// isDuplicateError checks if error is a duplicate key error.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL duplicate key error code
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		(err.Error() != "" && (
		// Check for common duplicate key error messages
		contains(err.Error(), "duplicate key") ||
			contains(err.Error(), "UNIQUE constraint")))
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

// GetByName finds team by team_name.
func (r *repository) GetByName(ctx context.Context, teamName string) (*teamModel.Team, error) {
	var team teamModel.Team
	err := r.db.WithContext(ctx).
		Where("team_name = ?", teamName).
		First(&team).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, teamModel.ErrTeamNotFound
		}
		return nil, err
	}

	return &team, nil
}

// CreateOrUpdateUser creates or updates a user in the team.
// Uses atomic OnConflict to prevent race conditions.
func (r *repository) CreateOrUpdateUser(
	ctx context.Context,
	teamName, userID, username string,
	isActive bool,
) (*userModel.User, error) {
	now := time.Now()
	user := &userModel.User{
		UserID:    userID,
		Username:  username,
		TeamName:  teamName,
		IsActive:  isActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use OnConflict for atomic upsert to prevent race conditions
	// On conflict, update only: username, team_name, is_active, updated_at
	// created_at is preserved from original insert
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"username":   username,
				"team_name":  teamName,
				"is_active":  isActive,
				"updated_at": now,
			}),
		}).
		Create(user).Error

	if err != nil {
		return nil, err
	}

	// Fetch the user to return complete data (including created_at if it was a new record)
	err = r.db.WithContext(ctx).Where("user_id = ?", userID).First(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetTeamMembers returns all members of a team.
func (r *repository) GetTeamMembers(ctx context.Context, teamName string) ([]teamModel.TeamMember, error) {
	var members []teamModel.TeamMember

	err := r.db.WithContext(ctx).
		Table("users").
		Select("user_id, username, is_active").
		Where("team_name = ?", teamName).
		Order("user_id ASC").
		Scan(&members).Error

	if err != nil {
		return nil, err
	}

	if members == nil {
		return []teamModel.TeamMember{}, nil
	}

	return members, nil
}
