package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

var (
	ErrQueueEmpty            = errtext.Sentence("A fila de ações está vazia.")
	ErrActionNotFound        = errtext.Sentence("Essa ação não está mais na fila.")
	ErrNoCurrentTurn         = errtext.Sentence("Não há turno neste round.")
	ErrReactionNotCompatible = errtext.Sentence("Essa reação não é para a ação atual.")
	// ErrTurnAlreadyClosed means RoundOrchestrator.AttachReaction was asked to attach to a
	// turn whose finishedAt is already set. This is the orchestrator's own copy of the check
	// matchsession.ErrTurnAlreadyClosed enforces first, on the same condition: the
	// orchestrator does not rely on MatchSession to keep it honest, since it is the one that
	// actually calls Turn.AddReaction.
	ErrTurnAlreadyClosed = errtext.Sentence("O turno já fechou: não dá para reagir.")
)
