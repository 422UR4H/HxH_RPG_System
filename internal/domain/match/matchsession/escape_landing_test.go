package matchsession_test

import (
	"errors"
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// ─── B13: where a failed escape leaves the piece is the master's call ──────
//
// The escape clears its test only if the dodge AND the movement both beat the attacker's hit
// (front-combat-phases.md §6A.5, B13). When it fails, the engine has no rule for where the
// piece ends up, so the master says — over edit_action, stored on the open turn, copied into
// the resolution by the resolver — and only when it failed.

type escapeFixture struct {
	session    *matchsession.MatchSession
	masterUUID uuid.UUID
	attacker   uuid.UUID
	victim     uuid.UUID
	actionID   uuid.UUID
	reactionID uuid.UUID // the victim's escape, attached and opened
}

// newEscapeFixture opens an attack from attacker on victim and attaches and opens the
// victim's Dash escape from (6,6) to (8,6), on the given board. The attached escape is left to the
// dice; each test puts the master's own condition on its moveSpeed and dodge to decide it.
func newEscapeFixture(t *testing.T, grid mapentity.GridShape) *escapeFixture {
	t.Helper()
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA := makeParticipant(matchUUID, &playerA)
	pB := makeParticipant(matchUUID, &playerB)
	attacker, victim := pA.Sheet.UUID, pB.Sheet.UUID

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attacker: buildPlainSheet(t),
		victim:   buildPlainSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{pA, pB})
	s.SetRollSource(fixedTopFaceSource{})
	s.SyncMapState(nil, grid)

	sword := enum.Sword
	atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
	a := action.NewAction(attacker, []uuid.UUID{victim}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}},
		nil, nil, atk, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerA, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	opened := mustOpen(t, s)
	act := opened.GetAction()

	r := action.NewAction(victim, nil, act.GetID(), nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil,
		&action.Move{
			Category: enum.Dash,
			From:     &[3]int{6, 6, 0},
			Position: [3]int{8, 6, 0},
			Speed:    &action.RollCheck{SkillName: enum.Accelerate.String()},
		},
		nil, nil, &action.Dodge{}, nil, nil,
	)
	r.ReactionKind = action.ReactEscape
	if _, err := s.AttachReaction(playerB, r); err != nil {
		t.Fatalf("AttachReaction(escape): %v", err)
	}
	// Opened, so it is a step of the chain and produces the victim's CharacterResult — an
	// attached-but-unopened reaction is only named in PendingReactions.
	if _, _, err := s.OpenReaction(r.GetID()); err != nil {
		t.Fatalf("OpenReaction(escape): %v", err)
	}
	return &escapeFixture{
		session: s, masterUUID: masterUUID, attacker: attacker, victim: victim,
		actionID: act.GetID(), reactionID: r.GetID(),
	}
}

var escapeGrid = mapentity.GridShape{
	Kind: mapentity.GridKindSquare, Cols: 30, Rows: 30, CellSize: 64, SkewRatio: 1,
}

// decide settles the escape with the master's own levers, so the TEST decides each half, not
// the dice. The movement is a condition on the escape's moveSpeed (±5000). The dodge cannot be
// levered the same way — a condition on the reaction's dodge does not read through to the
// resolver's Reflex derivation today — so it is decided from the other side, a condition on
// the attack's hit: −1000 and the dodge clears it, +1000 and it cannot. ±5000 on the movement
// stays clear of both hits. It returns the resolution after the second edit.
func (f *escapeFixture) decide(t *testing.T, movePasses, dodgePasses bool) *service.TurnResolution {
	t.Helper()
	hit, move := 1000, -5000
	if dodgePasses {
		hit = -1000
	}
	if movePasses {
		move = 5000
	}
	onHit := action.NewMasterAction()
	onHit.ActionID = f.actionID
	onHit.Conditions = []action.ConditionEdit{
		{Field: action.FieldHit, Condition: action.RollCondition{Modifier: hit}},
	}
	if _, err := f.session.ApplyMasterAction(onHit, f.masterUUID); err != nil {
		t.Fatalf("ApplyMasterAction(hit): %v", err)
	}
	onMove := action.NewMasterAction()
	onMove.ActionID = f.reactionID
	onMove.Conditions = []action.ConditionEdit{
		{Field: action.FieldMoveSpeed, Condition: action.RollCondition{Modifier: move}},
	}
	res, err := f.session.ApplyMasterAction(onMove, f.masterUUID)
	if err != nil {
		t.Fatalf("ApplyMasterAction(moveSpeed): %v", err)
	}
	return res
}

