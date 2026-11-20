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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// A action cancelada de um ator Escondido (revisão final do PR 2, achado 2): quem não o
// percebia no instante do cancelamento não recebe a entrada (nem quem cancelou, nem o ator)
// até uma revelação cobrir o instante. O mestre e o dono sempre recebem.
func TestGetMatchHistoryHiddenCancelledAction(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	ownerUUID, perceiver, bystander, outsider := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	sceneUUID, roundUUID := uuid.New(), uuid.New()
	npc, pc := uuid.New(), uuid.New()
	hidNPC, hidPC := uuid.New(), uuid.New()
	at := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

	cancelled := func(actor, hid uuid.UUID, by ...uuid.UUID) cancelledaction.Record {
		act := action.NewAction(actor, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, &action.Attack{}, nil, nil, nil, nil)
		return cancelledaction.Record{
			UUID: act.GetID(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
			ActorUUID: actor, CancelledBy: masterUUID, Action: *act, CancelledAt: at,
			HiddenFrom: statuseffect.HiddenFrom{actor: {Effect: hid, PerceivedBy: by}},
		}
	}
	npcRec := cancelled(npc, hidNPC, perceiver)
	pcRec := cancelled(pc, hidPC) // lista explícita players:[] — só o mestre e o dono

	get := func(t *testing.T, reveals map[uuid.UUID]time.Time, who uuid.UUID) map[uuid.UUID]bool {
		t.Helper()
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return []*matchEntity.Participant{{Sheet: csEntity.Summary{UUID: pc, PlayerUUID: &ownerUUID}}}, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return []match.HistoryScene{{UUID: sceneUUID, Rounds: []match.HistoryRound{{UUID: roundUUID}}}}, nil
			}},
			&mockParticipationChecker{}, nil, nil,
			&fakeCancelledActionLister{recs: []cancelledaction.Record{npcRec, pcRec}},
		).WithStatusEffectReveals(&fakeRevealReader{at: reveals})
		res, err := uc.Get(context.Background(), matchUUID, who)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		out := map[uuid.UUID]bool{}
		for _, ev := range res.Scenes[0].Rounds[0].Events {
			if ev.Kind == match.HistoryEventActionCancelled {
				out[ev.CancelledAction.UUID] = true
			}
		}
		return out
	}

	for _, c := range []struct {
		name         string
		who          uuid.UUID
		npc, pcEntry bool
		revealed     map[uuid.UUID]time.Time
	}{
		{"the master gets both", masterUUID, true, true, nil},
		{"the owner gets their hidden PC's", ownerUUID, false, true, nil},
		{"whoever perceived the NPC gets it", perceiver, true, false, nil},
		{"a bystander gets neither", bystander, false, false, nil},
		{"an outsider gets neither", outsider, false, false, nil},
		{"a reveal at the cancel instant shows it to everyone", outsider, true, true,
			map[uuid.UUID]time.Time{hidNPC: at, hidPC: at}},
		{"a reveal before the cancel instant does not", bystander, false, false,
			map[uuid.UUID]time.Time{hidNPC: at.Add(-time.Second), hidPC: at.Add(-time.Second)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := get(t, c.revealed, c.who)
			if got[npcRec.UUID] != c.npc || got[pcRec.UUID] != c.pcEntry {
				t.Fatalf("entries: npc %v pc %v, want npc %v pc %v", got[npcRec.UUID], got[pcRec.UUID], c.npc, c.pcEntry)
			}
		})
	}
}
