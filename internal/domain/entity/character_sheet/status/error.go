package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrStatusNotFound = domain.NewDomainError(errtext.Sentence("Barra de status não encontrada."))
	ErrSpiritualIsNil = domain.NewDomainError(errtext.Sentence("A aptidão espiritual não foi iniciada."))
	ErrInvalidValue   = domain.NewDomainError(errtext.Sentence("Valor inválido para a barra de status."))
)
