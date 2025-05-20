package campaign

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrEmptyName          = domain.NewValidationError(errors.New("name cannot be empty"))
	ErrMaxNameLength      = domain.NewValidationError(errors.New("name cannot exceed 32 characters"))
	ErrInvalidStartDate   = domain.NewValidationError(errors.New("story start date cannot be empty"))
	ErrMaxBriefDescLength = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
