package item

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrWeaponNotFound = domain.NewDomainError(errtext.Sentence("Arma não encontrada."))
)
