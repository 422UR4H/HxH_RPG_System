package matchsession_test

import (
	"errors"
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// ─── Editing an action still in the queue (owner decision, 2026-10-12) ─────

type queueFixture struct {
	session          *matchsession.MatchSession
	master           uuid.UUID
	playerA, playerB uuid.UUID
	a, b             uuid.UUID
	first, second    *action.Action // A's attack on B, then B's attack on A — both queued, Free
}

func newQueueFixture(t *testing.T) *queueFixture {
	t.Helper()
	matchUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA, pB := makeParticipant(matchUUID, &playerA), makeParticipant(matchUUID, &playerB)
	a, b := pA.Sheet.UUID, pB.Sheet.UUID
	s := matchsession.NewMatchSession(matchUUID,
		map[uuid.UUID]*csSheet.CharacterSheet{a: buildPlainSheet(t), b: buildPlainSheet(t)},
		[]*match.Participant{pA, pB})
	s.SetRollSource(&countingSource{face: 5})
	attack := func(player, actor, target uuid.UUID) *action.Action {
		sword := enum.Sword
		atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
		act := action.NewAction(actor, []uuid.UUID{target}, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, atk, nil, nil, nil, nil)
		if err := s.EnqueueAction(player, act); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		return act
	}
	f := &queueFixture{session: s, master: uuid.New(), playerA: playerA, playerB: playerB, a: a, b: b}
	f.first = attack(playerA, a, b)
	f.second = attack(playerB, b, a)
	return f
}

func (f *queueFixture) edit(t *testing.T, ma *action.MasterAction) error {
	t.Helper()
	_, err := f.session.ApplyMasterAction(ma, f.master)
	return err
}

func conditionOn(id uuid.UUID, field action.ConditionField, cond action.RollCondition) *action.MasterAction {
	ma := action.NewMasterAction()
	ma.ActionID = id
	ma.Conditions = []action.ConditionEdit{{Field: field, Condition: cond}}
	return ma
}

func TestQueuedEdit_LandsOnTheActionAndRidesIntoItsTurn(t *testing.T) {
	f := newQueueFixture(t)
	if !f.session.IsQueued(f.first.GetID()) {
		t.Fatal("test setup: the first attack must be queued")
	}
	res, err := f.session.ApplyMasterAction(
		conditionOn(f.first.GetID(), action.FieldHit, action.RollCondition{Modifier: 4}), f.master)
	if err != nil || res != nil {
		t.Fatalf("ApplyMasterAction on the queue = (%v, %v), want (nil, nil): no turn to resolve", res, err)
	}
	if c := f.first.Attack.Hit.Context.Condition; c == nil || c.Modifier != 4 {
		t.Fatalf("hit condition = %+v, want the master's +4 on the queued action itself", c)
	}

	opened := mustOpen(t, f.session)
	act := opened.GetAction()
	if act.GetID() != f.first.GetID() {
		t.Fatalf("opened %v, want the first attack", act.GetID())
	}
	var captured bool
	for _, ov := range f.session.PeekOverridesFor(opened) {
		if ov.ActionID == f.first.GetID() && ov.Field == "hit.condition" {
			captured = true
		}
	}
	if !captured {
		t.Fatal("the capture of a queued edit must be the turn's once its action opens")
	}
}

func TestQueuedEdit_TheSpeedMovesTheQueue(t *testing.T) {
	f := newQueueFixture(t)
	// Free: both speeds are the passive 11 and the earlier one wins the tie. +5 on the
	// second puts it first.
	if err := f.edit(t, conditionOn(f.second.GetID(), action.FieldSpeed, action.RollCondition{Modifier: 5})); err != nil {
		t.Fatalf("ApplyMasterAction(speed): %v", err)
	}
	if f.second.Speed.Result != f.first.Speed.Result+5 {
		t.Fatalf("speeds = %d and %d, want the second 5 ahead", f.first.Speed.Result, f.second.Speed.Result)
	}
	opened := mustOpen(t, f.session)
	if act := opened.GetAction(); act.GetID() != f.second.GetID() {
		t.Fatal("the edited speed must reorder the queue: the second attack opens first")
	}
}

func TestQueuedEdit_TargetsAndSkillsAreRefused(t *testing.T) {
	f := newQueueFixture(t)
	targets := action.NewMasterAction()
	targets.ActionID = f.first.GetID()
	targets.TargetID = []uuid.UUID{f.a}
	if err := f.edit(t, targets); !errors.Is(err, matchsession.ErrQueuedEditUnsupported) {
		t.Fatalf("targets: err = %v, want ErrQueuedEditUnsupported", err)
	}
	skills := action.NewMasterAction()
	skills.ActionID = f.first.GetID()
	skills.Skills = []action.Skill{}
	if err := f.edit(t, skills); !errors.Is(err, matchsession.ErrQueuedEditUnsupported) {
		t.Fatalf("skills: err = %v, want ErrQueuedEditUnsupported", err)
	}
	if len(f.first.TargetID) != 1 || f.first.TargetID[0] != f.b {
		t.Fatal("a refused edit must not touch the action")
	}
}

func TestQueuedEdit_CancellingTakesTheEditsWithIt(t *testing.T) {
	f := newQueueFixture(t)
	if err := f.edit(t, conditionOn(f.first.GetID(), action.FieldSpeed, action.RollCondition{Modifier: 7})); err != nil {
		t.Fatalf("ApplyMasterAction(speed): %v", err)
	}
	damage := action.NewMasterAction()
	damage.ActionID = f.first.GetID()
	grab := enum.Grab
	damage.DamageSkill = &grab
	if err := f.edit(t, damage); err != nil {
		t.Fatalf("ApplyMasterAction(damageSkill): %v", err)
	}
	speed := f.first.Speed.Result

	cancelled, err := f.session.CancelAction(f.playerA, f.first.GetID())
	if err != nil {
		t.Fatalf("CancelAction: %v", err)
	}
	if cancelled.Speed.Context.Condition != nil || cancelled.Speed.Result != speed-7 {
		t.Fatalf("speed = %d with condition %+v, want the declared %d and none", cancelled.Speed.Result,
			cancelled.Speed.Context.Condition, speed-7)
	}
	if got := cancelled.Attack.EffectiveDamageSkill(); got != enum.Push {
		t.Fatalf("damage skill = %v, want Push back — the edit goes with the action", got)
	}
}

func TestQueuedEdit_AReactionThatConsumesItTakesTheEditsWithIt(t *testing.T) {
	f := newQueueFixture(t)
	// The first attack opens; B's own queued attack is what B's repel consumes.
	opened := mustOpen(t, f.session)
	if err := f.edit(t, conditionOn(f.second.GetID(), action.FieldHit, action.RollCondition{Bias: 1})); err != nil {
		t.Fatalf("ApplyMasterAction(hit on the queue): %v", err)
	}
	act := opened.GetAction()
	repel := action.NewAction(f.b, nil, act.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)
	repel.ReactionKind = action.ReactRepel
	repel.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: enum.Repel.String()}}
	if _, err := f.session.AttachReaction(f.playerB, repel); err != nil {
		t.Fatalf("AttachReaction(repel): %v", err)
	}
	if len(repel.ConsumedActionIDs) != 1 || repel.ConsumedActionIDs[0] != f.second.GetID() {
		t.Fatalf("consumed = %v, want B's queued attack", repel.ConsumedActionIDs)
	}
	if f.second.Attack.Hit.Context.Condition != nil {
		t.Fatal("the consumed action must leave as it was declared")
	}
}

