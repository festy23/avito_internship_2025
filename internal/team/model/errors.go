package model

import "errors"

var (
	// ErrTeamExists indicates that a team with the given name already exists.
	ErrTeamExists = errors.New("team already exists")
	// ErrTeamNotFound indicates that the requested team does not exist.
	ErrTeamNotFound = errors.New("team not found")
	// ErrInvalidTeamName indicates that the provided team name is invalid (e.g., empty).
	ErrInvalidTeamName = errors.New("invalid team name")
	// ErrEmptyMembers indicates that the members list is empty.
	ErrEmptyMembers = errors.New("members list cannot be empty")
)

