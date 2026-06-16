package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type noopTargetReader struct{}

func (noopTargetReader) CategorizeTarget(uuid.UUID) service.TargetKind {
	return service.TargetKindUnknown
}
func (noopTargetReader) GetWall(string) (mapentity.WallSegment, bool) {
	return mapentity.WallSegment{}, false
}

func TestTurnResolver_Resolve(t *testing.T) {
	resolver := service.TurnResolver{}

	t.Run("returns non-nil TurnResolution for a Turn with only an action", func(t *testing.T) {
		tRn := makeTurn()
		res := resolver.Resolve(tRn, nil, noopTargetReader{})
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
	})

	t.Run("IsSettled is false when turn has no finishedAt", func(t *testing.T) {
		tRn := makeTurn()
		res := resolver.Resolve(tRn, nil, noopTargetReader{})
		if res.IsSettled {
			t.Error("expected IsSettled=false for open turn")
		}
	})

	t.Run("IsSettled is true when turn is closed", func(t *testing.T) {
		tRn := makeTurn()
		tRn.Close(time.Now())
		res := resolver.Resolve(tRn, nil, noopTargetReader{})
		if !res.IsSettled {
			t.Error("expected IsSettled=true for closed turn")
		}
	})

	t.Run("ReactionResults has one entry per reaction", func(t *testing.T) {
		tRn := makeTurn()
		act := tRn.GetAction()
		reaction := makeReactionTo((&act).GetID())
		tRn.AddReaction(reaction)

		res := resolver.Resolve(tRn, nil, noopTargetReader{})

		if len(res.ReactionResults) != 1 {
			t.Errorf("expected 1 ReactionResult, got %d", len(res.ReactionResults))
		}
	})
}

func makeTurn() *turn.Turn {
	a := action.NewAction(
		uuid.New(),
		[]uuid.UUID{uuid.New()},
		uuid.Nil,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	return turn.NewTurn(*a)
}

// mockWallReader implements TargetReader with one pre-configured wall.
type mockWallReader struct {
	wallID string
	wall   mapentity.WallSegment
}

func (m mockWallReader) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if id.String() == m.wallID {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}

func (m mockWallReader) GetWall(id string) (mapentity.WallSegment, bool) {
	if id == m.wallID {
		return m.wall, true
	}
	return mapentity.WallSegment{}, false
}

func TestTurnResolver_Resolve_WallTargets(t *testing.T) {
	resolver := service.TurnResolver{}
	wallID := uuid.New()
	wall := mapentity.WallSegment{
		ID:         wallID.String(),
		HP:         40,
		MaxHP:      40,
		Resistance: 5,
	}
	reader := mockWallReader{wallID: wallID.String(), wall: wall}

	t.Run("Attack on wall produces WallResult with Kind=attack", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil,
			&action.Attack{},
			nil, nil, nil, nil,
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, reader)

		if len(res.WallResults) != 1 {
			t.Fatalf("expected 1 WallResult, got %d", len(res.WallResults))
		}
		if res.WallResults[0].Kind != service.WallResultKindAttack {
			t.Errorf("expected Kind=attack, got %s", res.WallResults[0].Kind)
		}
		if res.WallResults[0].UpdatedWall.ID != wallID.String() {
			t.Errorf("UpdatedWall.ID mismatch")
		}
	})

	t.Run("Interact (open) on wall produces WallResult with Kind=interact", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil,
			&action.Interact{Kind: action.InteractOpen},
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, reader)

		if len(res.WallResults) != 1 {
			t.Fatalf("expected 1 WallResult, got %d", len(res.WallResults))
		}
		if res.WallResults[0].Kind != service.WallResultKindInteract {
			t.Errorf("expected Kind=interact, got %s", res.WallResults[0].Kind)
		}
		if !res.WallResults[0].UpdatedWall.Open {
			t.Error("expected UpdatedWall.Open=true after InteractOpen")
		}
	})

	t.Run("nil targets skips wall routing", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil,
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, nil)

		if len(res.WallResults) != 0 {
			t.Errorf("expected 0 WallResults when targets=nil, got %d", len(res.WallResults))
		}
	})
}
