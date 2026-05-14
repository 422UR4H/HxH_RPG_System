package matchsession

import "errors"

var (
	ErrParticipantNotFound = errors.New("participant not found in match session")
	ErrActionActorMismatch = errors.New("action actor does not match player")
	ErrRoundHasOpenTurn    = errors.New("cannot close round: current turn is still open")
	ErrCharSheetNotFound   = errors.New("character sheet not found in session")
	ErrNoActiveTurn        = errors.New("no active turn in current round")
)