// ─── Doing sooner what the queue already said is not a swap (2026-10-13) ───

func TestSwap_RepellingTheOneYourQueuedAttackWasAimedAt(t *testing.T) {
	repelOn := func(t *testing.T, f *queueFixture) *action.Action {
		t.Helper()
		opened := mustOpen(t, f.session) // A's attack on B
		act := opened.GetAction()
		repel := action.NewAction(f.b, nil, act.GetID(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		repel.ReactionKind = action.ReactRepel
		repel.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: enum.Repel.String()}}
		if _, err := f.session.AttachReaction(f.playerB, repel); err != nil {
			t.Fatalf("AttachReaction(repel): %v", err)
		}
		if len(repel.ConsumedActionIDs) != 1 {
			t.Fatalf("consumed = %v, want B's queued action", repel.ConsumedActionIDs)
		}
		return repel
	}

	t.Run("B's queued attack was on A, who attacks them: no Disadvantage", func(t *testing.T) {
		f := newQueueFixture(t) // second = B's attack on A
		if r := repelOn(t, f); r.SystemBias != 0 {
			t.Fatalf("SystemBias = %d, want 0 — the repel is the blow B had queued", r.SystemBias)
		}
	})

	t.Run("B's queued attack was on someone else: a swap, Disadvantage", func(t *testing.T) {
		f := newQueueFixture(t)
		f.second.TargetID = []uuid.UUID{uuid.New()}
		if r := repelOn(t, f); r.SystemBias != -1 {
			t.Fatalf("SystemBias = %d, want -1 — repelling A is not what B had queued", r.SystemBias)
		}
	})
}

// The system's Disadvantage on a passive reading rolls it against the base value, like the
// master's bias (2026-10-13): a swapped closed escape steps with Shift, whose Brake is passive.
func TestSwap_TheSystemsDisadvantageRollsAPassiveReading(t *testing.T) {
	f := newQueueFixture(t) // every die shows 5: a 2D10 set is 10, under the base 11
	opened := mustOpen(t, f.session)
	act := opened.GetAction()
	// A closed escape charges only the move bar, so what it consumes is a queued move: give B a
	// Shift to somewhere else.
	shift := action.NewAction(f.b, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Shift, Position: [3]int{5, 5, 0}}, nil, nil, nil, nil, nil)
	if err := f.session.EnqueueAction(f.playerB, shift); err != nil {
		t.Fatalf("EnqueueAction(shift): %v", err)
	}
	escape := action.NewAction(f.b, nil, act.GetID(), nil, action.ActionSpeed{}, nil,
		&action.Move{Category: enum.Shift, Position: [3]int{8, 8, 0}}, nil, nil,
		&action.Dodge{RollCheck: action.RollCheck{SkillName: enum.Reflex.String()}}, nil, nil)
	escape.ReactionKind = action.ReactClosedEscape
	if _, err := f.session.AttachReaction(f.playerB, escape); err != nil {
		t.Fatalf("AttachReaction(closedEscape): %v", err)
	}
	if escape.SystemBias != -1 {
		t.Fatalf("SystemBias = %d, want -1 — fleeing elsewhere is a swap", escape.SystemBias)
	}
	if escape.Move.Speed == nil || len(escape.Move.Speed.Attempts.Primary) != 2 {
		t.Fatalf("Move.Speed = %+v, want the Shift's dice rolled by the system's bias", escape.Move.Speed)
	}
	if escape.Move.FinalSpeed != 10 {
		t.Fatalf("FinalSpeed = %d, want 10 — the worse of the dice (5+5) and the base 11", escape.Move.FinalSpeed)
	}
}
