package match_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// ─── fixture ────────────────────────────────────────────────────────────────
//
// editFixture is internal/app/game/combat_e2e_test.go's combatFixture minus the HTTP
// server, hub and handler: the edit use case is driven directly against the session, the way
// every other application/match test already does (see attach_reaction_test.go).

// scriptedFaces hands out faces in order and NEVER repeats: once exhausted, it records an
// overrun instead of silently replaying the last face — see combat_e2e_test.go's same-named
// type for the full rationale (a repeat can never be told apart from a genuine match by a
// test that relies on it). That one lives in internal/app/game, a different Go package, and
// is unexported there, so this package needs its own copy rather than importing it.
type scriptedFaces struct {
	faces   []int
	i       int
	overran bool // set once a roll asks for a face beyond what was scripted
}

func (s *scriptedFaces) RollDie(_ enum.DieSides) int {
	if s.i >= len(s.faces) {
		s.overran = true
		return -1 // an impossible face: anything that accidentally depends on it fails loudly
	}
	f := s.faces[s.i]
	s.i++
	return f
}

// consumed is how many faces the scripted source has handed out. It is the only way to prove
// "no die was re-rolled" — asserting on the numbers cannot distinguish a re-roll that
// happened to land the same.
func (s *scriptedFaces) consumed() int { return s.i }

type editFixture struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	playerUUID uuid.UUID
	attackerID uuid.UUID // sheet UUID of the attacking character
	victimID   uuid.UUID // sheet UUID of the target
	// thirdID is a bystander: present in the session but never targeted by the fixture's
	// attack, on the shape of newCombatFixture's roster.
	thirdID uuid.UUID

	session    *matchsession.MatchSession
	rollSource *scriptedFaces

	actionID uuid.UUID // the open turn's own action
	// facesAtOpen is rollSource.consumed() right after the fixture opens the turn — the
	// baseline the advantage test checks an edit never moves.
	facesAtOpen int
	// primaryTotal is ActionResult.Total read off the Primary set alone, before any edit —
	// what the advantage test proves the edit reads PAST, onto the (scripted higher) Secondary
	// set.
	primaryTotal int
}

// currentResolution re-resolves the open turn without mutating it.
func (f *editFixture) currentResolution(t *testing.T) *service.TurnResolution {
	t.Helper()
	return f.session.ResolveTurn(f.openTurn())
}

// openTurn reads the session's currently open turn.
func (f *editFixture) openTurn() *turn.Turn {
	return f.session.GetActiveRound().CurrentTurn()
}

// newOpenAttackFixture builds a session with an attacker, a victim and a third character,
// enqueues one Sword attack from the attacker against the victim, and opens it — exactly the
// state a master's condition edit lands on.
//
// The dice budget: one Sword attack, in the session's default Free round mode, rolls the hit
// (a 2D10 test — 4 faces, Primary AND Secondary, because Roll always rolls both even with no
// advantage) and then the weapon's own damage dice (Sword is D10+D4, Primary only — damage has
// no advantage). Free mode never rolls actionSpeed, so that is the whole budget: 6 faces.
//
// Secondary is scripted strictly higher than Primary on the hit check, so a later advantage
// edit provably switches which already-rolled set is read.
func newOpenAttackFixture(t *testing.T) *editFixture {
	t.Helper()

	f := &editFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		attackerID: uuid.New(),
		victimID:   uuid.New(),
		thirdID:    uuid.New(),
	}

	// All three characters belong to the same player — enough here, exactly as
	// newCombatFixture: authorization is per player and only one client ever sends the attack.
	attacker := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID},
	}
	victim := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &f.playerUUID},
	}
	third := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.thirdID, PlayerUUID: &f.playerUUID},
	}

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.attackerID: newEditSheet(t),
		f.victimID:   newEditSheet(t),
		f.thirdID:    newEditSheet(t),
	}
	session := matchsession.NewMatchSession(
		f.matchUUID, sheets, []*matchDomain.Participant{attacker, victim, third},
	)

	f.rollSource = &scriptedFaces{faces: []int{
		3, 4, // Hit Primary: sum 7
		9, 8, // Hit Secondary: sum 17 — strictly higher, so advantage must switch sets
		5, 2, // Sword damage: D10, D4
	}}
	session.SetRollSource(f.rollSource)
	f.session = session

	weapon := enum.Sword
	act := action.NewAction(
		f.attackerID, []uuid.UUID{f.victimID}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil,
		&action.Attack{
			Weapon: &weapon,
			Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
			Damage: action.RollCheck{SkillName: enum.Push.String()},
		},
		nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(f.playerUUID, act); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}

	tr, err := session.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	openedAction := tr.Opened.GetAction()
	f.actionID = openedAction.GetID()
	f.facesAtOpen = f.rollSource.consumed()
	f.primaryTotal = f.currentResolution(t).ActionResult.Total
	return f
}

func newEditSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Combatant", FullName: "Edit Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

// ─── the tests ──────────────────────────────────────────────────────────────

func TestEditActionRollCondition(t *testing.T) {
	t.Run("a flat modifier moves the total without touching the dice", func(t *testing.T) {
		f := newOpenAttackFixture(t) // one open turn, attack vs one target, scripted dice
		before := f.currentResolution(t)
		diceBefore := append([]int(nil), before.ActionResult.DiceRolled...)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field:     action.FieldHit,
			Condition: action.RollCondition{Modifier: 3, Description: "creative positioning"},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got, want := res.Resolution.ActionResult.Total, before.ActionResult.Total+3; got != want {
			t.Fatalf("Total = %d, want %d", got, want)
		}
		if !slices.Equal(res.Resolution.ActionResult.DiceRolled, diceBefore) {
			t.Fatalf("the dice changed: %v → %v", diceBefore, res.Resolution.ActionResult.DiceRolled)
		}
	})

	t.Run("advantage reads the other set that was already rolled", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldHit, Condition: action.RollCondition{Bias: 1},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		// The fixture scripts Secondary strictly higher than Primary, so advantage must
		// switch which set is read — proving RollAttempts is why the edit needs no re-roll.
		if res.Resolution.ActionResult.Total <= f.primaryTotal {
			t.Fatalf("advantage did not switch to the better set (total %d)",
				res.Resolution.ActionResult.Total)
		}
		if f.rollSource.consumed() != f.facesAtOpen {
			t.Fatalf("the edit consumed %d extra faces — a die was re-rolled",
				f.rollSource.consumed()-f.facesAtOpen)
		}
	})

	t.Run("the edit never moves the economy", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		balanceBefore, speedsBefore := f.session.BarState(f.attackerID, action.BarAction)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldSpeed, Condition: action.RollCondition{Modifier: 12},
		}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		balanceAfter, speedsAfter := f.session.BarState(f.attackerID, action.BarAction)
		if balanceAfter != balanceBefore || !slices.Equal(speedsAfter, speedsBefore) {
			t.Fatalf("the edit rewrote the economy: %v/%v → %v/%v",
				balanceBefore, speedsBefore, balanceAfter, speedsAfter)
		}
	})

	t.Run("only the master edits", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(),
			action.NewMasterAction())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
	})
}
