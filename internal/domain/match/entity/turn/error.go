package turn

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrReactionNotCompatible = domain.NewDomainError(errtext.Sentence("Essa reação não serve para a ação atual."))
	ErrTurnIsEmpty           = domain.NewDomainError(errtext.Sentence("O turno atual não tem ações."))
)
