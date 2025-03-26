package charactersheet

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterClassNotFound = domain.NewValidationError(errors.New("character class not found"))
	ErrNicknameNotAllowed     = domain.NewValidationError(errors.New("nickname is not allowed"))
	ErrNicknameAlreadyExists  = domain.NewValidationError(errors.New("nickname already exists"))
)

// Helper functions to add context to errors
func NewCharacterClassNotFoundError(className string) error {
	return fmt.Errorf("%w: %s", ErrCharacterClassNotFound, className)
}

func NewNicknameNotAllowedError(nickname string) error {
	return fmt.Errorf("%w: %s", ErrNicknameNotAllowed, nickname)
}

func NewNicknameAlreadyExistsError(nickname string) error {
	return fmt.Errorf("%w: %s", ErrNicknameAlreadyExists, nickname)
}
