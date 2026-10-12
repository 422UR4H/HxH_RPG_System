package matchsession_test

import (
	"errors"
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// ─── The default reaction (owner decision, 2026-10-12) ─────────────────────
//
// Every character an attack names gets a default reaction the moment the turn opens — reflex
// dodge, then the default defense, at the dice set's base value. It is a real reaction: the
// master edits it, the history records it, and what the target sends replaces it.

type defaultFixture struct {
	session  *matchsession.MatchSession
	source   *countingSource
	master   uuid.UUID
	playerB  uuid.UUID
	playerC  uuid.UUID
	attacker uuid.UUID
	b, c     uuid.UUID
	turn     *turn.Turn
}

// newDefaultFixture opens one Sword attack from A on B and C. Every die shows 9.
func newDefaultFixture(t *testing.T) *defaultFixture {
	t.Helper()
	matchUUID := uuid.New()
	playerA, playerB, playerC := uuid.New(), uuid.New(), uuid.New()
	pA, pB, pC := makeParticipant(matchUUID, &playerA), makeParticipant(matchUUID, &playerB), makeParticipant(matchUUID, &playerC)
	attacker, b, c := pA.Sheet.UUID, pB.Sheet.UUID, pC.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attacker: buildPlainSheet(t), b: buildPlainSheet(t), c: buildPlainSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{pA, pB, pC})
	src := &countingSource{face: 9}
	s.SetRollSource(src)

	sword := enum.Sword
	atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
	a := action.NewAction(attacker, []uuid.UUID{b, c}, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, atk, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerA, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return &defaultFixture{
		session: s, source: src, master: uuid.New(), playerB: playerB, playerC: playerC,
		attacker: attacker, b: b, c: c, turn: mustOpen(t, s),
	}
}

func (f *defaultFixture) seatedOf(t *testing.T, actor uuid.UUID) action.Action {
	t.Helper()
	r := f.turn.SeatedFor(actor)
	if r == nil {
		t.Fatalf("no seated reaction for %v", actor)
	}
	return *r
}

func (f *defaultFixture) resultOf(t *testing.T, actor uuid.UUID) service.CharacterResult {
	t.Helper()
	for _, cr := range f.session.ResolveTurn(f.turn).CharacterResults {
		if cr.TargetID == actor {
			return cr
		}
	}
	t.Fatalf("no result for %v", actor)
	return service.CharacterResult{}
}

func (f *defaultFixture) edit(t *testing.T, actionID uuid.UUID, field action.ConditionField, cond action.RollCondition) {
	t.Helper()
	ma := action.NewMasterAction()
	ma.ActionID = actionID
	ma.Conditions = []action.ConditionEdit{{Field: field, Condition: cond}}
	if _, err := f.session.ApplyMasterAction(ma, f.master); err != nil {
		t.Fatalf("ApplyMasterAction(%s): %v", field, err)
	}
}

func (f *defaultFixture) actionID() uuid.UUID {
	a := f.turn.GetAction()
	return a.GetID()
}

func (f *defaultFixture) react(t *testing.T, player, actor uuid.UUID, kind action.ReactionKind) (*action.Action, error) {
	t.Helper()
	r := action.NewAction(actor, nil, f.actionID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, &action.Dodge{RollCheck: action.RollCheck{SkillName: enum.Reflex.String()}}, nil, nil)
	r.ReactionKind = kind
	_, err := f.session.AttachReaction(player, r)
	return r, err
}

func TestDefaultReaction_SeatedWhenTheTurnOpens(t *testing.T) {
	f := newDefaultFixture(t)

	t.Run("one per character the attack names, rolling nothing", func(t *testing.T) {
		rolls := f.source.rolls
		for _, id := range []uuid.UUID{f.b, f.c} {
			r := f.seatedOf(t, id)
			if r.ReactionKind != action.ReactDefault || r.ReactToID != f.actionID() {
				t.Fatalf("seated = %+v, want a default reaction to the attack", r)
			}
			if !r.Dodge.Attempts.IsEmpty() {
				t.Fatal("a default reaction rolls nothing until the master biases it")
			}
		}
		if f.source.rolls != rolls {
			t.Fatalf("rolls moved from %d to %d", rolls, f.source.rolls)
		}
	})

	t.Run("it is read in the calculation, passive, and listed for the floor as seated", func(t *testing.T) {
		cr := f.resultOf(t, f.b)
		if cr.ReactionKind != string(action.ReactDefault) || !cr.Dodge.Passive || cr.Dodge.Total != 11 {
			t.Fatalf("B = %+v, want the default reaction's passive dodge of 11", cr)
		}
		res := f.session.ResolveTurn(f.turn)
		if len(res.PendingReactions) != 2 || !res.PendingReactions[0].Seated {
			t.Fatalf("PendingReactions = %+v, want both seated defaults", res.PendingReactions)
		}
	})

	t.Run("it does not hold the close", func(t *testing.T) {
		if got := f.session.ReactionsAwaitingTheFloor(); len(got) != 0 {
			t.Fatalf("ReactionsAwaitingTheFloor = %d, want 0", len(got))
		}
	})
}

func TestDefaultReaction_TheMastersBiasRollsItOnce(t *testing.T) {
	f := newDefaultFixture(t)
	seated := f.seatedOf(t, f.b)
	rolls := f.source.rolls

	f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{Bias: 1})
	if got := f.source.rolls - rolls; got != 2 {
		t.Fatalf("the first bias rolled %d dice, want 2 — the primary 2D10 set", got)
	}
	if cr := f.resultOf(t, f.b); cr.Dodge.Total != 18 {
		t.Fatalf("Dodge = %d, want 18 — advantage keeps the better of 9+9 and the base 11", cr.Dodge.Total)
	}

	f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{Bias: -1})
	f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{})
	f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{Bias: 1})
	if got := f.source.rolls - rolls; got != 2 {
		t.Fatalf("re-biasing rolled again: %d dice in all, want 2", got)
	}

	t.Run("the default defense rolls on its own first bias, and disadvantage keeps the worse", func(t *testing.T) {
		before := f.source.rolls
		f.edit(t, seated.GetID(), action.FieldDefense, action.RollCondition{Bias: -1})
		if f.source.rolls-before != 2 {
			t.Fatalf("rolled %d dice for the defense, want 2", f.source.rolls-before)
		}
		// The defense is only derived when the dodge misses; B's dodge of 18 beats the hit,
		// so read C's instead with a dodge that cannot.
		sc := f.seatedOf(t, f.c)
		f.edit(t, sc.GetID(), action.FieldDodge, action.RollCondition{Modifier: -100})
		f.edit(t, sc.GetID(), action.FieldDefense, action.RollCondition{Bias: -1})
		if cr := f.resultOf(t, f.c); cr.Defense.Total != 11 {
			t.Fatalf("Defense = %d, want 11 — disadvantage keeps the worse of 9+9 and the base", cr.Defense.Total)
		}
	})
}

