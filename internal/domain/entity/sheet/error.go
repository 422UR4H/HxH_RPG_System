package sheet

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

const invalidOwnerMsg = "the owner must be a player (user) OR a scenario, exclusively (XOR)"

var (
	ErrClassNotApplied            = domain.NewDomainError(errors.New("class not applied"))
	ErrInvalidOwner               = domain.NewValidationError(errors.New(invalidOwnerMsg))
	ErrInvalidActiveCategoryCount = domain.NewValidationError(errors.New("at least one category must be active"))
	ErrInvalidNicknameLength      = domain.NewValidationError(errors.New("invalid nickname length"))
	ErrInvalidFullNameLength      = domain.NewValidationError(errors.New("invalid fullname length"))
	ErrInvalidAlignmentLength     = domain.NewValidationError(errors.New("invalid alignment length"))
	ErrInvalidBriefDescription    = domain.NewValidationError(errors.New("invalid brief description length"))
	ErrInvalidBirthday            = domain.NewValidationError(errors.New("invalid birthday"))
	ErrInvalidDistributionPoints  = domain.NewValidationError(errors.New("invalid distribution points"))
	ErrCharClassAlreadyExists     = domain.NewValidationError(errors.New("character class already exists"))
)

func NewClassNotAppliedError(msg string) error {
	return fmt.Errorf("%w: %s", ErrClassNotApplied, msg)
}

func NewInvalidNicknameLengthError(nick string) error {
	return fmt.Errorf("%w: %s", ErrInvalidNicknameLength, nick)
}

func NewInvalidFullNameLengthError(name string) error {
	return fmt.Errorf("%w: %s", ErrInvalidFullNameLength, name)
}

func NewInvalidAlignmentLengthError(alignment string) error {
	return fmt.Errorf("%w: %s", ErrInvalidAlignmentLength, alignment)
}

func NewInvalidBriefDescriptionError(desc string) error {
	return fmt.Errorf("%w: %s", ErrInvalidBriefDescription, desc)
}

func NewInvalidBirthdayError() error {
	return ErrInvalidBirthday
}
