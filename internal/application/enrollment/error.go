package enrollment

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterNotInCampaign   = domain.NewValidationError(errtext.Sentence("Esse personagem não pertence à campanha da partida."))
	ErrCharacterAlreadyEnrolled = domain.NewValidationError(errtext.Sentence("Esse personagem já está inscrito na partida."))
	ErrEnrollmentNotFound       = domain.NewValidationError(errtext.Sentence("Inscrição não encontrada."))
	ErrNotMatchMaster           = domain.NewValidationError(errtext.Sentence("Só o mestre da campanha pode fazer isso."))
	ErrMatchAlreadyStarted      = domain.NewValidationError(errtext.Sentence("A partida já começou."))
	ErrMatchAlreadyFinished     = domain.NewValidationError(errtext.Sentence("A partida já terminou."))
	ErrPlayerNotEnrolled        = domain.NewValidationError(errtext.Sentence("Esse jogador não está inscrito na partida."))
)
