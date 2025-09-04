package sheet

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

const invalidOwnerMsg = "the owner must be a player (user) OR a scenario, exclusively (XOR)"
const invalidAlignment = "alignment must be: '{Lawful, Neutral, Chaotic}-{Good, Neutral, Evil}'"

var (
	ErrClassNotApplied            = domain.NewDomainError(errors.New("class not applied"))
	ErrInvalidOwner               = domain.NewValidationError(errors.New(invalidOwnerMsg))
	ErrInvalidAlignment           = domain.NewValidationError(errors.New(invalidAlignment))
	ErrInvalidActiveCategoryCount = domain.NewValidationError(errors.New("at least one category must be active"))
	ErrInvalidNicknameLength      = domain.NewValidationError(errors.New("nickname must be 3-16 characters long"))
	ErrInvalidFullNameLength      = domain.NewValidationError(errors.New("fullname must be 6-32 characters long"))
	ErrInvalidBriefDescription    = domain.NewValidationError(errors.New("brief description must be at most 32 characters long"))
	ErrInvalidBirthday            = domain.NewValidationError(errors.New("invalid birthday"))
	ErrInvalidAge                 = domain.NewValidationError(errors.New("age must be positive"))
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

func NewInvalidAlignmentError(alignment string) error {
	return fmt.Errorf("%w: %s", ErrInvalidAlignment, alignment)
}

func NewInvalidBriefDescriptionError(desc string) error {
	return fmt.Errorf("%w: %s", ErrInvalidBriefDescription, desc)
}

func NewInvalidBirthdayError() error {
	return ErrInvalidBirthday
}

func NewInvalidAgeError() error {
	return ErrInvalidAge
}
