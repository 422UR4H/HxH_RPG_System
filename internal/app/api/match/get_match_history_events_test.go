package match_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func getHistory(t *testing.T, scenes []match.HistoryScene) []byte {
	t.Helper()
	_, api := humatest.New(t)
	handler := apiMatch.GetMatchHistoryHandler(&mockGetMatchHistory{
		fn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
			return &match.GetMatchHistoryResult{Scenes: scenes}, nil
		},
	})
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{uuid}/history"}, handler)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	resp := api.GetCtx(ctx, "/matches/"+uuid.New().String()+"/history")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d. Body: %s", resp.Code, resp.Body.String())
	}
	return resp.Body.Bytes()
}

// TestGetMatchHistoryCarriesEventsAndMasterActions is the wire half of B15 + §4.8: the round's
// events and the turn's masterActions reach the JSON in the order the use case put them, and
// Views — who saw what, which the MASTER's projection still carries — never does, for anyone.
func TestGetMatchHistoryCarriesEventsAndMasterActions(t *testing.T) {
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	turnUUID, player := uuid.New(), uuid.New()
	views := map[uuid.UUID]masteraction.View{player: masteraction.ViewLeft}

	inTurn := masteraction.Record{
		UUID: uuid.New(), TurnUUID: &turnUUID, Kind: masteraction.KindMovePiece,
		Content:    json.RawMessage(`{"characterId":"c","pieceId":"p","from":[1,1,0],"to":[2,2,0]}`),
		Views:      views,
		HappenedAt: now.Add(time.Second),
	}
	outside := masteraction.Record{
		UUID: uuid.New(), Kind: masteraction.KindWallInteract,
		Content:    json.RawMessage(`{"wallIds":["w1"],"interact":"open"}`),
		Views:      views,
		HappenedAt: now.Add(3 * time.Second),
	}
	modeChange := matchevent.Event{
		UUID: uuid.New(), Kind: matchevent.KindRoundModeChanged,
		Payload: json.RawMessage(`{"from":"Free","to":"Race"}`), CreatedAt: now.Add(2 * time.Second),
	}
	act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)

	body := getHistory(t, []match.HistoryScene{{
		UUID: uuid.New(), Category: "battle", CreatedAt: now,
		Rounds: []match.HistoryRound{
			{
				UUID: uuid.New(), Mode: "Race", CreatedAt: now,
				Turns: []match.HistoryTurn{{
					UUID: turnUUID, CreatedAt: now, FinishedAt: now,
					Action: *act, Reactions: []action.Action{},
					MasterActions: []masteraction.Record{inTurn},
				}},
				Events: []match.HistoryEvent{
					{Kind: match.HistoryEventRoundModeChanged, At: modeChange.CreatedAt, RoundModeChange: &modeChange},
					{Kind: match.HistoryEventMasterAction, At: outside.HappenedAt, MasterAction: &outside},
				},
			},
			// A round with nothing in it: both lists must still be [] on the wire, never null —
			// and a turn built before the use case filled MasterActions (nil) as well.
			{
				UUID: uuid.New(), Mode: "Free", CreatedAt: now,
				Turns: []match.HistoryTurn{{
					UUID: uuid.New(), CreatedAt: now, FinishedAt: now,
					Action: *act, Reactions: []action.Action{},
				}},
			},
		},
	}})

	if strings.Contains(string(body), `"views"`) || strings.Contains(string(body), player.String()) {
		t.Fatalf("who saw what reached the wire: %s", body)
	}
	if strings.Contains(string(body), `"masterActions":null`) || strings.Contains(string(body), `"events":null`) {
		t.Fatalf("an empty list went out as null: %s", body)
	}

	var got struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					MasterActions []map[string]json.RawMessage `json:"masterActions"`
				} `json:"turns"`
				Events []map[string]json.RawMessage `json:"events"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v — %s", err, body)
	}
	r0, r1 := got.Scenes[0].Rounds[0], got.Scenes[0].Rounds[1]

	if len(r1.Events) != 0 || r1.Events == nil || len(r1.Turns[0].MasterActions) != 0 || r1.Turns[0].MasterActions == nil {
		t.Fatalf("the empty round = %+v, want [] events and [] masterActions", r1)
	}

	mas := r0.Turns[0].MasterActions
	if len(mas) != 1 {
		t.Fatalf("masterActions = %d, want 1", len(mas))
	}
	wantMA := map[string]string{
		"uuid": `"` + inTurn.UUID.String() + `"`, "kind": `"movePiece"`,
		"turnId": `"` + turnUUID.String() + `"`, "happenedAt": `"2026-09-27T10:00:01Z"`,
	}
	for k, v := range wantMA {
		if string(mas[0][k]) != v {
			t.Fatalf("masterActions[0].%s = %s, want %s", k, mas[0][k], v)
		}
	}
	if string(mas[0]["content"]) != string(inTurn.Content) {
		t.Fatalf("masterActions[0].content = %s, want %s", mas[0]["content"], inTurn.Content)
	}

	if len(r0.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(r0.Events))
	}
	ev0, ev1 := r0.Events[0], r0.Events[1]
	if string(ev0["kind"]) != `"roundModeChanged"` || string(ev0["uuid"]) != `"`+modeChange.UUID.String()+`"` ||
		string(ev0["createdAt"]) != `"2026-09-27T10:00:02Z"` || string(ev0["payload"]) != `{"from":"Free","to":"Race"}` {
		t.Fatalf("events[0] = %v, want the regime change with its payload", stringify(ev0))
	}
	if _, ok := ev0["masterAction"]; ok {
		t.Fatalf("a regime change carries a masterAction: %v", stringify(ev0))
	}
	if string(ev1["kind"]) != `"masterAction"` || string(ev1["uuid"]) != `"`+outside.UUID.String()+`"` ||
		string(ev1["createdAt"]) != `"2026-09-27T10:00:03Z"` {
		t.Fatalf("events[1] = %v, want the master action outside the turn", stringify(ev1))
	}
	if _, ok := ev1["payload"]; ok {
		t.Fatalf("a master action event carries a payload: %v", stringify(ev1))
	}
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(ev1["masterAction"], &nested); err != nil {
		t.Fatalf("events[1].masterAction: %v", err)
	}
	if string(nested["kind"]) != `"wallInteract"` || string(nested["content"]) != string(outside.Content) {
		t.Fatalf("events[1].masterAction = %v", stringify(nested))
	}
	if _, ok := nested["turnId"]; ok {
		t.Fatalf("a master action with no turn carries a turnId: %v", stringify(nested))
	}
}

func stringify(m map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = string(v)
	}
	return out
}

// O fio do histórico leva os campos do relógio: tempos da história ingênuos (sem Z), omitidos
// quando nulos; counted/drained no round; os eventos do relógio com o payload repassado.
func TestGetMatchHistoryCarriesClockFields(t *testing.T) {
	now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	start := time.Date(2000, 1, 1, 23, 59, 0, 0, time.UTC)
	end := time.Date(2000, 1, 1, 23, 59, 59, 0, time.UTC)
	turnAt := time.Date(2000, 1, 1, 23, 59, 30, 0, time.UTC)
	final := "Fim da luta"
	adjusted := matchevent.Event{
		UUID: uuid.New(), Kind: matchevent.KindClockAdjusted, CreatedAt: now,
		Payload: json.RawMessage(`{"from":"2000-01-01T23:59:00","to":"2000-01-01T23:59:30","deltaSeconds":30}`),
	}
	act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)

	body := getHistory(t, []match.HistoryScene{
		{
			UUID: uuid.New(), Category: "battle", CreatedAt: now,
			BriefFinalDescription: &final, StoryStartedAt: &start, StoryFinishedAt: &end,
			Rounds: []match.HistoryRound{{
				UUID: uuid.New(), Mode: "Race", CreatedAt: now, Counted: true, Drained: true,
				Turns: []match.HistoryTurn{{
					UUID: uuid.New(), CreatedAt: now, FinishedAt: now, StoryAt: &turnAt,
					Action: *act, Reactions: []action.Action{},
				}},
				Events: []match.HistoryEvent{{Kind: match.HistoryEventClockAdjusted, At: now, ClockChange: &adjusted}},
			}},
		},
		// sem relógio: nada de campo nulo no fio
		{UUID: uuid.New(), Category: "roleplay", CreatedAt: now, Rounds: []match.HistoryRound{{
			UUID: uuid.New(), Mode: "Free", CreatedAt: now,
			Turns: []match.HistoryTurn{{UUID: uuid.New(), CreatedAt: now, FinishedAt: now, Action: *act, Reactions: []action.Action{}}},
		}}},
	})

	var got struct {
		Scenes []struct {
			BriefFinalDescription *string `json:"briefFinalDescription"`
			StoryStartedAt        *string `json:"storyStartedAt"`
			StoryFinishedAt       *string `json:"storyFinishedAt"`
			Rounds                []struct {
				Counted bool `json:"counted"`
				Drained bool `json:"drained"`
				Turns   []struct {
					StoryAt *string `json:"storyAt"`
				} `json:"turns"`
				Events []map[string]json.RawMessage `json:"events"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v — %s", err, body)
	}
	s0 := got.Scenes[0]
	if s0.BriefFinalDescription == nil || *s0.BriefFinalDescription != final ||
		s0.StoryStartedAt == nil || *s0.StoryStartedAt != "2000-01-01T23:59:00" ||
		s0.StoryFinishedAt == nil || *s0.StoryFinishedAt != "2000-01-01T23:59:59" {
		t.Errorf("scene clock fields = %v / %v / %v", s0.BriefFinalDescription, s0.StoryStartedAt, s0.StoryFinishedAt)
	}
	r0 := s0.Rounds[0]
	if !r0.Counted || !r0.Drained {
		t.Errorf("round counted=%v drained=%v, want both true", r0.Counted, r0.Drained)
	}
	if at := r0.Turns[0].StoryAt; at == nil || *at != "2000-01-01T23:59:30" {
		t.Errorf("turn storyAt = %v, want 2000-01-01T23:59:30", at)
	}
	if len(r0.Events) != 1 || string(r0.Events[0]["kind"]) != `"clockAdjusted"` ||
		string(r0.Events[0]["payload"]) != string(adjusted.Payload) ||
		string(r0.Events[0]["uuid"]) != `"`+adjusted.UUID.String()+`"` {
		t.Errorf("events = %v, want the clockAdjusted with its payload", r0.Events)
	}
	for _, k := range []string{`"briefFinalDescription":null`, `"storyStartedAt":null`, `"storyFinishedAt":null`, `"storyAt":null`} {
		if strings.Contains(string(body), k) {
			t.Errorf("a null story field reached the wire: %s", k)
		}
	}
}
