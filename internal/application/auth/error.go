package auth

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrUnauthorized            = domain.NewValidationError(errors.New("access denied"))
	ErrInsufficientPermissions = domain.NewValidationError(errors.New("insufficient permissions"))
)
