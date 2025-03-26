package sheet

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrInvalidActiveCategoryCount = domain.NewDomainError(errors.New("at least one category must be active"))
)