// escapeOf reads the victim's EscapeResult off a resolution.
func (f *escapeFixture) escapeOf(t *testing.T, res *service.TurnResolution) *service.EscapeResult {
	t.Helper()
	if res == nil {
		t.Fatal("no resolution")
	}
	for _, cr := range res.CharacterResults {
		if cr.TargetID == f.victim {
			return cr.Escape
		}
	}
	t.Fatalf("the victim has no CharacterResult: %+v", res.CharacterResults)
	return nil
}

func TestMatchSession_SetEscapeLanding(t *testing.T) {
	t.Run("a failed escape carries the landing the master chose", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		f.decide(t, false, true)

		res, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0})
		if err != nil {
			t.Fatalf("SetEscapeLanding: %v", err)
		}
		esc := f.escapeOf(t, res)
		if esc == nil || esc.Escaped || esc.MovePassed || !esc.DodgePassed {
			t.Fatalf("escape = %+v, want a failed movement with the dodge passing", esc)
		}
		if esc.Landing == nil || *esc.Landing != [3]int{7, 6, 0} {
			t.Fatalf("Landing = %v, want [7 6 0]", esc.Landing)
		}
	})

	t.Run("a nil position clears the choice", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		f.decide(t, false, true)
		if _, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0}); err != nil {
			t.Fatalf("SetEscapeLanding: %v", err)
		}

		res, err := f.session.SetEscapeLanding(f.reactionID, nil)
		if err != nil {
			t.Fatalf("SetEscapeLanding(nil): %v", err)
		}
		if esc := f.escapeOf(t, res); esc == nil || esc.Landing != nil {
			t.Fatalf("escape = %+v, want a failed escape with no landing after the clear", esc)
		}
	})

	// Review focus 4: the choice is a standing one, not a verdict. An escape that PASSES goes
	// to its own slot, whatever the master stored while it was failing.
	t.Run("an escape that passes never carries the landing, even when one is stored", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		f.decide(t, false, true)
		if _, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0}); err != nil {
			t.Fatalf("SetEscapeLanding: %v", err)
		}

		res := f.decide(t, true, true)
		esc := f.escapeOf(t, res)
		if esc == nil || !esc.Escaped {
			t.Fatalf("escape = %+v, want it to pass after the master changed the reading", esc)
		}
		if esc.Landing != nil {
			t.Fatalf("Landing = %v on an escape that passed; it goes to its own slot", *esc.Landing)
		}
	})

	t.Run("refuses a reaction that does not displace", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		// A second target reaction would need a second target; the turn's own action is
		// enough to prove the kind is checked — it is not a reaction at all, let alone an
		// escape.
		for _, id := range []uuid.UUID{f.actionID, uuid.Nil} {
			if _, err := f.session.SetEscapeLanding(id, &[3]int{7, 6, 0}); !errors.Is(err, matchsession.ErrNotAnEscape) {
				t.Fatalf("SetEscapeLanding(%s) err = %v, want ErrNotAnEscape", id, err)
			}
		}
	})

	t.Run("refuses a dodge attached to the turn", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		// Rewrite the attached escape into a dodge through the turn's own pointer — the only
		// way to have a non-displacing REACTION on this turn without a second target.
		tn := f.session.GetActiveRound().CurrentTurn()
		tn.ReactionRef(f.reactionID).ReactionKind = action.ReactDodge
		if _, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0}); !errors.Is(err, matchsession.ErrNotAnEscape) {
			t.Fatalf("err = %v, want ErrNotAnEscape", err)
		}
		if got := tn.EscapeLandings(); len(got) != 0 {
			t.Fatalf("a refused landing was stored anyway: %v", got)
		}
	})

	t.Run("refuses an id that is not on the turn", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		if _, err := f.session.SetEscapeLanding(uuid.New(), &[3]int{7, 6, 0}); !errors.Is(err, matchsession.ErrActionNotOnTurn) {
			t.Fatalf("err = %v, want ErrActionNotOnTurn", err)
		}
	})

	t.Run("refuses when no turn is open", func(t *testing.T) {
		f := newEscapeFixture(t, escapeGrid)
		if _, err := f.session.CloseOpenTurn(); err != nil {
			t.Fatalf("CloseOpenTurn: %v", err)
		}
		if _, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0}); !errors.Is(err, matchsession.ErrNoActiveTurn) {
			t.Fatalf("err = %v, want ErrNoActiveTurn", err)
		}
	})
}

