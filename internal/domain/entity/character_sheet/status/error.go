package status

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrStatusNotFound = domain.NewDomainError(errors.New("status not found"))
	ErrSpiritualIsNil = domain.NewDomainError(errors.New("spiritual ability is nil"))
	ErrInvalidValue   = domain.NewDomainError(errors.New("invalid value for status"))
)
