package spiritual

import (
	"errors"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNenHexAlreadyInitialized    = domain.NewDomainError(errtext.Sentence("O hexágono de Nen já foi iniciado."))
	ErrNenHexNotInitialized        = domain.NewDomainError(errtext.Sentence("O hexágono de Nen ainda não foi iniciado."))
	ErrInvalidCategoryPercents     = domain.NewDomainError(errtext.Sentence("Os percentuais das categorias devem ser 6."))
	ErrCategoryNotFound            = domain.NewDomainError(errtext.Sentence("Categoria não encontrada."))
	ErrPrincipleNotFound           = domain.NewDomainError(errtext.Sentence("Princípio não encontrado."))
	ErrFailedToSetCategoryPercents = domain.NewDomainError(errors.New("failed to set category percents"))
)
