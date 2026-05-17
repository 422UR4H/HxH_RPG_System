package enum

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrInvalidNameOf = domain.NewValidationError(errors.New("invalid name of "))
)

func newInvalidNameOfError(kind, value string) error {
	return fmt.Errorf("%w%s: %s", ErrInvalidNameOf, kind, value)
}
