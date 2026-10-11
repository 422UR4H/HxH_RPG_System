package match_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type cancelledActionsFeed []cancelledaction.Record

func (f cancelledActionsFeed) ListByMatch(context.Context, uuid.UUID) ([]cancelledaction.Record, error) {
	return f, nil
}

// historyTree is the round repository's read side only: the embedded nil interface panics on
// anything GET /history must never call.
type historyTree struct {
	match.IRoundRepository
	scenes []match.HistoryScene
}

func (h historyTree) FindMatchHistory(context.Context, uuid.UUID) ([]match.HistoryScene, error) {
	return h.scenes, nil
}

type anyoneParticipates struct{}

func (anyoneParticipates) ExistsSheetInCampaign(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

// TestGetMatchHistoryProjectsCancelledActionsPerReader is spec 2026-10-10 §8.1 P8 end to end
// through the REST handler and the real use case: a cancelled action is a round event
// {kind: "actionCancelled", actorId, action?} — the master reads the action at the Full cut
// (speed numbers kept), the actor's owner at the Declaration cut (speed reduced to its skill
// name, the cut of their own ownQueue), and anyone else reads actorId with no action key at all.
func TestGetMatchHistoryProjectsCancelledActionsPerReader(t *testing.T) {
	masterUUID, ownerUUID, otherUUID := uuid.New(), uuid.New(), uuid.New()
	matchUUID, roundUUID, sheetUUID := uuid.New(), uuid.New(), uuid.New()
	at := time.Date(2026, 10, 10, 10, 0, 7, 0, time.UTC)

	a := action.NewAction(sheetUUID, []uuid.UUID{uuid.New()}, uuid.Nil, nil,
		action.ActionSpeed{Bar: 1, RollCheck: action.RollCheck{SkillName: "Legerity", SkillValue: 4, Result: 13}},
		nil, nil, &action.Attack{Hit: action.RollCheck{SkillName: "Accuracy", Result: 12}},
		nil, nil, nil, nil)
	rec := cancelledaction.Record{
		UUID: a.GetID(), MatchUUID: matchUUID, RoundUUID: roundUUID, ActorUUID: sheetUUID,
		CancelledBy: ownerUUID, Action: *a, CancelledAt: at,
	}

	uc := match.NewGetMatchHistoryUC(
		&testutil.MockMatchRepo{
			GetMatchFn: func(context.Context, uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID}, nil
			},
			ListParticipantsByMatchUUIDFn: func(context.Context, uuid.UUID) ([]*matchEntity.Participant, error) {
				return []*matchEntity.Participant{
					{Sheet: csEntity.Summary{UUID: sheetUUID, PlayerUUID: &ownerUUID}},
					{Sheet: csEntity.Summary{UUID: uuid.New(), PlayerUUID: &otherUUID}},
				}, nil
			},
		},
		historyTree{scenes: []match.HistoryScene{{
			UUID: uuid.New(), Category: "battle", CreatedAt: at,
			Rounds: []match.HistoryRound{{UUID: roundUUID, Mode: "Race", CreatedAt: at}},
		}}},
		anyoneParticipates{}, nil, nil, cancelledActionsFeed{rec},
	)
	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{uuid}/history"},
		apiMatch.GetMatchHistoryHandler(uc))

	events := func(t *testing.T, reader uuid.UUID) []map[string]json.RawMessage {
		t.Helper()
		ctx := context.WithValue(context.Background(), auth.UserIDKey, reader)
		resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/history")
		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d. Body: %s", resp.Code, resp.Body.String())
		}
		var got struct {
			Scenes []struct {
				Rounds []struct {
					Events []map[string]json.RawMessage `json:"events"`
				} `json:"rounds"`
			} `json:"scenes"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v — %s", err, resp.Body.String())
		}
		evs := got.Scenes[0].Rounds[0].Events
		if len(evs) != 1 {
			t.Fatalf("events = %d, want 1: %s", len(evs), resp.Body.String())
		}
		ev := evs[0]
		if string(ev["kind"]) != `"actionCancelled"` || string(ev["uuid"]) != `"`+rec.UUID.String()+`"` ||
			string(ev["createdAt"]) != `"2026-10-10T10:00:07Z"` || string(ev["actorId"]) != `"`+sheetUUID.String()+`"` {
			t.Fatalf("event = %v, want the cancelled action with its actorId", stringify(ev))
		}
		for _, k := range []string{"payload", "masterAction"} {
			if _, ok := ev[k]; ok {
				t.Fatalf("a cancelled action event carries %q: %v", k, stringify(ev))
			}
		}
		return evs
	}
	speedResult := func(t *testing.T, raw json.RawMessage) (json.RawMessage, bool) {
		t.Helper()
		var act struct {
			ActorID string `json:"actorId"`
			Speed   struct {
				RollCheck map[string]json.RawMessage `json:"rollCheck"`
			} `json:"speed"`
		}
		if err := json.Unmarshal(raw, &act); err != nil {
			t.Fatalf("action: %v — %s", err, raw)
		}
		if act.ActorID != sheetUUID.String() || string(act.Speed.RollCheck["skillName"]) != `"Legerity"` {
			t.Fatalf("action = %s, want Gon's action declaring Legerity", raw)
		}
		r, ok := act.Speed.RollCheck["result"]
		return r, ok
	}

	t.Run("the master reads the whole action", func(t *testing.T) {
		ev := events(t, masterUUID)[0]
		if r, ok := speedResult(t, ev["action"]); !ok || string(r) != "13" {
			t.Fatalf("speed.rollCheck.result = %s (present %v), want 13 — Full keeps the numbers", r, ok)
		}
	})
	t.Run("the actor's owner reads what they declared, without the numbers", func(t *testing.T) {
		ev := events(t, ownerUUID)[0]
		if r, ok := speedResult(t, ev["action"]); ok {
			t.Fatalf("speed.rollCheck.result = %s, want it cut — Declaration", r)
		}
	})
	t.Run("another player reads only who cancelled", func(t *testing.T) {
		ev := events(t, otherUUID)[0]
		if raw, ok := ev["action"]; ok {
			t.Fatalf("another player received the action: %s", raw)
		}
	})
}
