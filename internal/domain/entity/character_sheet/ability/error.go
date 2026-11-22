package ability

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrAbilityNotFound = domain.NewDomainError(errtext.Sentence("Aptidão não encontrada."))
)