func TestDefaultReaction_TheAnswerReplacesIt(t *testing.T) {
	t.Run("an active reaction replaces the seated one, and the master's edit on it goes", func(t *testing.T) {
		f := newDefaultFixture(t)
		seated := f.seatedOf(t, f.b)
		f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{Modifier: 3})

		r, err := f.react(t, f.playerB, f.b, action.ReactDodge)
		if err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if f.turn.ReactionRef(seated.GetID()) != nil || f.turn.SeatedFor(f.b) != nil {
			t.Fatal("the seated reaction must leave the turn")
		}
		cr := f.resultOf(t, f.b)
		if cr.ReactionID != r.GetID() || cr.ReactionKind != string(action.ReactDodge) {
			t.Fatalf("B = %+v, want the dodge they sent", cr)
		}
		for _, ov := range f.session.PeekOverridesFor(f.turn) {
			if ov.ActionID == seated.GetID() {
				t.Fatalf("a capture of the replaced seated reaction survived: %+v", ov)
			}
		}
		if _, err := f.react(t, f.playerB, f.b, action.ReactDodge); !errors.Is(err, matchsession.ErrReactorAlreadyReacted) {
			t.Fatalf("a second answer: err = %v, want ErrReactorAlreadyReacted", err)
		}
	})

	t.Run("declaring the default keeps the dice and the edits — nothing rolls again", func(t *testing.T) {
		f := newDefaultFixture(t)
		seated := f.seatedOf(t, f.b)
		f.edit(t, seated.GetID(), action.FieldDodge, action.RollCondition{Bias: 1, Modifier: 2})
		before := f.resultOf(t, f.b).Dodge.Total
		rolls := f.source.rolls

		r, err := f.react(t, f.playerB, f.b, action.ReactDefault)
		if err != nil {
			t.Fatalf("AttachReaction(default): %v", err)
		}
		if f.source.rolls != rolls {
			t.Fatalf("declaring the default rolled %d dice", f.source.rolls-rolls)
		}
		cr := f.resultOf(t, f.b)
		if cr.ReactionID != r.GetID() || cr.Dodge.Total != before {
			t.Fatalf("B = %+v, want their declared default with the same dodge %d", cr, before)
		}
		var moved bool
		for _, ov := range f.session.PeekOverridesFor(f.turn) {
			if ov.ActionID == r.GetID() && ov.Field == "dodge.condition" {
				moved = true
			}
		}
		if !moved {
			t.Fatal("the master's capture must move to the declared default")
		}
		if got := f.session.ReactionsAwaitingTheFloor(); len(got) != 1 {
			t.Fatalf("ReactionsAwaitingTheFloor = %d, want 1 — a declared default was sent", len(got))
		}
	})
}

func TestDefaultReaction_NothingHasNothingToNarrate(t *testing.T) {
	f := newDefaultFixture(t)
	r, err := f.react(t, f.playerB, f.b, action.ReactNothing)
	if err != nil {
		t.Fatalf("AttachReaction(nothing): %v", err)
	}
	if _, _, err := f.session.OpenReaction(r.GetID()); !errors.Is(err, matchsession.ErrNothingToNarrate) {
		t.Fatalf("OpenReaction(nothing): err = %v, want ErrNothingToNarrate", err)
	}
	if cr := f.resultOf(t, f.b); cr.ReactionKind != string(action.ReactNothing) || cr.Avoided {
		t.Fatalf("B = %+v, want the unopened nothing read — no passives", cr)
	}
	for _, pr := range f.session.ResolveTurn(f.turn).PendingReactions {
		if pr.ReactionID == r.GetID() {
			t.Fatal("a nothing is never listed for the floor")
		}
	}
}

func TestDefaultReaction_FollowsTheTargets(t *testing.T) {
	f := newDefaultFixture(t)
	retarget := func(ids ...uuid.UUID) {
		ma := action.NewMasterAction()
		ma.TargetID = ids
		if _, err := f.session.ApplyMasterAction(ma, f.master); err != nil {
			t.Fatalf("ApplyMasterAction(targets): %v", err)
		}
	}
	retarget(f.b)
	if f.turn.SeatedFor(f.c) != nil {
		t.Fatal("C is no longer named: their seated reaction must leave")
	}
	retarget(f.b, f.c)
	if f.turn.SeatedFor(f.c) == nil {
		t.Fatal("C is named again: they get a default reaction again")
	}
	if len(f.turn.GetReactions()) != 2 {
		t.Fatalf("reactions = %d, want 2 — B's was never touched", len(f.turn.GetReactions()))
	}
}
