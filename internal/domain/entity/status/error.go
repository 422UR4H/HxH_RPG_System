package status

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrStatusNotFound = domain.NewDomainError(errors.New("status not found"))
	ErrInvalidValue   = domain.NewDomainError(errors.New("invalid value for status"))
)
