package campaign

import "errors"

var (
	ErrEmptyName          = errors.New("name cannot be empty")
	ErrMaxNameLength      = errors.New("name cannot exceed 32 characters")
	ErrInvalidStartDate   = errors.New("story start date cannot be empty")
	ErrMaxBriefDescLength = errors.New("brief description cannot exceed 64 characters")
)
