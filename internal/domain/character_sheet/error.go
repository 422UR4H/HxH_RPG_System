package charactersheet

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterSheetNotFound    = domain.NewDBError(errors.New("character sheet not found"))
	ErrCharacterClassNotFound    = domain.NewDBError(errors.New("character class not found"))
	ErrNicknameNotAllowed        = domain.NewValidationError(errors.New("nickname is not allowed"))
	ErrNicknameAlreadyExists     = domain.NewValidationError(errors.New("nickname already exists"))
	ErrNenHexNotInitialized      = domain.NewValidationError(errors.New("nen hexagon value not initialized"))
	ErrInvalidUpdateHexValMethod = domain.NewDomainError(errors.New("invalid update nen hexagon value method"))
)

func NewCharacterClassNotFoundError(className string) error {
	return fmt.Errorf("%w: %s", ErrCharacterClassNotFound, className)
}

func NewNicknameNotAllowedError(nickname string) error {
	return fmt.Errorf("%w: %s", ErrNicknameNotAllowed, nickname)
}

func NewNicknameAlreadyExistsError(nickname string) error {
	return fmt.Errorf("%w: %s", ErrNicknameAlreadyExists, nickname)
}
