package domain

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrTeamExists   = errors.New("team already exists")
	ErrPRExists     = errors.New("pull request already exists")
	ErrPRMerged     = errors.New("pull request is merged")
	ErrNotAssigned  = errors.New("reviewer not assigned")
	ErrNoCandidate  = errors.New("no candidate available")
	ErrPRNotFound   = errors.New("pull request not found")
	ErrTeamNotFound = errors.New("team not found")
	ErrUserNotFound = errors.New("user not found")
)
