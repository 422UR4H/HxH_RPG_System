package proficiency

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrProficiencyAlreadyInitialized = domain.NewDomainError(errors.New("proficiency already initialized"))
	ErrPhysSkillsCannotBeNil         = domain.NewDomainError(errors.New("physical skill exp cannot be nil"))
	ErrProficiencyAlreadyExists      = domain.NewValidationError(errors.New("proficiency already exists"))
	ErrProficiencyNotFound           = domain.NewValidationError(errors.New("proficiency not found"))
)