func TestMatchSession_SetEscapeLandingChecksTheGrid(t *testing.T) {
	hex := mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1}
	cases := []struct {
		name string
		grid mapentity.GridShape
		pos  [3]int
		ok   bool
	}{
		{"square, inside", escapeGrid, [3]int{29, 29, 0}, true},
		{"square, origin", escapeGrid, [3]int{0, 0, 0}, true},
		{"square, past the last column", escapeGrid, [3]int{30, 6, 0}, false},
		{"square, past the last row", escapeGrid, [3]int{6, 30, 0}, false},
		{"square, negative", escapeGrid, [3]int{-1, 6, 0}, false},
		// Hex is axial (q, r) on the wire; the grid's cols/rows are odd-r OFFSET, so the
		// column is q + floor(r/2) — the same bridge the front's isSlotInBounds uses.
		{"hex, q negative but its offset column is 0", hex, [3]int{-2, 5, 0}, true},
		{"hex, offset column -1", hex, [3]int{-3, 5, 0}, false},
		{"hex, last column of row 0", hex, [3]int{9, 0, 0}, true},
		{"hex, past the last column of row 0", hex, [3]int{10, 0, 0}, false},
		{"hex, past the last row", hex, [3]int{0, 10, 0}, false},
		// A board with no dimensions (none synced yet) has nothing to check against.
		{"no grid yet", mapentity.GridShape{}, [3]int{500, -3, 0}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newEscapeFixture(t, c.grid)
			_, err := f.session.SetEscapeLanding(f.reactionID, &c.pos)
			if c.ok && err != nil {
				t.Fatalf("SetEscapeLanding(%v) = %v, want it accepted", c.pos, err)
			}
			if !c.ok && !errors.Is(err, matchsession.ErrLandingOutOfGrid) {
				t.Fatalf("SetEscapeLanding(%v) err = %v, want ErrLandingOutOfGrid", c.pos, err)
			}
		})
	}
}

// The close reads the same stored choice: the settled resolution carries the landing of a
// failed escape, and none for one that passed.
func TestMatchSession_TheSettledResolutionCarriesTheLanding(t *testing.T) {
	cases := []struct {
		name        string
		movePasses  bool
		wantLanding bool
	}{
		{"failed escape → the master's landing", false, true},
		{"passed escape → no landing", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newEscapeFixture(t, escapeGrid)
			f.decide(t, c.movePasses, true)
			if _, err := f.session.SetEscapeLanding(f.reactionID, &[3]int{7, 6, 0}); err != nil {
				t.Fatalf("SetEscapeLanding: %v", err)
			}
			tr, err := f.session.CloseOpenTurn()
			if err != nil || tr.ClosedResolution == nil {
				t.Fatalf("CloseOpenTurn: %v", err)
			}
			esc := f.escapeOf(t, tr.ClosedResolution)
			if esc == nil {
				t.Fatal("the settled resolution lost the escape")
			}
			if got := esc.Landing != nil; got != c.wantLanding {
				t.Fatalf("Landing = %v, want present=%v", esc.Landing, c.wantLanding)
			}
		})
	}
}
