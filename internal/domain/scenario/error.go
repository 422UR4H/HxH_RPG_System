package scenario

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrScenarioNotFound          = domain.NewDBError(errors.New("scenario not found"))
	ErrScenarioNameAlreadyExists = domain.NewValidationError(errors.New("scenario name already exists"))
	ErrMinNameLength             = domain.NewValidationError(errors.New("name must be at least 5 characters"))
	ErrMaxNameLength             = domain.NewValidationError(errors.New("name cannot exceed 32 characters"))
	ErrMaxBriefDescLength        = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
