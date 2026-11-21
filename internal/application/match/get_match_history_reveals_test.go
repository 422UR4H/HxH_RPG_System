package match_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

type fakeRevealReader struct {
	at    map[uuid.UUID]time.Time
	calls [][]uuid.UUID
}

func (f *fakeRevealReader) RevealedAt(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	f.calls = append(f.calls, append([]uuid.UUID(nil), ids...))
	out := map[uuid.UUID]time.Time{}
	for _, id := range ids {
		if t, ok := f.at[id]; ok {
			out[id] = t
		}
	}
	return out, nil
}

// Revelações do mestre no histórico (Task 16): o caso de uso lê as revelações uma vez por
// requisição e as usa na regra dos eventos de status (R2) e no escondido dos turnos (R4).
func TestGetMatchHistoryReveals(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	ownerUUID, reader, third := uuid.New(), uuid.New(), uuid.New()
	sceneUUID, roundUUID := uuid.New(), uuid.New()
	ownersPC, pc, npc := uuid.New(), uuid.New(), uuid.New()
	hid1, hid2 := uuid.New(), uuid.New()
	base := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }
	everyone := statuseffect.Audience{Everyone: true}

	type turnSpec struct {
		id            uuid.UUID
		actionID      uuid.UUID
		opened        int
		actor, target uuid.UUID
		hiddenFrom    match.HiddenFrom
		avoided       bool // o golpe no alvo não acertou
	}
	newTurn := func(opened int, actor, target uuid.UUID, hf match.HiddenFrom) turnSpec {
		return turnSpec{id: uuid.New(), actionID: uuid.New(), opened: opened, actor: actor, target: target, hiddenFrom: hf}
	}
	margin := 7 // a margem do ataque de um alvo só (total - margem = a esquiva do alvo)
	tree := func(s turnSpec) match.HistoryTurn {
		act := action.NewAction(s.actor, []uuid.UUID{s.target}, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil, action.WithReconstructedID(s.actionID))
		return match.HistoryTurn{
			UUID: s.id, CreatedAt: at(s.opened), FinishedAt: at(s.opened + 5), Action: *act,
			Resolution: &service.TurnResolution{
				IsSettled:        true,
				ActionResult:     service.RollResult{Total: 20, Margin: &margin},
				CharacterResults: []service.CharacterResult{{TargetID: s.target, ReactionKind: "dodge", Avoided: s.avoided}},
			},
			ActionHiddenFrom: s.hiddenFrom,
		}
	}
	event := func(eff uuid.UUID, sec int, turn *turnSpec, anchored bool, listing statuseffect.Audience,
		views []uuid.UUID, hf map[uuid.UUID]uuid.UUID) statuseffect.EventRecord {
		vs := map[uuid.UUID]masteraction.View{}
		for _, v := range views {
			vs[v] = masteraction.ViewFull
		}
		ev := statuseffect.Event{
			ID: uuid.New(), Change: statuseffect.ChangeApplied, MatchID: matchUUID, SceneID: sceneUUID,
			RoundID: roundUUID, Listing: listing, Views: vs, HiddenFrom: hf, HappenedAt: at(sec),
		}
		if turn != nil {
			id := turn.id
			ev.TurnID = &id
			if anchored {
				a := turn.actionID
				ev.ActionID = &a
			}
		}
		return statuseffect.EventRecord{Event: ev,
			Effect: statuseffect.Effect{ID: eff, CharacterID: pc, Kind: statuseffect.KindProne, AppliedAt: at(sec)}}
	}

	get := func(t *testing.T, turns []turnSpec, recs []statuseffect.EventRecord, latest map[uuid.UUID]statuseffect.Audience,
		rv *fakeRevealReader, who uuid.UUID) match.HistoryRound {
		t.Helper()
		var ht []match.HistoryTurn
		for _, s := range turns {
			ht = append(ht, tree(s))
		}
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return []*matchEntity.Participant{
						{Sheet: csEntity.Summary{UUID: ownersPC, PlayerUUID: &ownerUUID}},
						{Sheet: csEntity.Summary{UUID: pc, PlayerUUID: &third}},
					}, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return []match.HistoryScene{{UUID: sceneUUID, Rounds: []match.HistoryRound{{UUID: roundUUID, Turns: ht}}}}, nil
			}},
			&mockParticipationChecker{}, nil, nil, nil,
		).WithStatusEffectEvents(&fakeStatusEventLister{recs: recs, latest: latest})
		if rv != nil {
			uc.WithStatusEffectReveals(rv)
		}
		res, err := uc.Get(context.Background(), matchUUID, who)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		return res.Scenes[0].Rounds[0]
	}
	turnByID := func(ro match.HistoryRound, id uuid.UUID) *match.HistoryTurn {
		for i := range ro.Turns {
			if ro.Turns[i].UUID == id {
				return &ro.Turns[i]
			}
		}
		return nil
	}
	statusIDs := func(ro match.HistoryRound, turn *match.HistoryTurn) []uuid.UUID {
		out := []uuid.UUID{}
		for _, ev := range ro.Events {
			if ev.Kind == match.HistoryEventStatusEffect {
				out = append(out, ev.StatusEffect.UUID)
			}
		}
		if turn != nil {
			for _, s := range turn.StatusEffects {
				out = append(out, s.UUID)
			}
		}
		return out
	}

	t.Run("a status event hidden by its listing shows to everyone after the effect is revealed", func(t *testing.T) {
		eff := uuid.New()
		turn := newTurn(100, ownersPC, pc, nil)
		// o leitor viu o personagem (está em Views) mas a lista do efeito não o inclui
		rec := event(eff, 102, &turn, false, everyone, []uuid.UUID{reader}, nil)
		latest := map[uuid.UUID]statuseffect.Audience{eff: {Players: []uuid.UUID{third}}}
		for _, who := range []uuid.UUID{reader, ownerUUID} {
			ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, &fakeRevealReader{}, who)
			if ids := statusIDs(ro, turnByID(ro, turn.id)); len(ids) != 0 {
				t.Fatalf("%s before the reveal: %v, want none", who, ids)
			}
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{eff: at(200)}}
		for _, who := range []uuid.UUID{reader, ownerUUID} { // a revelação é para todos, em Views ou não
			ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, who)
			if ids := statusIDs(ro, turnByID(ro, turn.id)); !slices.Equal(ids, []uuid.UUID{rec.Event.ID}) {
				t.Fatalf("%s after the reveal: %v, want the event", who, ids)
			}
		}
		// uma revelação ANTES do evento não o cobre
		rv = &fakeRevealReader{at: map[uuid.UUID]time.Time{eff: at(101)}}
		ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, reader)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); len(ids) != 0 {
			t.Fatalf("a reveal before the event covered it: %v", ids)
		}
	})

	t.Run("a hidden event shows through HiddenFrom when the Hidden is revealed", func(t *testing.T) {
		eff := uuid.New()
		turn := newTurn(100, ownersPC, pc, nil)
		rec := event(eff, 102, &turn, false, everyone, nil, map[uuid.UUID]uuid.UUID{reader: hid1})
		latest := map[uuid.UUID]statuseffect.Audience{eff: everyone}
		ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, &fakeRevealReader{}, reader)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); len(ids) != 0 {
			t.Fatalf("before: %v", ids)
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(200)}}
		ro = get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, reader)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); !slices.Equal(ids, []uuid.UUID{rec.Event.ID}) {
			t.Fatalf("after: %v, want the event", ids)
		}
		// quem o Escondido não escondeu e que não viu o personagem segue sem o evento
		ro = get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, third)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); len(ids) != 0 {
			t.Fatalf("a reader the Hidden did not hide it from got %v", ids)
		}
	})

	// O caminho dos eventos já é lista de permissão (Views/HiddenFrom só abrem): um leitor de
	// fora da sessão não ganha o evento com a revelação do Escondido, só com a do próprio efeito.
	t.Run("an outsider needs views, hiddenFrom or the effect's own reveal", func(t *testing.T) {
		eff, outsider := uuid.New(), uuid.New()
		turn := newTurn(100, ownersPC, pc, nil)
		rec := event(eff, 102, &turn, false, everyone, []uuid.UUID{reader}, map[uuid.UUID]uuid.UUID{third: hid1})
		latest := map[uuid.UUID]statuseffect.Audience{eff: everyone}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(200)}}
		ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, outsider)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); len(ids) != 0 {
			t.Fatalf("outsider with the Hidden revealed got %v, want none", ids)
		}
		rv = &fakeRevealReader{at: map[uuid.UUID]time.Time{eff: at(200)}}
		ro = get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, outsider)
		if ids := statusIDs(ro, turnByID(ro, turn.id)); !slices.Equal(ids, []uuid.UUID{rec.Event.ID}) {
			t.Fatalf("outsider with the effect revealed got %v, want the event", ids)
		}
	})

	t.Run("a hidden actor's turn comes back, with its anchored events", func(t *testing.T) {
		eff := uuid.New()
		turn := newTurn(100, ownersPC, pc, match.HiddenFrom{ownersPC: {Effect: hid1}})
		rec := event(eff, 102, &turn, true, everyone, []uuid.UUID{reader}, nil)
		latest := map[uuid.UUID]statuseffect.Audience{eff: everyone}
		ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, &fakeRevealReader{}, reader)
		if turnByID(ro, turn.id) != nil || len(statusIDs(ro, nil)) != 0 {
			t.Fatal("before: the hidden turn or its anchored event reached the reader")
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}
		ro = get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, reader)
		tu := turnByID(ro, turn.id)
		if tu == nil {
			t.Fatal("after: the turn is still hidden")
		}
		if ids := statusIDs(ro, tu); !slices.Equal(ids, []uuid.UUID{rec.Event.ID}) {
			t.Fatalf("after: the anchored event = %v", ids)
		}
	})

	t.Run("a hidden target's id and result come back", func(t *testing.T) {
		turn := newTurn(100, ownersPC, npc, match.HiddenFrom{npc: {Effect: hid1}})
		ro := get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, reader)
		tu := turnByID(ro, turn.id)
		if len(tu.Action.TargetID) != 0 || len(tu.Resolution.CharacterResults) != 0 {
			t.Fatalf("before: %v / %d", tu.Action.TargetID, len(tu.Resolution.CharacterResults))
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}
		ro = get(t, []turnSpec{turn}, nil, nil, rv, reader)
		tu = turnByID(ro, turn.id)
		if !slices.Equal(tu.Action.TargetID, []uuid.UUID{npc}) || len(tu.Resolution.CharacterResults) != 1 {
			t.Fatalf("after: %v / %d, want both back", tu.Action.TargetID, len(tu.Resolution.CharacterResults))
		}
	})

	// D2: o terceiro jogador não sabe quem foi atingido, só quantos; revelado, volta o resultado real.
	t.Run("a hidden target that was hit counts anonymously until revealed", func(t *testing.T) {
		turn := newTurn(100, ownersPC, npc, match.HiddenFrom{npc: {Effect: hid1}})
		ro := get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, reader)
		tu := turnByID(ro, turn.id)
		if tu.Resolution.HiddenTargetsHit != 1 || len(tu.Resolution.CharacterResults) != 0 {
			t.Fatalf("before: hit/results = %d/%d, want 1/0", tu.Resolution.HiddenTargetsHit, len(tu.Resolution.CharacterResults))
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}
		ro = get(t, []turnSpec{turn}, nil, nil, rv, reader)
		tu = turnByID(ro, turn.id)
		if tu.Resolution.HiddenTargetsHit != 0 || len(tu.Resolution.CharacterResults) != 1 {
			t.Fatalf("after: hit/results = %d/%d, want 0/1", tu.Resolution.HiddenTargetsHit, len(tu.Resolution.CharacterResults))
		}
		ro = get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, masterUUID)
		if tu = turnByID(ro, turn.id); tu.Resolution.HiddenTargetsHit != 0 || len(tu.Resolution.CharacterResults) != 1 {
			t.Fatalf("master: hit/results = %d/%d, want 0/1", tu.Resolution.HiddenTargetsHit, len(tu.Resolution.CharacterResults))
		}
	})

	// A margem entregaria a esquiva do Escondido (total - margem), até a de quem esquivou: sai
	// para quem não o percebe, e volta com a revelação. Um alvo visível único mantém a sua.
	t.Run("a single hidden target that dodged leaves no margin for who does not perceive it", func(t *testing.T) {
		turn := newTurn(100, ownersPC, npc, match.HiddenFrom{npc: {Effect: hid1}})
		turn.avoided = true
		ro := get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, reader)
		tu := turnByID(ro, turn.id)
		if tu.Resolution.HiddenTargetsHit != 0 || len(tu.Resolution.CharacterResults) != 0 || tu.Resolution.ActionResult.Margin != nil {
			t.Fatalf("reader: hit/results/margin = %d/%d/%v, want 0/0/nil",
				tu.Resolution.HiddenTargetsHit, len(tu.Resolution.CharacterResults), tu.Resolution.ActionResult.Margin)
		}
		ro = get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, masterUUID)
		if m := turnByID(ro, turn.id).Resolution.ActionResult.Margin; m == nil || *m != margin {
			t.Fatalf("master: margin = %v, want %d", m, margin)
		}
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}
		ro = get(t, []turnSpec{turn}, nil, nil, rv, reader)
		if m := turnByID(ro, turn.id).Resolution.ActionResult.Margin; m == nil || *m != margin {
			t.Fatalf("after the reveal: margin = %v, want %d", m, margin)
		}
	})

	t.Run("a single visible target keeps its margin", func(t *testing.T) {
		turn := newTurn(100, ownersPC, pc, nil)
		ro := get(t, []turnSpec{turn}, nil, nil, &fakeRevealReader{}, reader)
		if m := turnByID(ro, turn.id).Resolution.ActionResult.Margin; m == nil || *m != margin {
			t.Fatalf("reader: margin = %v, want %d", m, margin)
		}
	})

	t.Run("a reveal before a later turn does not cover it", func(t *testing.T) {
		hf := match.HiddenFrom{ownersPC: {Effect: hid1}}
		first, second := newTurn(100, ownersPC, pc, hf), newTurn(200, ownersPC, pc, hf)
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}
		ro := get(t, []turnSpec{first, second}, nil, nil, rv, reader)
		if turnByID(ro, first.id) == nil {
			t.Fatal("the turn before the reveal is still hidden")
		}
		if turnByID(ro, second.id) != nil {
			t.Fatal("the turn after the reveal was covered by it")
		}
	})

	t.Run("the master's concealedBy shrinks and revealed flips", func(t *testing.T) {
		turn := newTurn(100, ownersPC, npc, match.HiddenFrom{
			ownersPC: {Effect: hid1}, npc: {Effect: hid2},
		})
		eff := uuid.New()
		rec := event(eff, 102, &turn, false, everyone, []uuid.UUID{reader}, map[uuid.UUID]uuid.UUID{reader: hid1})
		latest := map[uuid.UUID]statuseffect.Audience{eff: everyone}
		want := func(rv *fakeRevealReader, concealed []uuid.UUID, revealed bool) {
			t.Helper()
			ro := get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, latest, rv, masterUUID)
			tu := turnByID(ro, turn.id)
			got := slices.Clone(tu.ConcealedBy)
			slices.SortFunc(got, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			slices.SortFunc(concealed, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			if !slices.Equal(got, concealed) {
				t.Fatalf("concealedBy = %v, want %v", got, concealed)
			}
			if r := tu.StatusEffects[0].Revealed; r == nil || *r != revealed {
				t.Fatalf("revealed = %v, want %v", r, revealed)
			}
		}
		want(&fakeRevealReader{}, []uuid.UUID{hid1, hid2}, false)
		// revelar só o Escondido da HiddenFrom não marca o evento: revealed é do PRÓPRIO efeito
		want(&fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150)}}, []uuid.UUID{hid2}, false)
		want(&fakeRevealReader{at: map[uuid.UUID]time.Time{hid1: at(150), hid2: at(150)}}, nil, false)
		want(&fakeRevealReader{at: map[uuid.UUID]time.Time{eff: at(150)}}, []uuid.UUID{hid1, hid2}, true)
		// uma revelação anterior ao turno não o cobre: segue na lista
		want(&fakeRevealReader{at: map[uuid.UUID]time.Time{hid2: at(50), eff: at(50)}}, []uuid.UUID{hid1, hid2}, false)
	})

	t.Run("the reveals are read once per request, for every effect the history needs", func(t *testing.T) {
		eff := uuid.New()
		turn := newTurn(100, ownersPC, npc, match.HiddenFrom{ownersPC: {Effect: hid1}, npc: {Effect: hid2}})
		rec := event(eff, 102, &turn, false, everyone, nil, map[uuid.UUID]uuid.UUID{reader: hid1})
		rv := &fakeRevealReader{}
		get(t, []turnSpec{turn}, []statuseffect.EventRecord{rec}, nil, rv, reader)
		if len(rv.calls) != 1 {
			t.Fatalf("RevealedAt calls = %d, want 1", len(rv.calls))
		}
		got := slices.Clone(rv.calls[0])
		want := []uuid.UUID{eff, hid1, hid2}
		cmp := func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) }
		slices.SortFunc(got, cmp)
		slices.SortFunc(want, cmp)
		if !slices.Equal(got, want) {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	})

	t.Run("the nil sentinel is never revealed", func(t *testing.T) {
		turn := newTurn(100, ownersPC, pc, match.HiddenFrom{ownersPC: {}})
		rv := &fakeRevealReader{at: map[uuid.UUID]time.Time{uuid.Nil: at(500)}}
		ro := get(t, []turnSpec{turn}, nil, nil, rv, reader)
		if turnByID(ro, turn.id) != nil {
			t.Fatal("a reveal of uuid.Nil uncovered the fail-closed turn")
		}
	})
}
