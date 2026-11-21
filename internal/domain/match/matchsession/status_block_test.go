package matchsession_test

import (
	"errors"
	"strings"
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Peças que bloqueiam (decisões 8 e 9 do plano): o que os status de um personagem proíbem a
// sessão recusa, tira da fila e senta na reação padrão.

func effectOf(charID uuid.UUID, kind statuseffect.Kind, p statuseffect.Params) statuseffect.Effect {
	return statuseffect.Effect{ID: uuid.New(), CharacterID: charID, Kind: kind, Params: p}
}

func paralyzedIn(charID uuid.UUID, ds ...statuseffect.Domain) statuseffect.Effect {
	return effectOf(charID, statuseffect.KindParalyzed, statuseffect.Params{Domains: ds, Points: 3})
}

func kindOnly(charID uuid.UUID, k statuseffect.Kind) statuseffect.Effect {
	return effectOf(charID, k, statuseffect.Params{Points: 3})
}

func setEffects(s *matchsession.MatchSession, charID uuid.UUID, es ...statuseffect.Effect) {
	s.ReplaceStatusEffects(charID, es)
}

func attackBy(actor, target uuid.UUID) *action.Action {
	sword := enum.Sword
	return action.NewAction(actor, []uuid.UUID{target}, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}},
		nil, nil, nil, nil)
}

func moveBy(actor uuid.UUID) *action.Action {
	return action.NewAction(actor, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, Position: [3]int{3, 3, 0}}, nil, nil, nil, nil, nil)
}

// reactionBy monta uma reação do tipo pedido com a forma que o mapper do fio dá a ela.
func reactionBy(actor, reactTo uuid.UUID, k action.ReactionKind) *action.Action {
	var dodge *action.Dodge
	var move *action.Move
	var repel *action.Repel
	var skills []action.Skill
	for _, c := range k.RequiredComponents() {
		switch c {
		case action.ComponentDodge:
			dodge = &action.Dodge{RollCheck: action.RollCheck{SkillName: enum.Reflex.String()}}
		case action.ComponentMove:
			cat, _ := k.RequiredMoveCategory()
			move = &action.Move{Category: cat, Position: [3]int{5, 5, 0}}
		case action.ComponentRepel:
			repel = &action.Repel{RollCheck: action.RollCheck{SkillName: enum.Repel.String()}}
		}
	}
	if k.RequiresEvasionSkill() {
		skills = []action.Skill{{SkillName: enum.Evasion.String()}}
	}
	r := action.NewAction(actor, nil, reactTo, skills, action.ActionSpeed{},
		nil, move, nil, nil, dodge, nil, nil)
	r.Repel = repel
	r.ReactionKind = k
	return r
}

