package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrPrimaryAttributeNotFound = domain.NewDomainError(errtext.Sentence("Atributo primário não encontrado."))
	ErrAttributeNotFound        = domain.NewDomainError(errtext.Sentence("Atributo não encontrado."))
)
