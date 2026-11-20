package match_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

type fakeStatusEventLister struct {
	recs        []statuseffect.EventRecord
	latest      map[uuid.UUID]statuseffect.Audience
	latestCalls [][]uuid.UUID
}

func (f *fakeStatusEventLister) ListEventsByMatch(_ context.Context, _ uuid.UUID) ([]statuseffect.EventRecord, error) {
	return f.recs, nil
}

func (f *fakeStatusEventLister) LatestListings(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]statuseffect.Audience, error) {
	f.latestCalls = append(f.latestCalls, append([]uuid.UUID(nil), ids...))
	out := map[uuid.UUID]statuseffect.Audience{}
	for _, id := range ids {
		if a, ok := f.latest[id]; ok {
			out[id] = a
		}
	}
	return out, nil
}

func TestGetMatchHistoryStatusEffects(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	inSight, outOfSight := uuid.New(), uuid.New()
	sceneUUID, roundUUID, turnUUID := uuid.New(), uuid.New(), uuid.New()
	pc := uuid.New()
	base := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }
	everyone := statuseffect.Audience{Everyone: true}
	masterOnly := statuseffect.Audience{Players: []uuid.UUID{}}

	rec := func(effectID uuid.UUID, change statuseffect.Change, sec int, listing statuseffect.Audience,
		views []uuid.UUID, turn *uuid.UUID) statuseffect.EventRecord {
		vs := map[uuid.UUID]masteraction.View{}
		for _, v := range views {
			vs[v] = masteraction.ViewFull
		}
		return statuseffect.EventRecord{
			Event: statuseffect.Event{
				ID: uuid.New(), Change: change, MatchID: matchUUID, SceneID: sceneUUID, RoundID: roundUUID,
				TurnID: turn, Listing: listing, Views: vs, HappenedAt: at(sec),
			},
			Effect: statuseffect.Effect{ID: effectID, CharacterID: pc, Kind: statuseffect.KindPoisoned,
				Params: statuseffect.Params{HPPerRound: 1}, Audience: &listing, AppliedAt: at(sec)},
		}
	}

	tree := func() []match.HistoryScene {
		return []match.HistoryScene{{
			UUID: sceneUUID, Category: "battle",
			Rounds: []match.HistoryRound{{
				UUID: roundUUID, Mode: "Race",
				Turns: []match.HistoryTurn{
					{UUID: turnUUID, FinishedAt: at(100), Action: actionWithFeint(uuid.New()), Reactions: []action.Action{}},
				},
			}},
		}}
	}
	newUC := func(l match.IStatusEffectEventLister) *match.GetMatchHistoryUC {
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return tree(), nil
			}},
			&mockParticipationChecker{}, nil, nil, nil,
		)
		if l != nil {
			uc = uc.WithStatusEffectEvents(l)
		}
		return uc
	}
	get := func(t *testing.T, uc *match.GetMatchHistoryUC, reader uuid.UUID) match.HistoryRound {
		t.Helper()
		res, err := uc.Get(context.Background(), matchUUID, reader)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		return res.Scenes[0].Rounds[0]
	}
	roundStatus := func(ro match.HistoryRound) []match.HistoryStatusEffect {
		var out []match.HistoryStatusEffect
		for _, ev := range ro.Events {
			if ev.Kind == match.HistoryEventStatusEffect {
				if ev.StatusEffect == nil {
					panic("statusEffect event without its payload")
				}
				if !ev.At.Equal(ev.StatusEffect.HappenedAt) {
					panic("event At differs from happenedAt")
				}
				out = append(out, *ev.StatusEffect)
			}
		}
		return out
	}
	idsOf := func(list []match.HistoryStatusEffect) []uuid.UUID {
		out := []uuid.UUID{}
		for _, s := range list {
			out = append(out, s.UUID)
		}
		return out
	}
	assertIDs := func(t *testing.T, who string, got []match.HistoryStatusEffect, want ...uuid.UUID) {
		t.Helper()
		if want == nil {
			want = []uuid.UUID{}
		}
		if g := idsOf(got); !slices.Equal(g, want) {
			t.Fatalf("%s: events = %v, want %v", who, g, want)
		}
	}

	t.Run("master sees every event with its visibility", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, masterOnly, nil, nil)
		uc := newUC(&fakeStatusEventLister{recs: []statuseffect.EventRecord{a}})
		got := roundStatus(get(t, uc, masterUUID))
		assertIDs(t, "master", got, a.Event.ID)
		if got[0].Effect.Audience == nil || got[0].Public == nil || *got[0].Public {
			t.Fatalf("master: audience/public = %v/%v, want the list and public=false", got[0].Effect.Audience, got[0].Public)
		}
		if got[0].Change != statuseffect.ChangeApplied || got[0].Effect.ID != eff || got[0].Effect.CharacterID != pc {
			t.Fatalf("master: %+v", got[0])
		}
	})

	t.Run("a listed reader in sight sees it without visibility", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, everyone, []uuid.UUID{inSight}, nil)
		uc := newUC(&fakeStatusEventLister{recs: []statuseffect.EventRecord{a}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}})
		got := roundStatus(get(t, uc, inSight))
		assertIDs(t, "in sight", got, a.Event.ID)
		if got[0].Effect.Audience != nil || got[0].Public != nil {
			t.Fatalf("player got the visibility: %+v/%v", got[0].Effect.Audience, got[0].Public)
		}
		if got[0].Effect.Params.HPPerRound != 1 {
			t.Fatalf("snapshot lost: %+v", got[0].Effect)
		}
	})

	t.Run("hidden then made public later shows the past to who was in sight", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, masterOnly, []uuid.UUID{inSight}, nil)
		e := rec(eff, statuseffect.ChangeEdited, 20, everyone, []uuid.UUID{inSight, outOfSight}, nil)
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a, e}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		assertIDs(t, "in sight", roundStatus(get(t, newUC(l), inSight)), a.Event.ID, e.Event.ID)
	})

	t.Run("never to who was not in sight", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, masterOnly, []uuid.UUID{inSight}, nil)
		e := rec(eff, statuseffect.ChangeEdited, 20, everyone, []uuid.UUID{inSight, outOfSight}, nil)
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a, e}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		assertIDs(t, "out of sight", roundStatus(get(t, newUC(l), outOfSight)), e.Event.ID)
	})

	t.Run("removed effect keeps its last audience", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, everyone, []uuid.UUID{inSight}, nil)
		e := rec(eff, statuseffect.ChangeEdited, 20, masterOnly, []uuid.UUID{inSight}, nil)
		r := rec(eff, statuseffect.ChangeRemoved, 30, masterOnly, []uuid.UUID{inSight}, nil)
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a, e, r}, latest: map[uuid.UUID]statuseffect.Audience{eff: masterOnly}}
		assertIDs(t, "player", roundStatus(get(t, newUC(l), inSight)))
		assertIDs(t, "master", roundStatus(get(t, newUC(l), masterUUID)), a.Event.ID, e.Event.ID, r.Event.ID)
	})

	t.Run("latest from another match counts", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, masterOnly, []uuid.UUID{inSight}, nil)
		// O último evento do efeito está em outra partida: só o LatestListings o conhece.
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		assertIDs(t, "player", roundStatus(get(t, newUC(l), inSight)), a.Event.ID)
	})

	t.Run("no latest listing fails closed", func(t *testing.T) {
		a := rec(uuid.New(), statuseffect.ChangeApplied, 10, everyone, []uuid.UUID{inSight}, nil)
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a}}
		assertIDs(t, "player", roundStatus(get(t, newUC(l), inSight)))
	})

	t.Run("hidden from the reader and never revealed stays hidden", func(t *testing.T) {
		eff := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 10, everyone, nil, nil)
		a.Event.HiddenFrom = map[uuid.UUID]uuid.UUID{outOfSight: uuid.New()}
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		assertIDs(t, "hidden-from reader", roundStatus(get(t, newUC(l), outOfSight)))
	})

	t.Run("in-turn event goes inside its turn", func(t *testing.T) {
		eff := uuid.New()
		tu := turnUUID
		anchor := uuid.New()
		late := rec(eff, statuseffect.ChangeEdited, 50, everyone, []uuid.UUID{inSight}, &tu)
		early := rec(eff, statuseffect.ChangeApplied, 40, everyone, []uuid.UUID{inSight}, &tu)
		early.Event.ActionID = &anchor
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{late, early}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		ro := get(t, newUC(l), inSight)
		assertIDs(t, "round", roundStatus(ro))
		got := ro.Turns[0].StatusEffects
		assertIDs(t, "turn", got, early.Event.ID, late.Event.ID)
		if got[0].TurnUUID == nil || *got[0].TurnUUID != turnUUID || got[0].ActionID == nil || *got[0].ActionID != anchor {
			t.Fatalf("turn/action = %v/%v", got[0].TurnUUID, got[0].ActionID)
		}
		if got[1].ActionID != nil {
			t.Fatalf("unanchored event got an action: %v", got[1].ActionID)
		}
	})

	t.Run("turn not in the tree falls to the round's events", func(t *testing.T) {
		eff := uuid.New()
		lost := uuid.New()
		a := rec(eff, statuseffect.ChangeApplied, 40, everyone, []uuid.UUID{inSight}, &lost)
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a}, latest: map[uuid.UUID]statuseffect.Audience{eff: everyone}}
		ro := get(t, newUC(l), inSight)
		assertIDs(t, "round", roundStatus(ro), a.Event.ID)
		assertIDs(t, "turn", ro.Turns[0].StatusEffects)
	})

	t.Run("round not in the tree is dropped", func(t *testing.T) {
		a := rec(uuid.New(), statuseffect.ChangeApplied, 40, everyone, nil, nil)
		a.Event.RoundID = uuid.New()
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{a}}
		assertIDs(t, "master", roundStatus(get(t, newUC(l), masterUUID)))
	})

	t.Run("latest listings are read once with each effect once", func(t *testing.T) {
		e1, e2 := uuid.New(), uuid.New()
		l := &fakeStatusEventLister{recs: []statuseffect.EventRecord{
			rec(e1, statuseffect.ChangeApplied, 10, everyone, nil, nil),
			rec(e2, statuseffect.ChangeApplied, 11, everyone, nil, nil),
			rec(e1, statuseffect.ChangeEdited, 12, everyone, nil, nil),
		}}
		get(t, newUC(l), inSight)
		if len(l.latestCalls) != 1 || len(l.latestCalls[0]) != 2 ||
			!slices.Contains(l.latestCalls[0], e1) || !slices.Contains(l.latestCalls[0], e2) {
			t.Fatalf("LatestListings calls = %v, want one with [e1 e2]", l.latestCalls)
		}
	})

	t.Run("no lister, nothing stitched", func(t *testing.T) {
		ro := get(t, newUC(nil), masterUUID)
		assertIDs(t, "round", roundStatus(ro))
		if ro.Turns[0].StatusEffects == nil {
			t.Fatal("turn statusEffects is nil, want empty")
		}
	})
}
