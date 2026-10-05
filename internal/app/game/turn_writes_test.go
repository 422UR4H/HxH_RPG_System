package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// recordOpenedReactionMoveViews holds, with the open turn, what each session player saw of an
// opened reaction's destination — keyed by the reaction — until the close writes it with the
// reaction's row (Phase 7, item 1). fogTestRoom's player stands on (0,0) and sees the whole open
// board, so a destination in sight is Full for them.
func TestTurnWrites_RecordOpenedReactionMoveViews(t *testing.T) {
	// openTurn appends a turn to the session's round and returns its id: the turn the
	// reaction is opened in.
	openTurn := func(r *Room) uuid.UUID {
		tr := turnentity.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
		r.session.GetActiveRound().AppendTurn(tr)
		return tr.GetID()
	}
	// escape is a reaction by actorID moving to (2,0), in the player's sight.
	escape := func(actorID uuid.UUID) action.Action {
		return *action.NewAction(actorID, nil, uuid.New(), nil, action.ActionSpeed{}, nil,
			&action.Move{Position: [3]int{2, 0, 0}}, nil, nil, &action.Dodge{}, nil, nil)
	}

	t.Run("a third party who sees the destination is recorded Full", func(t *testing.T) {
		r, playerUUID, _ := fogTestRoom(t)
		reactor := uuid.New() // nobody's: the player is a third party to it
		r.pieces["reactor"] = PieceMovedPayload{PieceID: "reactor", CharacterID: reactor.String(), Slot: squareSlot(1, 0)}
		turnID := openTurn(r)
		react := escape(reactor)

		r.recordOpenedReactionMoveViews(turnID, react)

		views := r.pendingTurns[turnID].reactionMoveViews[react.GetID()]
		if len(views) != 1 || views[playerUUID] != masteraction.ViewFull {
			t.Fatalf("views = %v, want only the player at %q", views, masteraction.ViewFull)
		}
		if _, ok := views[r.masterUUID]; ok {
			t.Fatal("the master was recorded: he sees everything and is never recorded")
		}
	})

	t.Run("the reactor's owner is never recorded", func(t *testing.T) {
		r, playerUUID, _ := fogTestRoom(t)
		// fogTestRoom's piece p1 is the player's own character.
		owned, err := uuid.Parse(r.pieces["p1"].CharacterID)
		if err != nil {
			t.Fatal(err)
		}
		turnID := openTurn(r)
		react := escape(owned)

		r.recordOpenedReactionMoveViews(turnID, react)

		views, ok := r.pendingTurns[turnID].reactionMoveViews[react.GetID()]
		if !ok {
			t.Fatal("no entry for the reaction: recorded, nobody saw, is an empty map — not absence")
		}
		if _, ok := views[playerUUID]; ok || len(views) != 0 {
			t.Fatalf("views = %v, want empty: the owner always sees all of it", views)
		}
	})

	t.Run("a reaction with no move records nothing", func(t *testing.T) {
		r, _, _ := fogTestRoom(t)
		turnID := openTurn(r)
		react := *action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{}, nil, nil, nil, nil, &action.Dodge{}, nil, nil)

		r.recordOpenedReactionMoveViews(turnID, react)

		if w, ok := r.pendingTurns[turnID]; ok && w.reactionMoveViews != nil {
			t.Fatalf("reactionMoveViews = %v, want none for a reaction that does not move", w.reactionMoveViews)
		}
	})

	t.Run("a turn that is no longer the open one records nothing", func(t *testing.T) {
		r, _, _ := fogTestRoom(t)
		turnID := openTurn(r)
		react := escape(uuid.New())

		r.recordOpenedReactionMoveViews(uuid.New(), react)

		if _, ok := r.pendingTurns[turnID]; ok {
			t.Fatal("something was held for the open turn from a record aimed at another")
		}
		if len(r.pendingTurns) != 0 {
			t.Fatalf("pendingTurns = %v, want nothing held for a turn that is not open", r.pendingTurns)
		}
	})

	t.Run("the close drains it with the rest of the turn", func(t *testing.T) {
		r, playerUUID, _ := fogTestRoom(t)
		reactor := uuid.New()
		r.pieces["reactor"] = PieceMovedPayload{PieceID: "reactor", CharacterID: reactor.String(), Slot: squareSlot(1, 0)}
		turnID := openTurn(r)
		react := escape(reactor)
		r.recordOpenedReactionMoveViews(turnID, react)

		r.mu.Lock()
		w := r.takeTurnWritesLocked(turnID)
		r.mu.Unlock()

		if w.reactionMoveViews[react.GetID()][playerUUID] != masteraction.ViewFull {
			t.Fatalf("drained reactionMoveViews = %v, want the player at %q", w.reactionMoveViews, masteraction.ViewFull)
		}
		if _, ok := r.pendingTurns[turnID]; ok {
			t.Fatal("the turn's writes are still held after the drain")
		}
	})
}
