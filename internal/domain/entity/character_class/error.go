package characterclass

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNoSkillDistribution         = domain.NewDomainError(errors.New("character class has no skill distribution"))
	ErrSkillsCountMismatch         = domain.NewDomainError(errors.New("skills count mismatch"))
	ErrSkillNotAllowed             = domain.NewDomainError(errors.New("skill not allowed in character class"))
	ErrSkillsPointsMismatch        = domain.NewDomainError(errors.New("skills points mismatch"))
	ErrNoProficiencyDistribution   = domain.NewDomainError(errors.New("character class has no proficiency distribution"))
	ErrProficienciesCountMismatch  = domain.NewDomainError(errors.New("proficiencies count mismatch"))
	ErrProficiencyNotAllowed       = domain.NewDomainError(errors.New("proficiency not allowed in character class"))
	ErrProficienciesPointsMismatch = domain.NewDomainError(errors.New("proficiencies points mismatch"))
)

// Helper functions to add context to errors
func NewNoSkillDistributionError(className string) error {
	return fmt.Errorf("%w: %s", ErrNoSkillDistribution, className)
}

func NewSkillsCountMismatchError(className string) error {
	return fmt.Errorf("%w: %s", ErrSkillsCountMismatch, className)
}

func NewSkillNotAllowedError(skillName, className string) error {
	return fmt.Errorf("%w: skill %s not allowed in character class %s", ErrSkillNotAllowed, skillName, className)
}

func NewSkillsPointsMismatchError(className string) error {
	return fmt.Errorf("%w: %s", ErrSkillsPointsMismatch, className)
}

func NewNoProficiencyDistributionError(className string) error {
	return fmt.Errorf("%w: %s", ErrNoProficiencyDistribution, className)
}

func NewProficienciesCountMismatchError(className string) error {
	return fmt.Errorf("%w: %s", ErrProficienciesCountMismatch, className)
}

func NewProficiencyNotAllowedError(profName, className string) error {
	return fmt.Errorf("%w: proficiency %s not allowed in character class %s", ErrProficiencyNotAllowed, profName, className)
}

func NewProficienciesPointsMismatchError(className string) error {
	return fmt.Errorf("%w: %s", ErrProficienciesPointsMismatch, className)
}
