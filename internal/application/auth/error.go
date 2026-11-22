package auth

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrUnauthorized            = domain.NewValidationError(errtext.Sentence("E-mail ou senha incorretos."))
	ErrInsufficientPermissions = domain.NewValidationError(errtext.Sentence("Você não tem permissão para fazer isso."))
)
