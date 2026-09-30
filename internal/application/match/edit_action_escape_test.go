package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// ─── edit_action's escapeLanding (B13) ─────────────────────────────────────
//
// Where a failed escape leaves the piece is the master's choice, made over the same edit
// surface as every other correction to the open turn. These tests drive EditActionUC straight
// against a session, like the rest of this file's neighbours.

// topFaces lands every die on its top face. The escape is decided by the master's own
// condition levers below, never by the dice, so exact faces do not matter here.
type topFaces struct{}

func (topFaces) RollDie(sides enum.DieSides) int { return sides.GetSides() }

type escapeEditFixture struct {
	session    *matchsession.MatchSession
	masterUUID uuid.UUID
	victimID   uuid.UUID
	reactionID uuid.UUID
}

// newEscapeEditFixture opens an attack on the victim and attaches and opens the victim's
// Dash escape from (6,6) to (8,6) on a 30x30 square board, with its moveSpeed pushed far below
// the hit — an escape that FAILS on the movement alone.
func newEscapeEditFixture(t *testing.T) *escapeEditFixture {
	t.Helper()
	matchUUID, masterUUID, playerUUID := uuid.New(), uuid.New(), uuid.New()
	attackerID, victimID := uuid.New(), uuid.New()
	participants := []*matchDomain.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: attackerID, PlayerUUID: &playerUUID}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: victimID, PlayerUUID: &playerUUID}},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attackerID: newEditSheet(t),
		victimID:   newEditSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, participants)
	s.SetRollSource(topFaces{})
	s.SyncMapState(nil, mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 30, Rows: 30, CellSize: 64, SkewRatio: 1,
	})

	weapon := enum.Sword
	act := action.NewAction(attackerID, []uuid.UUID{victimID}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil,
		&action.Attack{
			Weapon: &weapon,
			Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
			Damage: action.RollCheck{SkillName: enum.Push.String()},
		},
		nil, nil, nil, nil,
	)
	if err := s.EnqueueAction(playerUUID, act); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	r := action.NewAction(victimID, nil, act.GetID(), nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil,
		&action.Move{
			Category: enum.Dash, From: [3]int{6, 6, 0}, Position: [3]int{8, 6, 0},
			Speed: &action.RollCheck{SkillName: enum.Accelerate.String()},
		},
		nil, nil, &action.Dodge{}, nil, nil,
	)
	r.ReactionKind = action.ReactEscape
	if _, err := s.AttachReaction(playerUUID, r); err != nil {
		t.Fatalf("AttachReaction: %v", err)
	}
	if _, _, err := s.OpenReaction(r.GetID()); err != nil {
		t.Fatalf("OpenReaction: %v", err)
	}

	f := &escapeEditFixture{session: s, masterUUID: masterUUID, victimID: victimID, reactionID: r.GetID()}
	f.setMoveSpeed(t, -1000)
	return f
}

// setMoveSpeed puts the master's condition on the escape's moveSpeed. With every die on its top
// face the dodge ties the hit and passes, so only the movement decides.
func (f *escapeEditFixture) setMoveSpeed(t *testing.T, modifier int) {
	t.Helper()
	ma := action.NewMasterAction()
	ma.ActionID = f.reactionID
	ma.Conditions = []action.ConditionEdit{
		{Field: action.FieldMoveSpeed, Condition: action.RollCondition{Modifier: modifier}},
	}
	if _, err := match.NewEditActionUC().Execute(
		context.Background(), f.session, f.masterUUID, f.masterUUID, ma, nil); err != nil {
		t.Fatalf("setMoveSpeed: %v", err)
	}
}

func (f *escapeEditFixture) escapeIn(t *testing.T, res *service.TurnResolution) *service.EscapeResult {
	t.Helper()
	for _, cr := range res.CharacterResults {
		if cr.TargetID == f.victimID {
			if cr.Escape == nil {
				t.Fatal("the victim's result carries no escape")
			}
			return cr.Escape
		}
	}
	t.Fatal("no result for the victim")
	return nil
}

func (f *escapeEditFixture) masterActionsOnTurn() int {
	return len(f.session.GetActiveRound().CurrentTurn().GetMasterActions())
}

