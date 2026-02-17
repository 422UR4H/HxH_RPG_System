package turn

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrReactionNotCompatible = domain.NewDomainError(errors.New("reaction is not compatible with the current action"))
	ErrTurnIsEmpty           = domain.NewDomainError(errors.New("current turn has no actions"))
)
