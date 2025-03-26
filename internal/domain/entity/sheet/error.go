package sheet

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrInvalidActiveCategoryCount = domain.NewValidationError(errors.New("at least one category must be active"))
	ErrInvalidNicknameLength      = domain.NewValidationError(errors.New("invalid nickname length"))
	ErrInvalidFullNameLength      = domain.NewValidationError(errors.New("invalid fullname length"))
	ErrInvalidAlignmentLength     = domain.NewValidationError(errors.New("invalid alignment length"))
	ErrInvalidBriefDescription    = domain.NewValidationError(errors.New("invalid brief description length"))
	ErrInvalidBirthday            = domain.NewValidationError(errors.New("invalid birthday"))
)

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
