package match

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchIsNil = domain.NewValidationError(errors.New("match cannot be nil"))
)
