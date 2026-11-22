package match

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchNotFound        = domain.NewValidationError(errtext.Sentence("Partida não encontrada."))
	ErrMinTitleLength       = domain.NewValidationError(errtext.Sentence("O título deve ter pelo menos 5 caracteres."))
	ErrMaxTitleLength       = domain.NewValidationError(errtext.Sentence("O título deve ter no máximo 32 caracteres."))
	ErrMinOfStoryStartAt    = domain.NewValidationError(errtext.Sentence("A data de início da história deve ser depois do início da campanha."))
	ErrMaxOfStoryStartAt    = domain.NewValidationError(errtext.Sentence("A data de início da história deve ser antes do fim da campanha."))
	ErrMinOfGameScheduledAt = domain.NewValidationError(errtext.Sentence("A data marcada da partida não pode estar no passado."))
	ErrMaxOfGameScheduledAt = domain.NewValidationError(errtext.Sentence("A data marcada da partida não pode passar de um ano a partir de hoje."))
	ErrMaxBriefDescLength   = domain.NewValidationError(errtext.Sentence("A descrição curta deve ter no máximo 255 caracteres."))
	ErrMatchAlreadyStarted  = domain.NewValidationError(errtext.Sentence("A partida já começou."))
	ErrMatchAlreadyFinished = domain.NewValidationError(errtext.Sentence("A partida já terminou."))
	ErrNotMatchMaster       = domain.NewValidationError(errtext.Sentence("Só o mestre da partida pode fazer isso."))
	// ErrRoundHasNoTurns: ending a round in which no turn ever opened. A round closes on what
	// happened in it; ending empty rounds one after another would only spin the clock.
	ErrRoundHasNoTurns  = domain.NewValidationError(errtext.Sentence("Ainda não há turno aberto neste round."))
	ErrInvalidRoundMode = domain.NewValidationError(errtext.Sentence("O modo do round deve ser livre ou corrida."))

	ErrSheetNotNPC            = domain.NewValidationError(errtext.Sentence("Essa ficha não é de um NPC."))
	ErrSheetNotOwnedByMaster  = domain.NewValidationError(errtext.Sentence("Essa ficha não pertence ao mestre nem à campanha."))
	ErrNPCAlreadyInMatch      = domain.NewValidationError(errtext.Sentence("Esse NPC já está na partida."))
	ErrNPCNotInMatch          = domain.NewValidationError(errtext.Sentence("Esse NPC não está na partida."))
	ErrCharacterSheetNotFound = domain.NewValidationError(errtext.Sentence("Ficha não encontrada."))
)