func TestStatusRefusal(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	c, other := uuid.New(), uuid.New()
	attackID := uuid.New()

	type probe struct {
		name    string
		a       *action.Action
		refused bool
	}
	react := func(k action.ReactionKind) *action.Action { return reactionBy(c, attackID, k) }
	cases := []struct {
		name    string
		effects []statuseffect.Effect
		want    string // tipo que a mensagem tem que nomear
		probes  []probe
	}{
		{
			name: "stunned", want: "stunned",
			effects: []statuseffect.Effect{kindOnly(c, statuseffect.KindStunned)},
			probes: []probe{
				{"attack", attackBy(c, other), true},
				{"dodge reaction", react(action.ReactDodge), true},
				{"default reaction", react(action.ReactDefault), true},
				{"nothing", react(action.ReactNothing), false},
			},
		},
		{
			name: "paralyzed attack", want: "paralyzed",
			effects: []statuseffect.Effect{paralyzedIn(c, statuseffect.DomainAttack)},
			probes: []probe{
				{"attack", attackBy(c, other), true},
				{"move", moveBy(c), false},
			},
		},
		{
			name: "restrained", want: "restrained",
			effects: []statuseffect.Effect{kindOnly(c, statuseffect.KindRestrained)},
			probes: []probe{
				{"move", moveBy(c), true},
				{"escape", react(action.ReactEscape), true},
				{"dodge", react(action.ReactDodge), false},
				{"attack", attackBy(c, other), false},
			},
		},
		{
			name: "openGuard", want: "openGuard",
			effects: []statuseffect.Effect{kindOnly(c, statuseffect.KindOpenGuard)},
			probes: []probe{
				{"dodge", react(action.ReactDodge), true},
				{"closedDodge", react(action.ReactClosedDodge), true},
				{"repel", react(action.ReactRepel), true},
				{"escape", react(action.ReactEscape), false},
				{"default keeps the defense", react(action.ReactDefault), false},
			},
		},
		{
			name: "airborne", want: "airborne",
			effects: []statuseffect.Effect{kindOnly(c, statuseffect.KindAirborne)},
			probes: []probe{
				{"escape", react(action.ReactEscape), true},
				{"closedEscape", react(action.ReactClosedEscape), true},
				{"escapeGuard", react(action.ReactEscapeGuard), true},
				{"repel", react(action.ReactRepel), false},
			},
		},
		{
			name: "paralyzed dodge", want: "paralyzed",
			effects: []statuseffect.Effect{paralyzedIn(c, statuseffect.DomainDodge)},
			probes: []probe{
				{"escape needs dodge", react(action.ReactEscape), true},
				{"dodge", react(action.ReactDodge), true},
				{"repel", react(action.ReactRepel), false},
				{"move", moveBy(c), false},
			},
		},
		{
			name: "paralyzed defense", want: "paralyzed",
			effects: []statuseffect.Effect{paralyzedIn(c, statuseffect.DomainDefense)},
			probes: []probe{
				{"repel", react(action.ReactRepel), true},
				{"default keeps the reflex", react(action.ReactDefault), false},
			},
		},
		{
			name: "openGuard and paralyzed defense", want: "openGuard",
			effects: []statuseffect.Effect{
				kindOnly(c, statuseffect.KindOpenGuard), paralyzedIn(c, statuseffect.DomainDefense),
			},
			probes: []probe{
				{"default with both passives gone", react(action.ReactDefault), true},
				{"nothing", react(action.ReactNothing), false},
			},
		},
		{
			name: "no statuses", effects: nil,
			probes: []probe{
				{"attack", attackBy(c, other), false},
				{"escape", react(action.ReactEscape), false},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEffects(s, c, tc.effects...)
			for _, p := range tc.probes {
				err := s.StatusRefusal(p.a)
				if !p.refused {
					if err != nil {
						t.Errorf("%s: refused (%v), want allowed", p.name, err)
					}
					continue
				}
				if !errors.Is(err, matchsession.ErrBlockedByStatus) {
					t.Errorf("%s: err = %v, want ErrBlockedByStatus", p.name, err)
					continue
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Errorf("%s: message %q does not name %q", p.name, err.Error(), tc.want)
				}
			}
		})
	}
}

func TestStatusRefusal_EnqueueAndAttachRefuse(t *testing.T) {
	f := newDefaultFixture(t)
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	before := len(f.session.PendingActions())

	err := f.session.EnqueueAction(f.playerB, attackBy(f.b, f.c))
	if !errors.Is(err, matchsession.ErrBlockedByStatus) {
		t.Fatalf("EnqueueAction err = %v, want ErrBlockedByStatus", err)
	}
	if got := len(f.session.PendingActions()); got != before {
		t.Fatalf("queue grew from %d to %d on a refusal", before, got)
	}
	if _, err := f.react(t, f.playerB, f.b, action.ReactDodge); !errors.Is(err, matchsession.ErrBlockedByStatus) {
		t.Fatalf("AttachReaction err = %v, want ErrBlockedByStatus", err)
	}
	if r := f.turn.SeatedFor(f.b); r == nil {
		t.Fatal("the refused reaction took the seat")
	}
	r := reactionBy(f.b, f.actionID(), action.ReactNothing)
	if _, err := f.session.AttachReaction(f.playerB, r); err != nil {
		t.Fatalf("a nothing reaction is always allowed: %v", err)
	}
}

