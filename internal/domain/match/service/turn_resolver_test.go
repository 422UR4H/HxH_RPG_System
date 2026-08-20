package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// resolveWith builds the input for a resolution that carries no sheets — the wall and
// lifecycle paths never touch one.
func resolveWith(tn *turn.Turn, targets service.TargetReader) service.ResolveInput {
	return service.ResolveInput{
		Turn:    tn,
		Targets: targets,
		Rules:   match.NewDefaultMatchRules(),
		Weapons: item.NewWeaponsManagerFactory().Build(),
	}
}

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
		res := resolver.Resolve(resolveWith(tRn, noopTargetReader{}))
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
	})

	t.Run("IsSettled is false when turn has no finishedAt", func(t *testing.T) {
		tRn := makeTurn()
		res := resolver.Resolve(resolveWith(tRn, noopTargetReader{}))
		if res.IsSettled {
			t.Error("expected IsSettled=false for open turn")
		}
	})

	t.Run("IsSettled is true when turn is closed", func(t *testing.T) {
		tRn := makeTurn()
		tRn.Close(time.Now())
		res := resolver.Resolve(resolveWith(tRn, noopTargetReader{}))
		if !res.IsSettled {
			t.Error("expected IsSettled=true for closed turn")
		}
	})

	t.Run("ReactionResults has one entry per reaction", func(t *testing.T) {
		tRn := makeTurn()
		act := tRn.GetAction()
		reaction := makeReactionTo((&act).GetID())
		tRn.AddReaction(reaction)

		res := resolver.Resolve(resolveWith(tRn, noopTargetReader{}))

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

		res := resolver.Resolve(resolveWith(tRn, reader))

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

		res := resolver.Resolve(resolveWith(tRn, reader))

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

		res := resolver.Resolve(resolveWith(tRn, nil))

		if len(res.WallResults) != 0 {
			t.Errorf("expected 0 WallResults when targets=nil, got %d", len(res.WallResults))
		}
	})
}

// TestResolve_ClosedDodgeReserveIsRead pins the wiring from round 1 review: a closed dodge's
// banked reserve (match.Modifier, Dimension: DimDodge, Against: ScopeAllBut(originalAttacker))
// only does anything if a LATER resolution can see it. Before this fix, ResolveInput had no
// way to reach a character's ModifierLedger and deriveReflex/deriveEvasion never read
// ReactionInput.Ledger — the reserve was written and never applied, for anyone.
//
// This test goes through TurnResolver.Resolve with ResolveInput.Statuses populated, the same
// way MatchSession.ResolveTurn now populates it from its own statuses map — not a narrower
// ResolveReaction-level check, which would prove Derive reads a ledger but not that the
// resolver actually threads one to it.
func TestResolve_ClosedDodgeReserveIsRead(t *testing.T) {
	target := uuid.New()
	bankedAgainst := uuid.New() // the attacker the reserve does NOT apply to
	otherAttacker := uuid.New() // per ScopeAllBut, anyone else gets it

	ledger := match.NewModifierLedger()
	ledger.Add(match.Modifier{
		Amount: 5, Applies: match.DimDodge, Source: match.SourceSystem,
		Against: match.ScopeAllBut(bankedAgainst), ExpiresAt: match.LifetimeEndOfTurn,
		Reason: "test: banked closed dodge reserve",
	})
	statuses := map[uuid.UUID]*match.CharacterStatus{
		target: {Ledger: ledger},
	}

	dodgeTotalAgainst := func(t *testing.T, attackerID uuid.UUID) int {
		t.Helper()
		// A weak hit (5) so the passive path resolves without a reaction opened — the
		// reserve should still count, because deriveReflex reads it regardless of kind.
		tn := attackTurn(attackerID, target, []int{3, 2}, []int{1}, nil)
		in := resolveInput(t, attackerID, target, tn)
		in.Statuses = statuses
		res := service.TurnResolver{}.Resolve(in)
		if len(res.CharacterResults) != 1 {
			t.Fatalf("expected 1 character result, got %d", len(res.CharacterResults))
		}
		return res.CharacterResults[0].Dodge.Total
	}

	t.Run("against the attacker the reserve was earned against, it does not count", func(t *testing.T) {
		if got := dodgeTotalAgainst(t, bankedAgainst); got != 11 {
			t.Fatalf("dodge total = %d, want 11 (passive, no bonus)", got)
		}
	})

	t.Run("against a different attacker, the reserve counts", func(t *testing.T) {
		if got := dodgeTotalAgainst(t, otherAttacker); got != 16 {
			t.Fatalf("dodge total = %d, want 16 (11 passive + 5 reserve) — ScopeAllBut must actually be read", got)
		}
	})
}