func TestEditActionEscapeLanding(t *testing.T) {
	uc := match.NewEditActionUC()
	landing := func(f *escapeEditFixture, pos *[3]int) *match.EscapeLandingEdit {
		return &match.EscapeLandingEdit{ReactionID: f.reactionID, Position: pos}
	}

	t.Run("a landing alone sets the choice and recomputes, with no condition edit recorded", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		before := f.masterActionsOnTurn()
		ma := action.NewMasterAction()
		ma.ActionID = f.reactionID

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma,
			landing(f, &[3]int{7, 6, 0}))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		esc := f.escapeIn(t, res.Resolution)
		if esc.Landing == nil || *esc.Landing != [3]int{7, 6, 0} {
			t.Fatalf("Landing = %v, want [7 6 0]", esc.Landing)
		}
		if res.TurnID != f.session.CurrentTurnID() {
			t.Fatalf("TurnID = %s, want the open turn's %s", res.TurnID, f.session.CurrentTurnID())
		}
		// Choosing a landing is not a correction of any roll; an empty MasterAction appended
		// for it would be a blank entry in the turn's edit history.
		if after := f.masterActionsOnTurn(); after != before {
			t.Fatalf("master actions on the turn went from %d to %d; a landing-only edit edits no roll",
				before, after)
		}
	})

	t.Run("a null position clears it", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		ma := action.NewMasterAction()
		ma.ActionID = f.reactionID
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma,
			landing(f, &[3]int{7, 6, 0})); err != nil {
			t.Fatalf("Execute(set): %v", err)
		}
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma, landing(f, nil))
		if err != nil {
			t.Fatalf("Execute(clear): %v", err)
		}
		if esc := f.escapeIn(t, res.Resolution); esc.Landing != nil {
			t.Fatalf("Landing = %v after a null position, want none", *esc.Landing)
		}
	})

	t.Run("a condition edit and a landing in one payload both land", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		ma := action.NewMasterAction()
		ma.ActionID = f.reactionID
		// Still failing, by a different amount: the condition is what proves it landed.
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldMoveSpeed, Condition: action.RollCondition{Modifier: -999},
		}}
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma,
			landing(f, &[3]int{7, 6, 0}))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if esc := f.escapeIn(t, res.Resolution); esc.Landing == nil {
			t.Fatal("the landing did not land")
		}
		r := f.session.GetActiveRound().CurrentTurn().ReactionRef(f.reactionID)
		if c := r.Move.Speed.Context.Condition; c == nil || c.Modifier != -999 {
			t.Fatalf("moveSpeed condition = %+v, want the -999 edit applied", c)
		}
	})

	// Validate everything before mutating anything: a refused landing must not leave the
	// condition edit behind it applied.
	t.Run("a refused landing leaves the condition edit unapplied", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		before := f.masterActionsOnTurn()
		ma := action.NewMasterAction()
		ma.ActionID = f.reactionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldMoveSpeed, Condition: action.RollCondition{Modifier: -999},
		}}
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma,
			landing(f, &[3]int{99, 6, 0}))
		if !errors.Is(err, matchsession.ErrLandingOutOfGrid) {
			t.Fatalf("err = %v, want ErrLandingOutOfGrid", err)
		}
		r := f.session.GetActiveRound().CurrentTurn().ReactionRef(f.reactionID)
		if c := r.Move.Speed.Context.Condition; c == nil || c.Modifier != -1000 {
			t.Fatalf("moveSpeed condition = %+v, want the fixture's -1000 untouched", c)
		}
		if after := f.masterActionsOnTurn(); after != before {
			t.Fatalf("master actions went from %d to %d on a refused edit", before, after)
		}
	})

	t.Run("a landing on the turn's own action is refused", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		ma := action.NewMasterAction()
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma,
			&match.EscapeLandingEdit{ReactionID: uuid.Nil, Position: &[3]int{7, 6, 0}})
		if !errors.Is(err, matchsession.ErrNotAnEscape) {
			t.Fatalf("err = %v, want ErrNotAnEscape", err)
		}
	})

	t.Run("only the master", func(t *testing.T) {
		f := newEscapeEditFixture(t)
		ma := action.NewMasterAction()
		ma.ActionID = f.reactionID
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(), ma,
			landing(f, &[3]int{7, 6, 0}))
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
		if got := f.session.GetActiveRound().CurrentTurn().EscapeLandings(); len(got) != 0 {
			t.Fatalf("a refused caller stored a landing: %v", got)
		}
	})
}
