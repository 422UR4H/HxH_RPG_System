package skill

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrSkillsAlreadyInitialized = domain.NewDomainError(errtext.Sentence("As perícias já foram iniciadas."))
	ErrJointSkillNotInitialized = domain.NewDomainError(errtext.Sentence("A perícia conjunta ainda não foi iniciada."))

	ErrAbilitySkillsAlreadyInitialized = domain.NewDomainError(errtext.Sentence("A experiência das perícias da aptidão já foi iniciada."))
	ErrAbilitySkillsCannotBeNil        = domain.NewDomainError(errtext.Sentence("A experiência das perícias da aptidão não pode ser vazia."))
	ErrSkillNotFound                   = domain.NewDomainError(errtext.Sentence("Perícia não encontrada."))

	ErrJointSkillAlreadyExists = domain.NewValidationError(errtext.Sentence("Essa perícia conjunta já existe."))
)