func TestDropRefusedByStatus(t *testing.T) {
	t.Run("queued actions of the character leave, another character's stay", func(t *testing.T) {
		f := newDefaultFixture(t)
		bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
		bMove := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		cAttack := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))

		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		removed := f.session.DropRefusedByStatus(f.b)

		got := idsOf(removed)
		if len(got) != 2 || got[0] != bAttack.GetID() || got[1] != bMove.GetID() {
			t.Fatalf("removed = %v, want [%v %v]", got, bAttack.GetID(), bMove.GetID())
		}
		pending := pendingIDsOf(f.session)
		if len(pending) != 1 || pending[0] != cAttack.GetID() {
			t.Fatalf("queue = %v, want only C's attack %v", pending, cAttack.GetID())
		}
		if again := f.session.DropRefusedByStatus(f.c); len(again) != 0 {
			t.Fatalf("C has no status and lost %v", idsOf(again))
		}
	})

	t.Run("an allowed queued action stays", func(t *testing.T) {
		f := newDefaultFixture(t)
		bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
		mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindRestrained))

		removed := f.session.DropRefusedByStatus(f.b)
		if len(removed) != 1 || removed[0].Move == nil {
			t.Fatalf("removed = %v, want only the move", idsOf(removed))
		}
		if pending := pendingIDsOf(f.session); len(pending) != 1 || pending[0] != bAttack.GetID() {
			t.Fatalf("queue = %v, want the attack %v", pending, bAttack.GetID())
		}
	})

	t.Run("an unopened reaction is replaced by the seated nothing, an opened one stays", func(t *testing.T) {
		f := newDefaultFixture(t)
		bDodge, err := f.react(t, f.playerB, f.b, action.ReactDodge)
		if err != nil {
			t.Fatalf("B dodge: %v", err)
		}
		cDodge, err := f.react(t, f.playerC, f.c, action.ReactDodge)
		if err != nil {
			t.Fatalf("C dodge: %v", err)
		}
		if _, _, err := f.session.OpenReaction(cDodge.GetID()); err != nil {
			t.Fatalf("OpenReaction: %v", err)
		}

		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		setEffects(f.session, f.c, kindOnly(f.c, statuseffect.KindStunned))
		if removed := f.session.DropRefusedByStatus(f.b); len(removed) != 0 {
			t.Fatalf("a replaced reaction is not a removed queued action: %v", idsOf(removed))
		}
		f.session.DropRefusedByStatus(f.c)

		if f.turn.ReactionRef(bDodge.GetID()) != nil {
			t.Fatal("B's unopened dodge is still on the turn")
		}
		seated := f.turn.SeatedFor(f.b)
		if seated == nil || seated.ReactionKind != action.ReactNothing {
			t.Fatalf("B seated = %+v, want a nothing", seated)
		}
		if f.turn.ReactionRef(cDodge.GetID()) == nil {
			t.Fatal("C's opened dodge left the turn")
		}
		if f.turn.SeatedFor(f.c) != nil {
			t.Fatal("C got a seat on top of an opened reaction")
		}
		if _, err := f.session.CloseOpenTurn(); err != nil {
			t.Fatalf("CloseOpenTurn: %v", err)
		}
	})

	t.Run("an unopened seated default is re-seated to the new set", func(t *testing.T) {
		f := newDefaultFixture(t)
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindOpenGuard))
		f.session.DropRefusedByStatus(f.b)
		seated := f.turn.SeatedFor(f.b)
		if seated == nil || seated.ReactionKind != action.ReactDefault || !seated.WithoutReflexDodge || seated.WithoutDefaultDefense {
			t.Fatalf("B seated = %+v, want a default with only the defense", seated)
		}

		// Tirar o status devolve a reação padrão inteira.
		setEffects(f.session, f.b)
		f.session.DropRefusedByStatus(f.b)
		seated = f.turn.SeatedFor(f.b)
		if seated == nil || seated.ReactionKind != action.ReactDefault || seated.WithoutReflexDodge {
			t.Fatalf("B seated after removal = %+v, want the whole default", seated)
		}
	})
}

// lastPlayerB é o jogador de B do último openAttackOn (os testes não rodam em paralelo).
var lastPlayerB uuid.UUID

// openAttackOn abre um ataque de A em B com os efeitos de B já postos.
func openAttackOn(t *testing.T, face int, effects func(b uuid.UUID) []statuseffect.Effect) (*matchsession.MatchSession, *turn.Turn, uuid.UUID) {
	t.Helper()
	matchUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA, pB := makeParticipant(matchUUID, &playerA), makeParticipant(matchUUID, &playerB)
	lastPlayerB = playerB
	attacker, b := pA.Sheet.UUID, pB.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{attacker: buildPlainSheet(t), b: buildPlainSheet(t)}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{pA, pB})
	s.SetRollSource(&countingSource{face: face})
	setEffects(s, b, effects(b)...)
	if err := s.EnqueueAction(playerA, attackBy(attacker, b)); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return s, mustOpen(t, s), b
}

