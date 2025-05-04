package scenario

import "errors"

var (
	ErrEmptyName          = errors.New("name cannot be empty")
	ErrMaxNameLength      = errors.New("name cannot exceed 32 characters")
	ErrMaxBriefDescLength = errors.New("brief description cannot exceed 64 characters")
)
