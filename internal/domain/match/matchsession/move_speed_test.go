package matchsession_test

import (
	"errors"
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type moveSpeedFixture struct {
	s      *matchsession.MatchSession
	player uuid.UUID
	char   uuid.UUID
	sheet  *csSheet.CharacterSheet
}

func newMoveSpeedFixture(t *testing.T, mode enum.RoundMode, src service.RollSource) *moveSpeedFixture {
	t.Helper()
	matchUUID := uuid.New()
	player := uuid.New()
	p := makeParticipant(matchUUID, &player)
	char := p.Sheet.UUID
	sheet := buildPlainSheet(t)
	s := matchsession.NewMatchSession(matchUUID, map[uuid.UUID]*csSheet.CharacterSheet{char: sheet},
		[]*match.Participant{p})
	s.GetActiveRound().SetMode(mode)
	s.SetRollSource(src)
	return &moveSpeedFixture{s: s, player: player, char: char, sheet: sheet}
}

func (f *moveSpeedFixture) dash() *action.Action {
	return action.NewAction(f.char, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
		&action.Move{Category: enum.Dash, Position: [3]int{2, 2, 0},
			Speed: &action.RollCheck{SkillName: enum.Accelerate.String()}},
		nil, nil, nil, nil, nil)
}

// D1: a pure move has no actionSpeed — no skill, no dice, result 0 — only its moveSpeed.
func TestPureMove_HasNoActionSpeed(t *testing.T) {
	t.Run("a Dash in Race rolls only Accelerate", func(t *testing.T) {
		// 4 faces cover exactly one 2D10 check; a Legerity roll would drain past them.
		src := &countingSource{face: 6}
		f := newMoveSpeedFixture(t, enum.Race, src)
		a := f.dash()
		if err := f.s.EnqueueAction(f.player, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		if !a.Speed.Attempts.IsEmpty() || a.Speed.SkillName != "" || a.Speed.Result != 0 {
			t.Errorf("Speed = %+v, want empty: a pure move has no actionSpeed", a.Speed.RollCheck)
		}
		want := 12 + skillValue(t, f.sheet, enum.Accelerate)
		if a.Move.FinalSpeed != want {
			t.Errorf("FinalSpeed = %d, want %d", a.Move.FinalSpeed, want)
		}
		if src.rolls != 4 {
			t.Errorf("dice rolled = %d, want 4 (moveSpeed only; Legerity must consume none)", src.rolls)
		}
	})

	t.Run("a Shift takes Brake passive and nothing else", func(t *testing.T) {
		src := &countingSource{face: 9}
		f := newMoveSpeedFixture(t, enum.Race, src)
		a := action.NewAction(f.char, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Shift, Position: [3]int{2, 2, 0},
				Speed: &action.RollCheck{SkillName: enum.Brake.String()}},
			nil, nil, nil, nil, nil)
		if err := f.s.EnqueueAction(f.player, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		if a.Speed.SkillName != "" || a.Speed.Result != 0 || src.rolls != 0 {
			t.Errorf("Speed = %+v, rolls = %d, want no actionSpeed and no dice", a.Speed.RollCheck, src.rolls)
		}
		if want := 11 + skillValue(t, f.sheet, enum.Brake); a.Move.FinalSpeed != want {
			t.Errorf("FinalSpeed = %d, want %d", a.Move.FinalSpeed, want)
		}
	})

	t.Run("an attack and a combined action keep Legerity", func(t *testing.T) {
		f := newMoveSpeedFixture(t, enum.Race, &countingSource{face: 5})
		atk := action.NewAction(f.char, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil,
			&action.Attack{}, nil, nil, nil, nil)
		comb := action.NewAction(f.char, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Dash, Position: [3]int{2, 2, 0}}, &action.Attack{}, nil, nil, nil, nil)
		for _, a := range []*action.Action{atk, comb} {
			if err := f.s.EnqueueAction(f.player, a); err != nil {
				t.Fatalf("EnqueueAction: %v", err)
			}
			if a.Speed.SkillName != enum.Legerity.String() || a.Speed.Attempts.IsEmpty() {
				t.Errorf("Speed = %+v, want Legerity rolled", a.Speed.RollCheck)
			}
		}
	})

	t.Run("a move-only escape reaction gets no Legerity", func(t *testing.T) {
		f := newQueueFixture(t)
		opened := mustOpen(t, f.session)
		shift := action.NewAction(f.b, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, &action.Move{Category: enum.Shift, Position: [3]int{5, 5, 0}}, nil, nil, nil, nil, nil)
		if err := f.session.EnqueueAction(f.playerB, shift); err != nil {
			t.Fatalf("EnqueueAction(shift): %v", err)
		}
		escape := action.NewAction(f.b, nil, opened.ActionRef().GetID(), nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Dash, Position: [3]int{8, 8, 0}}, nil, nil,
			&action.Dodge{RollCheck: action.RollCheck{SkillName: enum.Reflex.String()}}, nil, nil)
		escape.ReactionKind = action.ReactClosedEscape
		if _, err := f.session.AttachReaction(f.playerB, escape); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if escape.Speed.SkillName != "" || !escape.Speed.Attempts.IsEmpty() || escape.Speed.Result != 0 {
			t.Errorf("Speed = %+v, want empty on a move-only escape", escape.Speed.RollCheck)
		}
	})

	t.Run("a condition edit on speed is refused", func(t *testing.T) {
		f := newMoveSpeedFixture(t, enum.Free, &countingSource{face: 5})
		a := f.dash()
		if err := f.s.EnqueueAction(f.player, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		ma := action.NewMasterAction()
		ma.ActionID = a.GetID()
		ma.Conditions = []action.ConditionEdit{{Field: action.FieldSpeed, Condition: action.RollCondition{Modifier: 3}}}
		_, err := f.s.ApplyMasterAction(ma, uuid.New())
		if !errors.Is(err, matchsession.ErrConditionTargetMissing) {
			t.Fatalf("err = %v, want ErrConditionTargetMissing", err)
		}
	})
}

// D2: Free orders by the speed on the action's bars.
func TestFree_OrdersByTheSpeedOfTheBars(t *testing.T) {
	// Every die 10: the Dash (tested) is 20 + Accelerate; the attack takes the passive 11 + Legerity.
	f := newMoveSpeedFixture(t, enum.Free, &countingSource{face: 10})
	atk := action.NewAction(f.char, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil,
		&action.Attack{}, nil, nil, nil, nil)
	if err := f.s.EnqueueAction(f.player, atk); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	dash := f.dash()
	if err := f.s.EnqueueAction(f.player, dash); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	if dash.OrderSpeed() <= atk.OrderSpeed() {
		t.Fatalf("test setup: dash %d must outrun attack %d", dash.OrderSpeed(), atk.OrderSpeed())
	}
	if got := mustOpen(t, f.s).ActionRef().GetID(); got != dash.GetID() {
		t.Errorf("opened %v first, want the pure move (moveSpeed %d over the attack's %d)",
			got, dash.OrderSpeed(), atk.OrderSpeed())
	}
}
