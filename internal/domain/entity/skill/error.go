package skill

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrSkillsAlreadyInitialized = domain.NewDomainError(errors.New("skills already initialized"))
	ErrJointSkillNotInitialized = domain.NewDomainError(errors.New("joint skill not initialized"))
	ErrJointSkillAlreadyExists  = domain.NewDomainError(errors.New("joint skill already exists"))
	ErrSkillNotFound            = domain.NewDomainError(errors.New("skill not found"))

	ErrAbilitySkillsAlreadyInitialized = domain.NewDomainError(errors.New("ability skills exp already initialized"))
	ErrAbilitySkillsCannotBeNil        = domain.NewDomainError(errors.New("ability skills exp cannot be nil"))
)
