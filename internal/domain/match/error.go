package match

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchNotFound      = domain.NewDBError(errors.New("match not found"))
	ErrMinTitleLength     = domain.NewValidationError(errors.New("title must be at least 5 characters"))
	ErrMaxTitleLength     = domain.NewValidationError(errors.New("title cannot exceed 32 characters"))
	ErrMinOfStartDate     = domain.NewValidationError(errors.New("story start date must be after campaign start date"))
	ErrMaxOfStartDate     = domain.NewValidationError(errors.New("story start date must be before campaign end date"))
	ErrMaxBriefDescLength = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
