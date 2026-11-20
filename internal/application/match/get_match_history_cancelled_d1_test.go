package match_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// D1 (dono do produto, 2026-11-20): "ações canceladas devem ser mostradas apenas para o dono e
// para o mestre, mas não para outros jogadores." Quem não é nem o mestre nem o dono do ator não
// recebe a entrada — nem o actorId. O Escondido do ator não muda isso: o mestre e o dono o
// enxergam sempre.
type cancelledD1Fixture struct {
	matchUUID, masterUUID, ownerUUID, thirdUUID uuid.UUID
	actor                                       uuid.UUID
	rec                                         cancelledaction.Record
	turns                                       []match.HistoryTurn
}

func newCancelledD1Fixture() cancelledD1Fixture {
	f := cancelledD1Fixture{
		matchUUID: uuid.New(), masterUUID: uuid.New(), ownerUUID: uuid.New(), thirdUUID: uuid.New(),
		actor: uuid.New(),
	}
	sceneUUID, roundUUID := uuid.New(), uuid.New()
	act := action.NewAction(f.actor, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, &action.Attack{}, nil, nil, nil, nil)
	f.rec = cancelledaction.Record{
		UUID: act.GetID(), MatchUUID: f.matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
		ActorUUID: f.actor, CancelledBy: f.ownerUUID, Action: *act,
		CancelledAt: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}
	return f
}

// cancelledEvents devolve os eventos actionCancelled do único round, lidos por who.
func (f cancelledD1Fixture) cancelledEvents(t *testing.T, who uuid.UUID) []match.HistoryEvent {
	t.Helper()
	uc := match.NewGetMatchHistoryUC(
		&testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: f.matchUUID, MasterUUID: f.masterUUID, IsPublic: true}, nil
			},
			ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
				return []*matchEntity.Participant{{Sheet: csEntity.Summary{UUID: f.actor, PlayerUUID: &f.ownerUUID}}}, nil
			},
		},
		&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
			return []match.HistoryScene{{UUID: f.rec.SceneUUID, Rounds: []match.HistoryRound{{UUID: f.rec.RoundUUID, Turns: f.turns}}}}, nil
		}},
		&mockParticipationChecker{}, nil, nil,
		&fakeCancelledActionLister{recs: []cancelledaction.Record{f.rec}},
	)
	res, err := uc.Get(context.Background(), f.matchUUID, who)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var out []match.HistoryEvent
	for _, ev := range res.Scenes[0].Rounds[0].Events {
		if ev.Kind == match.HistoryEventActionCancelled {
			out = append(out, ev)
		}
	}
	return out
}

func TestGetMatchHistory_CancelledAction_ThirdPlayerSeesNothing(t *testing.T) {
	f := newCancelledD1Fixture()
	if got := f.cancelledEvents(t, f.thirdUUID); len(got) != 0 {
		t.Fatalf("a third player got %d actionCancelled event(s), want none: %+v", len(got), got)
	}
}

func TestGetMatchHistory_CancelledAction_OwnerSeesDeclaration(t *testing.T) {
	f := newCancelledD1Fixture()
	got := f.cancelledEvents(t, f.ownerUUID)
	if len(got) != 1 || got[0].CancelledAction.Cut != match.CancelledActionCutDeclaration ||
		got[0].CancelledAction.Action == nil || got[0].CancelledAction.ActorUUID != f.actor {
		t.Fatalf("owner events = %+v, want one at the Declaration cut with the action", got)
	}
}

func TestGetMatchHistory_CancelledAction_MasterSeesFull(t *testing.T) {
	f := newCancelledD1Fixture()
	got := f.cancelledEvents(t, f.masterUUID)
	if len(got) != 1 || got[0].CancelledAction.Cut != match.CancelledActionCutFull ||
		got[0].CancelledAction.Action == nil {
		t.Fatalf("master events = %+v, want one at the Full cut with the action", got)
	}
}

// Master and owner see through Hidden: the actor is Hidden (a turn of theirs carries the
// allow-list), which keeps a third player out of the turn and of the cancelled action alike, and
// leaves the master and the owner reading both. No per-record allow-list is involved.
func TestGetMatchHistory_CancelledAction_HiddenActorOwnerAndMasterStillSee(t *testing.T) {
	f := newCancelledD1Fixture()
	act := action.NewAction(f.actor, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, &action.Attack{}, nil, nil, nil, nil)
	f.turns = []match.HistoryTurn{{
		UUID: uuid.New(), CreatedAt: f.rec.CancelledAt, FinishedAt: f.rec.CancelledAt,
		Action: *act, Reactions: []action.Action{},
		ActionHiddenFrom: match.HiddenFrom{f.actor: {Effect: uuid.New()}},
	}}
	for _, c := range []struct {
		name string
		who  uuid.UUID
		want int
	}{
		{"master", f.masterUUID, 1}, {"owner", f.ownerUUID, 1}, {"third player", f.thirdUUID, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := f.cancelledEvents(t, c.who); len(got) != c.want {
				t.Fatalf("events = %d, want %d", len(got), c.want)
			}
		})
	}
}
