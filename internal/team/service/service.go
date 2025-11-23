// Package service provides business logic layer for team module.
package service

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	teamModel "github.com/festy23/avito_internship/internal/team/model"
	"github.com/festy23/avito_internship/internal/team/repository"
)

// Service defines the interface for team business logic operations.
type Service interface {
	// AddTeam creates a new team with members.
	AddTeam(ctx context.Context, req *teamModel.AddTeamRequest) (*teamModel.TeamResponse, error)

	// GetTeam returns a team with its members.
	GetTeam(ctx context.Context, teamName string) (*teamModel.TeamResponse, error)
}

type service struct {
	repo   repository.Repository
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// New creates a new team service instance.
func New(repo repository.Repository, db *gorm.DB, logger *zap.SugaredLogger) Service {
	return &service{
		repo:   repo,
		db:     db,
		logger: logger,
	}
}

// AddTeam creates a new team with members in a transaction.
func (s *service) AddTeam(ctx context.Context, req *teamModel.AddTeamRequest) (*teamModel.TeamResponse, error) {
	// Validate input
	if req.TeamName == "" {
		return nil, teamModel.ErrInvalidTeamName
	}

	if len(req.Members) == 0 {
		return nil, teamModel.ErrEmptyMembers
	}

	// Use transaction to ensure atomicity
	var result *teamModel.TeamResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create repository with transaction
		txRepo := repository.New(tx, s.logger)

		// Create team
		_, err := txRepo.Create(ctx, req.TeamName)
		if err != nil {
			return err
		}

		// Create or update all members
		for _, member := range req.Members {
			if member.UserID == "" {
				continue // Skip members with empty user_id
			}
			_, err = txRepo.CreateOrUpdateUser(ctx, req.TeamName, member.UserID, member.Username, member.IsActive)
			if err != nil {
				return err
			}
		}

		// Fetch team members
		members, err := txRepo.GetTeamMembers(ctx, req.TeamName)
		if err != nil {
			return err
		}

		result = &teamModel.TeamResponse{
			TeamName: req.TeamName,
			Members:  members,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTeam returns a team with its members.
func (s *service) GetTeam(ctx context.Context, teamName string) (*teamModel.TeamResponse, error) {
	if teamName == "" {
		return nil, teamModel.ErrInvalidTeamName
	}

	// Check if team exists
	_, err := s.repo.GetByName(ctx, teamName)
	if err != nil {
		return nil, err
	}

	// Get team members
	members, err := s.repo.GetTeamMembers(ctx, teamName)
	if err != nil {
		return nil, err
	}

	return &teamModel.TeamResponse{
		TeamName: teamName,
		Members:  members,
	}, nil
}
