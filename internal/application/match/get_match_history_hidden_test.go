package match_test

import (
	"context"
	"encoding/json"
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

// Histórico de quem estava Escondido (Task 15, R4): um personagem no actions.hidden_from de um
// turno fica escondido do leitor enquanto não for revelado. Ator escondido: o turno some para
// ele (suas master actions e eventos sem âncora vão aos eventos do round; os ancorados somem
// junto). Reator escondido: a reação some. Alvo escondido: sai de targetId e do resultado. O
// mestre e o dono sempre veem; ampliar a lista do Escondido nunca revela o passado.
func TestGetMatchHistoryHiddenActor(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	ownerUUID, reader, other := uuid.New(), uuid.New(), uuid.New()
	sceneUUID, roundUUID, turnUUID := uuid.New(), uuid.New(), uuid.New()
	npc, pc, ownersPC := uuid.New(), uuid.New(), uuid.New()
	hid := uuid.New()
	base := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }
	everyone := statuseffect.Audience{Everyone: true}

	// O turno: actor ataca target; reactor reage. Os ids são fixos para as âncoras.
	actionID, reactionID := uuid.New(), uuid.New()
	turn := func(actor, target, reactor uuid.UUID, actionHF match.HiddenFrom, reactionHF map[uuid.UUID]match.HiddenFrom) match.HistoryTurn {
		act := action.NewAction(actor, []uuid.UUID{target}, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil, action.WithReconstructedID(actionID))
		react := action.NewAction(reactor, nil, actionID, nil, action.ActionSpeed{},
			nil, nil, nil, nil, &action.Dodge{}, nil, nil, action.WithReconstructedID(reactionID))
		react.ReactionKind = action.ReactDodge
		return match.HistoryTurn{
			UUID: turnUUID, CreatedAt: at(100), FinishedAt: at(100),
			Action: *act, Reactions: []action.Action{*react},
			Resolution: &service.TurnResolution{
				IsSettled:        true,
				CharacterResults: []service.CharacterResult{{TargetID: target, ReactionID: reactionID, ReactionKind: "dodge"}},
			},
			ActionHiddenFrom: actionHF, ReactionHiddenFrom: reactionHF,
		}
	}

	type deps struct {
		turn    match.HistoryTurn
		status  []statuseffect.EventRecord
		latest  map[uuid.UUID]statuseffect.Audience
		masters []masteraction.Record
	}
	get := func(t *testing.T, d deps, who uuid.UUID) (match.HistoryRound, *match.HistoryTurn) {
		t.Helper()
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return []*matchEntity.Participant{
						{Sheet: csEntity.Summary{UUID: ownersPC, PlayerUUID: &ownerUUID}},
						{Sheet: csEntity.Summary{UUID: pc, PlayerUUID: &other}},
					}, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return []match.HistoryScene{{UUID: sceneUUID, Rounds: []match.HistoryRound{{
					UUID: roundUUID, Turns: []match.HistoryTurn{d.turn},
				}}}}, nil
			}},
			&mockParticipationChecker{}, nil, &fakeMasterActionLister{recs: d.masters}, nil,
		).WithStatusEffectEvents(&fakeStatusEventLister{recs: d.status, latest: d.latest})
		res, err := uc.Get(context.Background(), matchUUID, who)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		ro := res.Scenes[0].Rounds[0]
		for i := range ro.Turns {
			if ro.Turns[i].UUID == turnUUID {
				return ro, &ro.Turns[i]
			}
		}
		return ro, nil
	}
	statusRec := func(sec int, anchor *uuid.UUID, views ...uuid.UUID) statuseffect.EventRecord {
		vs := map[uuid.UUID]masteraction.View{}
		for _, v := range views {
			vs[v] = masteraction.ViewFull
		}
		tu := turnUUID
		eff := uuid.New()
		return statuseffect.EventRecord{
			Event: statuseffect.Event{
				ID: uuid.New(), Change: statuseffect.ChangeApplied, MatchID: matchUUID, SceneID: sceneUUID,
				RoundID: roundUUID, TurnID: &tu, ActionID: anchor, Listing: everyone, Views: vs, HappenedAt: at(sec),
			},
			Effect: statuseffect.Effect{ID: eff, CharacterID: pc, Kind: statuseffect.KindProne, AppliedAt: at(sec)},
		}
	}
	latestOf := func(recs ...statuseffect.EventRecord) map[uuid.UUID]statuseffect.Audience {
		out := map[uuid.UUID]statuseffect.Audience{}
		for _, r := range recs {
			out[r.Effect.ID] = everyone
		}
		out[hid] = everyone // ampliar a lista do Escondido não revela nada
		return out
	}
	roundStatusIDs := func(ro match.HistoryRound) []uuid.UUID {
		out := []uuid.UUID{}
		for _, ev := range ro.Events {
			if ev.Kind == match.HistoryEventStatusEffect {
				out = append(out, ev.StatusEffect.UUID)
			}
		}
		return out
	}
	roundMasterIDs := func(ro match.HistoryRound) []uuid.UUID {
		out := []uuid.UUID{}
		for _, ev := range ro.Events {
			if ev.Kind == match.HistoryEventMasterAction {
				out = append(out, ev.MasterAction.UUID)
			}
		}
		return out
	}
	turnStatusIDs := func(tu *match.HistoryTurn) []uuid.UUID {
		out := []uuid.UUID{}
		for _, s := range tu.StatusEffects {
			out = append(out, s.UUID)
		}
		return out
	}
	hiddenActor := func() match.HistoryTurn {
		return turn(ownersPC, pc, pc, match.HiddenFrom{ownersPC: {reader: hid}}, nil)
	}

	t.Run("turn hidden from the reader until revealed", func(t *testing.T) {
		d := deps{turn: hiddenActor(), latest: map[uuid.UUID]statuseffect.Audience{hid: everyone}}
		if _, tu := get(t, d, reader); tu != nil {
			t.Fatal("reader: the hidden actor's turn is in the history")
		}
		if _, tu := get(t, d, other); tu == nil {
			t.Fatal("a reader the actor was not hidden from lost the turn")
		}
		_, tu := get(t, d, ownerUUID)
		if tu == nil {
			t.Fatal("the actor's owner lost the turn")
		}
		if tu.ActionHiddenFrom != nil || tu.ReactionHiddenFrom != nil || tu.ConcealedBy != nil {
			t.Fatalf("owner: the record reached the reader: %v / %v / %v", tu.ActionHiddenFrom, tu.ReactionHiddenFrom, tu.ConcealedBy)
		}
		_, tu = get(t, d, masterUUID)
		if tu == nil {
			t.Fatal("the master lost the turn")
		}
		if !slices.Equal(tu.ConcealedBy, []uuid.UUID{hid}) {
			t.Fatalf("master: concealedBy = %v, want [%s]", tu.ConcealedBy, hid)
		}
	})

	t.Run("a hidden turn's master actions and unanchored status events fall to the round's events", func(t *testing.T) {
		tu := turnUUID
		ma := masteraction.Record{
			UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID, MasterUUID: masterUUID,
			TurnUUID: &tu, Kind: masteraction.KindTurnNote, Content: json.RawMessage(`{"note":"x"}`),
			Views: map[uuid.UUID]masteraction.View{reader: masteraction.ViewFull}, HappenedAt: at(101),
		}
		anchor := actionID
		loose, anchored := statusRec(102, nil, reader), statusRec(103, &anchor, reader)
		d := deps{turn: hiddenActor(), status: []statuseffect.EventRecord{loose, anchored},
			latest: latestOf(loose, anchored), masters: []masteraction.Record{ma}}
		ro, got := get(t, d, reader)
		if got != nil {
			t.Fatal("reader: the hidden turn is in the history")
		}
		if ids := roundMasterIDs(ro); !slices.Equal(ids, []uuid.UUID{ma.UUID}) {
			t.Fatalf("round master actions = %v, want the turn's", ids)
		}
		if ids := roundStatusIDs(ro); !slices.Equal(ids, []uuid.UUID{loose.Event.ID}) {
			t.Fatalf("round status events = %v, want only the unanchored one", ids)
		}
		_, got = get(t, d, masterUUID)
		if ids := turnStatusIDs(got); !slices.Equal(ids, []uuid.UUID{loose.Event.ID, anchored.Event.ID}) {
			t.Fatalf("master: turn status events = %v, want both", ids)
		}
	})

	t.Run("a hidden reactor's reaction is dropped", func(t *testing.T) {
		anchor := reactionID
		onReaction, onAction := statusRec(102, &anchor, reader), statusRec(103, nil, reader)
		an := actionID
		onAction.Event.ActionID = &an
		d := deps{
			turn:   turn(ownersPC, pc, npc, nil, map[uuid.UUID]match.HiddenFrom{reactionID: {npc: {reader: hid}}}),
			status: []statuseffect.EventRecord{onReaction, onAction}, latest: latestOf(onReaction, onAction),
		}
		_, tu := get(t, d, reader)
		if tu == nil {
			t.Fatal("reader: a visible actor's turn is missing")
		}
		if len(tu.Reactions) != 0 {
			t.Fatalf("reader: reactions = %d, want the hidden reactor's dropped", len(tu.Reactions))
		}
		if ids := turnStatusIDs(tu); !slices.Equal(ids, []uuid.UUID{onAction.Event.ID}) {
			t.Fatalf("reader: turn status events = %v, want the one on the action only", ids)
		}
		_, tu = get(t, d, other)
		if len(tu.Reactions) != 1 {
			t.Fatalf("other: reactions = %d, want the reaction", len(tu.Reactions))
		}
		_, tu = get(t, d, masterUUID)
		if len(tu.Reactions) != 1 || !slices.Equal(tu.ConcealedBy, []uuid.UUID{hid}) {
			t.Fatalf("master: reactions/concealedBy = %d/%v", len(tu.Reactions), tu.ConcealedBy)
		}
	})

	t.Run("a hidden target leaves targetId and the resolution", func(t *testing.T) {
		d := deps{turn: turn(ownersPC, npc, npc, match.HiddenFrom{npc: {reader: hid}}, nil)}
		_, tu := get(t, d, reader)
		if tu == nil {
			t.Fatal("reader: a visible actor's turn is missing")
		}
		if len(tu.Action.TargetID) != 0 {
			t.Fatalf("reader: targetId = %v, want the hidden target cut", tu.Action.TargetID)
		}
		if len(tu.Resolution.CharacterResults) != 0 {
			t.Fatalf("reader: results = %+v, want the hidden target's cut", tu.Resolution.CharacterResults)
		}
		_, tu = get(t, d, other)
		if len(tu.Action.TargetID) != 1 || len(tu.Resolution.CharacterResults) != 1 {
			t.Fatalf("other: targets/results = %v/%d, want both", tu.Action.TargetID, len(tu.Resolution.CharacterResults))
		}
	})

	t.Run("an anchor naming nothing stored belongs to the turn, unanchored", func(t *testing.T) {
		gone := uuid.New() // uma reação sentada que foi substituída depois
		s := statusRec(102, &gone, reader)
		d := deps{turn: turn(ownersPC, pc, pc, nil, nil), status: []statuseffect.EventRecord{s}, latest: latestOf(s)}
		for _, who := range []uuid.UUID{reader, masterUUID} {
			_, tu := get(t, d, who)
			if len(tu.StatusEffects) != 1 || tu.StatusEffects[0].ActionID != nil {
				t.Fatalf("%s: turn status events = %+v, want the event, no anchor", who, tu.StatusEffects)
			}
		}
	})

	t.Run("an unreadable record hides the turn from everyone but the master and the owner", func(t *testing.T) {
		d := deps{turn: turn(ownersPC, pc, pc, match.HiddenFrom{ownersPC: {match.HiddenFromEveryone: uuid.Nil}}, nil)}
		for _, who := range []uuid.UUID{reader, other} {
			if _, tu := get(t, d, who); tu != nil {
				t.Fatalf("%s: the turn is in the history", who)
			}
		}
		for _, who := range []uuid.UUID{ownerUUID, masterUUID} {
			if _, tu := get(t, d, who); tu == nil {
				t.Fatalf("%s: the turn is missing", who)
			}
		}
		if _, tu := get(t, d, masterUUID); tu.ConcealedBy != nil {
			t.Fatalf("master: concealedBy = %v, want no effect for the sentinel", tu.ConcealedBy)
		}
	})

	t.Run("the master's status events carry revealed false", func(t *testing.T) {
		s := statusRec(102, nil, reader)
		d := deps{turn: turn(ownersPC, pc, pc, nil, nil), status: []statuseffect.EventRecord{s}, latest: latestOf(s)}
		_, tu := get(t, d, masterUUID)
		if r := tu.StatusEffects[0].Revealed; r == nil || *r {
			t.Fatalf("master: revealed = %v, want false", r)
		}
		_, tu = get(t, d, reader)
		if tu.StatusEffects[0].Revealed != nil {
			t.Fatal("reader: revealed reached a player")
		}
	})

	// O dono está em Views (o dono sempre entra), mas a lista explícita players:[] o deixa de
	// fora: o evento não é dele.
	t.Run("an owner in views with an explicit empty listing does not see the event", func(t *testing.T) {
		s := statusRec(102, nil, ownerUUID)
		s.Effect.CharacterID = ownersPC
		s.Event.Listing = statuseffect.Audience{Players: []uuid.UUID{}}
		d := deps{turn: turn(ownersPC, pc, pc, nil, nil), status: []statuseffect.EventRecord{s},
			latest: map[uuid.UUID]statuseffect.Audience{s.Effect.ID: {Players: []uuid.UUID{}}}}
		if _, tu := get(t, d, ownerUUID); len(tu.StatusEffects) != 0 {
			t.Fatalf("owner: turn status events = %v, want none", turnStatusIDs(tu))
		}
	})
}