func TestSeatDefaultReactionUnderStatus(t *testing.T) {
	cases := []struct {
		name                string
		effects             func(b uuid.UUID) []statuseffect.Effect
		kind                action.ReactionKind
		noReflex, noDefense bool
	}{
		{"openGuard leaves the defense only", func(b uuid.UUID) []statuseffect.Effect {
			return []statuseffect.Effect{kindOnly(b, statuseffect.KindOpenGuard)}
		}, action.ReactDefault, true, false},
		{"stunned seats a nothing", func(b uuid.UUID) []statuseffect.Effect {
			return []statuseffect.Effect{kindOnly(b, statuseffect.KindStunned)}
		}, action.ReactNothing, false, false},
		{"paralyzed defense leaves the reflex only", func(b uuid.UUID) []statuseffect.Effect {
			return []statuseffect.Effect{paralyzedIn(b, statuseffect.DomainDefense)}
		}, action.ReactDefault, false, true},
		{"both passives gone seat a nothing", func(b uuid.UUID) []statuseffect.Effect {
			return []statuseffect.Effect{kindOnly(b, statuseffect.KindOpenGuard), paralyzedIn(b, statuseffect.DomainDefense)}
		}, action.ReactNothing, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, tr, b := openAttackOn(t, 1, tc.effects)
			seated := tr.SeatedFor(b)
			if seated == nil {
				t.Fatal("no seated reaction")
			}
			if seated.ReactionKind != tc.kind || seated.WithoutReflexDodge != tc.noReflex || seated.WithoutDefaultDefense != tc.noDefense {
				t.Fatalf("seated = kind %s noReflex %v noDefense %v, want %s %v %v",
					seated.ReactionKind, seated.WithoutReflexDodge, seated.WithoutDefaultDefense,
					tc.kind, tc.noReflex, tc.noDefense)
			}
			res := s.ResolveTurn(tr)
			if len(res.Errors) != 0 {
				t.Fatalf("resolution errors: %+v", res.Errors)
			}
			var found bool
			for _, cr := range res.CharacterResults {
				if cr.TargetID == b {
					found = true
					if cr.ReactionKind != string(tc.kind) {
						t.Errorf("result reaction kind = %s, want %s", cr.ReactionKind, tc.kind)
					}
				}
			}
			if !found {
				t.Fatal("no result for B")
			}
			if _, err := s.CloseOpenTurn(); err != nil {
				t.Fatalf("CloseOpenTurn: %v", err)
			}
		})
	}
}

// Decisão do controlador (após o relatório): a defesa padrão também sai das reações enviadas que
// a mantêm, quando um status trava a defesa (§5.4, Paralisado — "Defesa: conforme (defesa)").
func TestAttachedReactionLosesDefaultDefenseUnderStatus(t *testing.T) {
	paralyzedDefense := func(b uuid.UUID) []statuseffect.Effect {
		return []statuseffect.Effect{paralyzedIn(b, statuseffect.DomainDefense)}
	}
	for _, k := range []action.ReactionKind{action.ReactDodge, action.ReactEscapeGuard} {
		t.Run(string(k), func(t *testing.T) {
			// Todo dado mostra 9: o acerto é 18; a esquiva enviada já caiu em 1+1 e falha. Sem a
			// trava, a defesa padrão passiva (11) contra CD 8 ampararia.
			s, tr, b := openAttackOn(t, 9, paralyzedDefense)
			a := tr.GetAction()
			r := reactionBy(b, a.GetID(), k)
			r.Dodge.Attempts = action.RollAttempts{Primary: []int{1, 1}}
			if _, err := s.AttachReaction(lastPlayerB, r); err != nil {
				t.Fatalf("AttachReaction(%s): %v", k, err)
			}
			live := tr.ReactionRef(r.GetID())
			if live == nil || !live.WithoutDefaultDefense || live.WithoutReflexDodge {
				t.Fatalf("attached = %+v, want WithoutDefaultDefense only", live)
			}
			for _, cr := range s.ResolveTurn(tr).CharacterResults {
				if cr.TargetID != b {
					continue
				}
				if cr.Avoided {
					t.Fatal("the fixture is wrong: the dodge was meant to fail")
				}
				if cr.Defended {
					t.Fatal("a failed dodge fell back on a default defense the status removed")
				}
				return
			}
			t.Fatal("no result for B")
		})
	}

	t.Run("without the status the defense still stands", func(t *testing.T) {
		s, tr, b := openAttackOn(t, 9, func(uuid.UUID) []statuseffect.Effect { return nil })
		a := tr.GetAction()
		r := reactionBy(b, a.GetID(), action.ReactDodge)
		r.Dodge.Attempts = action.RollAttempts{Primary: []int{1, 1}}
		if _, err := s.AttachReaction(lastPlayerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		for _, cr := range s.ResolveTurn(tr).CharacterResults {
			if cr.TargetID == b && (cr.Avoided || !cr.Defended) {
				t.Fatalf("avoided %v defended %v, want a failed dodge caught by the defense", cr.Avoided, cr.Defended)
			}
		}
	})
}

func TestStatusBlockError(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	c := uuid.New()
	setEffects(s, c, kindOnly(c, statuseffect.KindStunned))
	err := s.StatusRefusal(attackBy(c, uuid.New()))
	var be *matchsession.StatusBlockError
	if !errors.As(err, &be) || !errors.Is(err, matchsession.ErrBlockedByStatus) {
		t.Fatalf("err = %v, want a *StatusBlockError wrapping ErrBlockedByStatus", err)
	}
	if be.CharacterID != c || be.What != "agir" || len(be.Kinds) != 1 || be.Kinds[0] != statuseffect.KindStunned {
		t.Fatalf("cause = %+v", be)
	}
	if got := be.Message(nil); strings.Contains(got, "stunned") || !strings.Contains(got, "um efeito") {
		t.Fatalf("Message(nil) = %q, want the generic cause", got)
	}
}
