package match

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchIsNil = domain.NewValidationError(errtext.Sentence("Partida não encontrada."))
)
