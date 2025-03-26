package enum

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrInvalidNameOf = domain.NewDomainError(errors.New("invalid name of "))
)
