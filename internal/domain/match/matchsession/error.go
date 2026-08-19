package matchsession

import "errors"

var (
	ErrParticipantNotFound     = errors.New("participant not found in match session")
	ErrActionActorMismatch     = errors.New("action actor does not match player")
	ErrRoundHasOpenTurn        = errors.New("cannot close round: current turn is still open")
	ErrCharSheetNotFound       = errors.New("character sheet not found in session")
	ErrNoActiveTurn            = errors.New("no active turn in current round")
	ErrCharacterStatusNotFound = errors.New("character status not found in session")
	// ErrReactionActorMismatch means the caller tried to react through a character that is
	// not theirs. The same rule EnqueueAction enforces, on the other route.
	ErrReactionActorMismatch = errors.New("the reacting character does not belong to this player")
	// ErrReactorNotTargeted means the caller was not aimed at. Bystanders watch.
	ErrReactorNotTargeted = errors.New("only a target of the open action may react to it")
)
