package proficiency

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrProficiencyAlreadyInitialized = domain.NewDomainError(errors.New("proficiency already initialized"))
	ErrProficiencyAlreadyExists      = domain.NewDomainError(errors.New("proficiency already exists"))
	ErrPhysSkillsCannotBeNil         = domain.NewDomainError(errors.New("physical skill exp cannot be nil"))
	ErrProficiencyNotFound           = domain.NewDomainError(errors.New("proficiency not found"))
)
