package match_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// O registro do Escondido é uma lista de permissão (revisão final do PR 2, achado 1): quem não
// era jogador da sessão no instante — um leitor de fora numa partida pública, um membro da
// campanha não inscrito, um jogador inscrito depois do turno — não está no registro e fica
// escondido até a revelação. Quem percebia vê; o dono e o mestre sempre veem.
func TestGetMatchHistoryHiddenOutsiders(t *testing.T) {
	masterUUID, matchUUID, campaignUUID := uuid.New(), uuid.New(), uuid.New()
	perceiver, bystander := uuid.New(), uuid.New()
	outsider, campaignMember, lateEnrollee := uuid.New(), uuid.New(), uuid.New()
	sceneUUID, roundUUID, turnUUID := uuid.New(), uuid.New(), uuid.New()
	npc, perceiverPC, lateSheet := uuid.New(), uuid.New(), uuid.New()
	hid := uuid.New()
	opened := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)

	act := action.NewAction(npc, []uuid.UUID{perceiverPC}, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	hiddenTurn := match.HistoryTurn{
		UUID: turnUUID, CreatedAt: opened, FinishedAt: opened.Add(5 * time.Second), Action: *act,
		// O NPC Escondido com a lista {perceiver}: o bystander era da sessão e não percebia; os
		// três leitores de fora nem eram da sessão.
		ActionHiddenFrom: match.HiddenFrom{npc: {Effect: hid, PerceivedBy: []uuid.UUID{perceiver}}},
	}

	get := func(t *testing.T, public bool, reveals map[uuid.UUID]time.Time, who uuid.UUID) *match.HistoryTurn {
		t.Helper()
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID, IsPublic: public}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return []*matchEntity.Participant{
						{Sheet: csEntity.Summary{UUID: perceiverPC, PlayerUUID: &perceiver}},
						{Sheet: csEntity.Summary{UUID: lateSheet, PlayerUUID: &lateEnrollee}},
					}, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return []match.HistoryScene{{UUID: sceneUUID, Rounds: []match.HistoryRound{{
					UUID: roundUUID, Turns: []match.HistoryTurn{hiddenTurn},
				}}}}, nil
			}},
			&mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }},
			nil, nil, nil,
		).WithStatusEffectReveals(&fakeRevealReader{at: reveals})
		res, err := uc.Get(context.Background(), matchUUID, who)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		for i, tu := range res.Scenes[0].Rounds[0].Turns {
			if tu.UUID == turnUUID {
				return &res.Scenes[0].Rounds[0].Turns[i]
			}
		}
		return nil
	}

	readers := []struct {
		name   string
		who    uuid.UUID
		public bool
	}{
		{"an outsider of a public match", outsider, true},
		{"a campaign member not enrolled", campaignMember, false},
		{"a player enrolled after the turn", lateEnrollee, false},
		{"a session player not in the listing", bystander, false},
	}
	for _, r := range readers {
		t.Run(r.name+" is concealed until the reveal", func(t *testing.T) {
			if tu := get(t, r.public, nil, r.who); tu != nil {
				t.Fatal("the hidden actor's turn is in the history")
			}
			if tu := get(t, r.public, map[uuid.UUID]time.Time{hid: opened}, r.who); tu == nil {
				t.Fatal("the revealed turn is missing")
			}
		})
	}

	t.Run("a session player in the listing sees it", func(t *testing.T) {
		if tu := get(t, false, nil, perceiver); tu == nil {
			t.Fatal("the perceiver lost the turn")
		}
	})

	t.Run("the master always sees it and gets the effect to reveal", func(t *testing.T) {
		tu := get(t, false, nil, masterUUID)
		if tu == nil || len(tu.ConcealedBy) != 1 || tu.ConcealedBy[0] != hid {
			t.Fatalf("master: turn = %v", tu)
		}
	})
}
