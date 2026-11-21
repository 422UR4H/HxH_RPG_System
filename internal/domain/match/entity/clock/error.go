package clock

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNotSet              = domain.NewDomainError(errtext.Sentence("Defina a hora da partida antes."))
	ErrAlreadySet          = domain.NewDomainError(errtext.Sentence("A hora inicial já foi definida."))
	ErrBelowLimit          = domain.NewDomainError(errtext.Sentence("Não dá para voltar antes do limite do relógio."))
	ErrLongSkipUnconfirmed = domain.NewDomainError(errtext.Sentence("Confirme o avanço de mais de 2 dias."))
	ErrZeroDelta           = domain.NewDomainError(errtext.Sentence("Esse ajuste não muda a hora."))
	ErrAlreadyRunning      = domain.NewDomainError(errtext.Sentence("O tempo real já está ligado."))
)
