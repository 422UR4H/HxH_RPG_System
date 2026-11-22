package characterclass

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNoSkillDistribution         = domain.NewValidationError(errtext.Sentence("A classe não tem distribuição de perícias."))
	ErrSkillsCountMismatch         = domain.NewValidationError(errtext.Sentence("A quantidade de perícias não confere com a da classe."))
	ErrSkillNotAllowed             = domain.NewValidationError(errtext.Sentence("Essa perícia não é permitida na classe."))
	ErrSkillsPointsMismatch        = domain.NewValidationError(errtext.Sentence("Os pontos de perícia não conferem com os da classe."))
	ErrNoProficiencyDistribution   = domain.NewValidationError(errtext.Sentence("A classe não tem distribuição de proficiências."))
	ErrProficienciesCountMismatch  = domain.NewValidationError(errtext.Sentence("A quantidade de proficiências não confere com a da classe."))
	ErrProficiencyNotAllowed       = domain.NewValidationError(errtext.Sentence("Essa proficiência não é permitida na classe."))
	ErrProficienciesPointsMismatch = domain.NewValidationError(errtext.Sentence("Os pontos de proficiência não conferem com os da classe."))
)

// Helper functions to add context to errors
func NewNoSkillDistributionError(className string) error {
	return errtext.Wrap(ErrNoSkillDistribution, "A classe "+className+" não tem distribuição de perícias.")
}

func NewSkillsCountMismatchError(className string) error {
	return errtext.Wrap(ErrSkillsCountMismatch, "A quantidade de perícias não confere com a da classe "+className+".")
}

func NewSkillNotAllowedError(skillName, className string) error {
	return errtext.Wrap(ErrSkillNotAllowed, "A perícia "+skillName+" não é permitida na classe "+className+".")
}

func NewSkillsPointsMismatchError(className string) error {
	return errtext.Wrap(ErrSkillsPointsMismatch, "Os pontos de perícia não conferem com os da classe "+className+".")
}

func NewNoProficiencyDistributionError(className string) error {
	return errtext.Wrap(ErrNoProficiencyDistribution, "A classe "+className+" não tem distribuição de proficiências.")
}

func NewProficienciesCountMismatchError(className string) error {
	return errtext.Wrap(ErrProficienciesCountMismatch, "A quantidade de proficiências não confere com a da classe "+className+".")
}

func NewProficiencyNotAllowedError(profName, className string) error {
	return errtext.Wrap(ErrProficiencyNotAllowed, "A proficiência "+profName+" não é permitida na classe "+className+".")
}

func NewProficienciesPointsMismatchError(className string) error {
	return errtext.Wrap(ErrProficienciesPointsMismatch, "Os pontos de proficiência não conferem com os da classe "+className+".")
}
