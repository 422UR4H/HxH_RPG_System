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
	// ErrTurnAlreadyClosed means the turn the caller tried to open a reaction on has already
	// finished — there is no one left to narrate for.
	ErrTurnAlreadyClosed = errors.New("cannot open a reaction: turn already closed")
	// ErrReactionNotFound means the given id is not attached to the open turn.
	ErrReactionNotFound = errors.New("reaction not found on the current turn")
	// ErrNoOpenTurn is close_turn with nothing under the baton. It is an error rather than a
	// no-op because the master pressed a button that describes an action they believe is
	// happening; answering silently would leave them believing it happened.
	ErrNoOpenTurn = errors.New("no open turn to close")
	// ErrActionNotOnTurn means a condition edit named an ActionID that is neither the open
	// turn's own action nor one of its attached reactions.
	ErrActionNotOnTurn = errors.New("action is not on the open turn")
	// ErrConditionTargetMissing means a condition edit named a Field or SkillName that names
	// nothing present on the action — a client bug and never a no-op: the master pressed a
	// control describing a test they believe exists, and answering silently would leave them
	// believing they changed it.
	ErrConditionTargetMissing = errors.New("condition edit targets a check that is not on this action")
	// ErrAmbiguousConditionEdit means a condition edit set both Field and SkillName. They are
	// alternatives, never both at once.
	ErrAmbiguousConditionEdit = errors.New("condition edit must set either field or skillName, not both")
)
