package match

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchNotFound      = domain.NewValidationError(errors.New("match not found"))
	ErrMinTitleLength     = domain.NewValidationError(errors.New("title must be at least 5 characters"))
	ErrMaxTitleLength     = domain.NewValidationError(errors.New("title cannot exceed 32 characters"))
	ErrMinOfStoryStartAt  = domain.NewValidationError(errors.New("story start date must be after campaign start date"))
	ErrMaxOfStoryStartAt  = domain.NewValidationError(errors.New("story start date must be before campaign end date"))
	ErrMinOfGameStartAt   = domain.NewValidationError(errors.New("game start at cannot be in the past"))
	ErrMaxOfGameStartAt   = domain.NewValidationError(errors.New("game start at cannot be greater than one year from now"))
	ErrMaxBriefDescLength = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
