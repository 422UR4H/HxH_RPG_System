// internal/domain/cancelledaction/record.go
package cancelledaction

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Record is an action its declarer took back out of the queue before it opened (cancel_action),
// as persisted (spec 2026-10-10 §8.1 P6). It has a table of its own, not actions: actions
// requires a turn, and a cancelled action never had one — the same reason master_actions exists.
//
// Like a master action, it is written the instant it happened when no turn is open, and with the
// close of the turn that was open then, in that turn's own transaction, otherwise (P7) — its
// fields are still those of the instant it was cancelled.
type Record struct {
	// UUID is the cancelled action's own id: the action and its record are one thing.
	UUID      uuid.UUID
	MatchUUID uuid.UUID
	// SceneUUID and RoundUUID are where the action would have happened: the active pair at the
	// instant it was cancelled.
	SceneUUID uuid.UUID
	RoundUUID uuid.UUID
	// TurnUUID is the turn open at the instant of the cancellation, nil when none was. No FK, as
	// in master_actions: a held record is written in that turn's own transaction anyway.
	TurnUUID *uuid.UUID
	// ActorUUID is the character sheet that would have acted.
	ActorUUID uuid.UUID
	// CancelledBy is the USER who cancelled — the actor's player, or the master for an NPC.
	CancelledBy uuid.UUID
	// Action is the action as it stood in the queue when it was cancelled, dice already rolled.
	Action      action.Action
	CancelledAt time.Time
}
