package item

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrWeaponNotFound = domain.NewDomainError(errors.New("weapon not found"))
)
