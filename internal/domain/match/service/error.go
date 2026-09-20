package service

import "errors"

var (
	ErrQueueEmpty            = errors.New("action queue is empty")
	ErrActionNotFound        = errors.New("action not found in queue")
	ErrNoCurrentTurn         = errors.New("no current turn in round")
	ErrReactionNotCompatible = errors.New("reaction does not target the current action")
	// ErrTurnAlreadyClosed means RoundOrchestrator.AttachReaction was asked to attach to a
	// turn whose finishedAt is already set. This is the orchestrator's own copy of the check
	// matchsession.ErrTurnAlreadyClosed enforces first, on the same condition: the
	// orchestrator does not rely on MatchSession to keep it honest, since it is the one that
	// actually calls Turn.AddReaction.
	ErrTurnAlreadyClosed = errors.New("cannot attach a reaction: turn already closed")
)
