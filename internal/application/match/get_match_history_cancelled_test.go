package match_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

type fakeCancelledActionLister struct{ recs []cancelledaction.Record }

func (f *fakeCancelledActionLister) ListByMatch(_ context.Context, _ uuid.UUID) ([]cancelledaction.Record, error) {
	return f.recs, nil
}

// TestGetMatchHistoryStitchesCancelledActions is spec 2026-10-10 §8.1 P8: every cancelled action
// is an actionCancelled event of its round, in time order with the round's other events — even
// one cancelled while a turn the tree holds was open: it never belonged to that turn — and each
// reader gets their cut of it: the master the action at Full, the actor's owner at Declaration
// (the cut of their own ownQueue), anyone else who cancelled and nothing more.
func TestGetMatchHistoryStitchesCancelledActions(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	ownerUUID, otherUUID := uuid.New(), uuid.New()
	gonSheet, killuaSheet := uuid.New(), uuid.New()
	sceneUUID, roundUUID, turnUUID := uuid.New(), uuid.New(), uuid.New()
	base := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }

	rec := func(turn *uuid.UUID, cancelled time.Time) cancelledaction.Record {
		a := actionWithFeint(gonSheet)
		a.Speed = action.ActionSpeed{Bar: 1, RollCheck: action.RollCheck{SkillName: "Legerity", Result: 9}}
		return cancelledaction.Record{
			UUID: a.GetID(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
			TurnUUID: turn, ActorUUID: gonSheet, CancelledBy: ownerUUID, Action: a,
			CancelledAt: cancelled,
		}
	}
	beforeTurn := rec(nil, at(20))
	duringTurn := rec(&turnUUID, at(35)) // a turn the tree holds: still the round's event
	stray := rec(nil, at(36))
	stray.RoundUUID = uuid.New() // a round the tree does not hold: logged and dropped

	modeChange := matchevent.Event{
		UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
		Kind: matchevent.KindRoundModeChanged, Payload: json.RawMessage(`{"from":"Free","to":"Race"}`),
		CreatedAt: at(30),
	}

	newUC := func() *match.GetMatchHistoryUC {
		return match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return []*matchEntity.Participant{
						{Sheet: csEntity.Summary{UUID: gonSheet, PlayerUUID: &ownerUUID}},
						{Sheet: csEntity.Summary{UUID: killuaSheet, PlayerUUID: &otherUUID}},
					}, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return []match.HistoryScene{{
					UUID: sceneUUID, Category: "battle",
					Rounds: []match.HistoryRound{{
						UUID: roundUUID, Mode: "Race",
						Turns: []match.HistoryTurn{{
							UUID: turnUUID, FinishedAt: at(40), Action: actionWithFeint(killuaSheet),
							Reactions: []action.Action{},
						}},
					}},
				}}, nil
			}},
			&mockParticipationChecker{},
			&fakeEventLister{events: []matchevent.Event{modeChange}},
			nil,
			// Deliberately out of time order: the use case orders, it does not trust the feed.
			&fakeCancelledActionLister{recs: []cancelledaction.Record{duringTurn, stray, beforeTurn}},
		)
	}

	tests := []struct {
		name       string
		reader     uuid.UUID
		wantCut    match.CancelledActionCut
		wantAction bool
	}{
		{name: "the master reads the whole action", reader: masterUUID, wantCut: match.CancelledActionCutFull, wantAction: true},
		{name: "the actor's owner reads what they declared", reader: ownerUUID, wantCut: match.CancelledActionCutDeclaration, wantAction: true},
		{name: "another player reads only who cancelled", reader: otherUUID, wantCut: match.CancelledActionCutActor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newUC().Get(context.Background(), matchUUID, tt.reader)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			round := res.Scenes[0].Rounds[0]
			if ma := round.Turns[0].MasterActions; len(ma) != 0 {
				t.Fatalf("the turn holds %d master actions — a cancelled action is not one", len(ma))
			}

			wantOrder := []struct {
				kind match.HistoryEventKind
				rec  *cancelledaction.Record
			}{
				{match.HistoryEventActionCancelled, &beforeTurn},
				{match.HistoryEventRoundModeChanged, nil},
				{match.HistoryEventActionCancelled, &duringTurn},
			}
			if len(round.Events) != len(wantOrder) {
				t.Fatalf("events = %d (%+v), want %d", len(round.Events), round.Events, len(wantOrder))
			}
			for i, want := range wantOrder {
				ev := round.Events[i]
				if ev.Kind != want.kind {
					t.Fatalf("events[%d].kind = %q, want %q", i, ev.Kind, want.kind)
				}
				if want.rec == nil {
					if ev.CancelledAction != nil {
						t.Fatalf("events[%d] (a regime change) carries a cancelled action", i)
					}
					continue
				}
				ca := ev.CancelledAction
				if ca == nil || ev.RoundModeChange != nil || ev.MasterAction != nil {
					t.Fatalf("events[%d] = %+v, want the cancelled action %s alone", i, ev, want.rec.UUID)
				}
				if !ev.At.Equal(want.rec.CancelledAt) {
					t.Fatalf("events[%d].At = %v, want its CancelledAt %v", i, ev.At, want.rec.CancelledAt)
				}
				if ca.UUID != want.rec.UUID || ca.ActorUUID != gonSheet {
					t.Fatalf("events[%d] = uuid %s actor %s, want %s / %s", i, ca.UUID, ca.ActorUUID, want.rec.UUID, gonSheet)
				}
				if ca.Cut != tt.wantCut {
					t.Fatalf("events[%d].Cut = %q, want %q", i, ca.Cut, tt.wantCut)
				}
				if !tt.wantAction {
					if ca.Action != nil {
						t.Fatalf("events[%d] hands this reader the action: %+v", i, ca.Action)
					}
					continue
				}
				if ca.Action == nil || !reflect.DeepEqual(*ca.Action, want.rec.Action) {
					t.Fatalf("events[%d].Action = %+v, want the recorded action", i, ca.Action)
				}
			}
		})
	}
}
