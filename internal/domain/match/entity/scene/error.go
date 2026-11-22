package scene

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrSceneIsFinished = domain.NewDomainError(errtext.Sentence("A cena já terminou."))
)
