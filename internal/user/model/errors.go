package model

import "errors"

var (
	// ErrUserNotFound indicates that the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidUserID indicates that the provided user ID is invalid (e.g., empty).
	ErrInvalidUserID = errors.New("invalid user ID")
	// ErrInvalidIsActive indicates that is_active field is missing or invalid.
	ErrInvalidIsActive = errors.New("is_active field is required")
)
