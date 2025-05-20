package scenario

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrScenarioNameAlreadyExists = domain.NewValidationError(errors.New("scenario name already exists"))
	ErrScenarioNotFound          = domain.NewDBError(errors.New("scenario not found"))
)
