package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrProficiencyAlreadyInitialized = domain.NewDomainError(errtext.Sentence("As proficiências já foram iniciadas."))
	ErrPhysSkillsCannotBeNil         = domain.NewDomainError(errtext.Sentence("A experiência das perícias físicas não pode ser vazia."))
	ErrProficiencyAlreadyExists      = domain.NewValidationError(errtext.Sentence("Essa proficiência já existe."))
	ErrProficiencyNotFound           = domain.NewValidationError(errtext.Sentence("Proficiência não encontrada."))
)
