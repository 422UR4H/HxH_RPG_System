package scenario

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrScenarioNotFound          = domain.NewDBError(errtext.Sentence("Cenário não encontrado."))
	ErrScenarioNameAlreadyExists = domain.NewValidationError(errtext.Sentence("Já existe um cenário com esse nome."))
	ErrMinNameLength             = domain.NewValidationError(errtext.Sentence("O nome deve ter pelo menos 5 caracteres."))
	ErrMaxNameLength             = domain.NewValidationError(errtext.Sentence("O nome deve ter no máximo 32 caracteres."))
	ErrMaxBriefDescLength        = domain.NewValidationError(errtext.Sentence("A descrição curta deve ter no máximo 64 caracteres."))
)
