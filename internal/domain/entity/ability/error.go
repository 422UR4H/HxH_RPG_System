package ability

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrAbilityNotFound = domain.NewDomainError(errors.New("ability not found"))
)
