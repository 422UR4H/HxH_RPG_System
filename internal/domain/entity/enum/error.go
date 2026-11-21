package enum

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrInvalidNameOf = domain.NewValidationError(errtext.Sentence("Valor inválido."))
)

func newInvalidNameOfError(kind, value string) error {
	return errtext.Wrap(ErrInvalidNameOf, "Valor inválido para "+kind+": "+value+".")
}
