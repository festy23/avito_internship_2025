// Package repository provides data access layer for team module.
package repository

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

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
	CreateOrUpdateUser(
		ctx context.Context,
		teamName, userID, username string,
		isActive bool,
	) (*userModel.User, error)

	// GetTeamMembers returns all members of a team.
	GetTeamMembers(ctx context.Context, teamName string) ([]teamModel.TeamMember, error)
}

type repository struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new team repository instance.
func New(db *gorm.DB, logger *zap.SugaredLogger) Repository {
	return &repository{db: db, logger: logger}
}

// Create creates a new team.
func (r *repository) Create(ctx context.Context, teamName string) (*teamModel.Team, error) {
	r.logger.Infow("Creating team", "team_name", teamName)

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
			r.logger.Debugw("Create team duplicate key", "team_name", teamName)
			return nil, teamModel.ErrTeamExists
		}
		r.logger.Errorw("Failed to create team", "team_name", teamName, "error", err)
		return nil, err
	}

	r.logger.Infow("Team created", "team_name", teamName)
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
	r.logger.Debugw("GetByName called", "team_name", teamName)

	var team teamModel.Team
	err := r.db.WithContext(ctx).
		Where("team_name = ?", teamName).
		First(&team).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debugw("GetByName team not found", "team_name", teamName)
			return nil, teamModel.ErrTeamNotFound
		}
		r.logger.Errorw("GetByName database error", "team_name", teamName, "error", err)
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
	r.logger.Infow("CreateOrUpdateUser called", "team_name", teamName, "user_id", userID, "is_active", isActive)

	now := time.Now()
	user := &userModel.User{
		UserID:    userID,
		Username:  username,
		TeamName:  teamName,
		IsActive:  isActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use raw SQL for atomic upsert to prevent race conditions
	// Why raw SQL instead of GORM OnConflict?
	// 1. SQLite applies DEFAULT value from schema (DEFAULT TRUE) even when GORM explicitly sets IsActive=false
	// 2. GORM's OnConflict with clause.Assignments doesn't override SQLite's DEFAULT constraint
	// 3. Raw SQL allows explicit value that bypasses DEFAULT constraint
	// 4. This is a known limitation when using GORM with SQLite and DEFAULT values
	// Note: GORM handles boolean-to-INTEGER conversion automatically for SQLite
	err := r.db.WithContext(ctx).
		Exec("INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET username = ?, team_name = ?, is_active = ?, updated_at = ?",
			userID, username, teamName, isActive, now, now,
			username, teamName, isActive, now).
		Error

	if err != nil {
		r.logger.Errorw("CreateOrUpdateUser database error", "team_name", teamName, "user_id", userID, "error", err)
		return nil, err
	}

	// Fetch the user to return complete data (including created_at if it was a new record)
	// Use the same db connection (which may be a transaction) to ensure consistency
	err = r.db.WithContext(ctx).Where("user_id = ?", userID).First(user).Error
	if err != nil {
		r.logger.Errorw("CreateOrUpdateUser failed to fetch user", "user_id", userID, "error", err)
		return nil, err
	}

	r.logger.Infow("CreateOrUpdateUser completed", "team_name", teamName, "user_id", userID)
	return user, nil
}

// GetTeamMembers returns all members of a team.
func (r *repository) GetTeamMembers(
	ctx context.Context,
	teamName string,
) ([]teamModel.TeamMember, error) {
	r.logger.Debugw("GetTeamMembers called", "team_name", teamName)

	var members []teamModel.TeamMember

	err := r.db.WithContext(ctx).
		Table("users").
		Select("user_id, username, is_active").
		Where("team_name = ?", teamName).
		Order("user_id ASC").
		Scan(&members).Error

	if err != nil {
		r.logger.Errorw("GetTeamMembers database error", "team_name", teamName, "error", err)
		return nil, err
	}

	if members == nil {
		members = []teamModel.TeamMember{}
	}

	r.logger.Debugw("GetTeamMembers completed", "team_name", teamName, "member_count", len(members))
	return members, nil
}
