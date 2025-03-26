package attribute

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrPrimaryAttributeNotFound = domain.NewDomainError(errors.New("primary attribute not found"))
	ErrAttributeNotFound        = domain.NewDomainError(errors.New("attribute not found"))
)
