package match

import "errors"

var (
	ErrEmptyTitle         = errors.New("title cannot be empty")
	ErrMaxTitleLength     = errors.New("title cannot exceed 32 characters")
	ErrMinTitleLength     = errors.New("title must be at least 5 characters")
	ErrMaxBriefDescLength = errors.New("brief description cannot exceed 64 characters")
	ErrInvalidStartDate   = errors.New("story start date cannot be empty")
)
