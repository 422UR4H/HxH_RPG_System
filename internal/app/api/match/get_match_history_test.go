package match_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	authUC "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetMatchHistoryHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	actorID := uuid.New()
	now := time.Now()

	// thirdPartyProjectedScenes is what the use case returns to a viewer who is NEITHER the
	// master NOR the owner — i.e. already run through service.ProjectAction. The reaction
	// arrives demoted (dodge, not closedDodge) and its Evasion skill entry is simply absent.
	// This handler must serialize that as-is: no second filter here.
	thirdPartyProjectedScenes := func() []match.HistoryScene {
		mainAction := action.NewAction(actorID, []uuid.UUID{uuid.New()}, uuid.Nil,
			[]action.Skill{{SkillName: enum.Legerity.String()}},
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil,
		)
		reaction := action.NewAction(actorID, nil, uuid.New(),
			[]action.Skill{{SkillName: enum.Legerity.String()}}, // Evasion already stripped upstream
			action.ActionSpeed{}, nil, nil, nil, nil, &action.Dodge{}, nil, nil,
		)
		reaction.ReactionKind = action.ReactDodge // already demoted upstream

		return []match.HistoryScene{{
			UUID: uuid.New(), Category: "combat", CreatedAt: now,
			Rounds: []match.HistoryRound{{
				UUID: uuid.New(), Mode: "combat", CreatedAt: now,
				Turns: []match.HistoryTurn{{
					UUID: uuid.New(), CreatedAt: now, FinishedAt: now,
					Action:    *mainAction,
					Reactions: []action.Action{*reaction},
				}},
			}},
		}}
	}

	tests := []struct {
		name       string
		ucFn       func(ctx context.Context, matchID, uid uuid.UUID) (*match.GetMatchHistoryResult, error)
		wantStatus int
	}{
		{
			name: "200 with a third party's reaction demoted",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
				return &match.GetMatchHistoryResult{Scenes: thirdPartyProjectedScenes()}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "200 with empty history",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
				return &match.GetMatchHistoryResult{Scenes: []match.HistoryScene{}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 on ErrMatchNotFound",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
				return nil, match.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "403 on ErrInsufficientPermissions",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
				return nil, authUC.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "500 on generic error",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
				return nil, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := apiMatch.GetMatchHistoryHandler(&mockGetMatchHistory{fn: tc.ucFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}/history",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/history")

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. Body: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			var body map[string]any
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			scenes, ok := body["scenes"].([]any)
			if !ok {
				t.Fatal("response missing 'scenes' array")
			}
			if tc.name == "200 with empty history" {
				if len(scenes) != 0 {
					t.Fatalf("len(scenes) = %d, want 0", len(scenes))
				}
				return
			}

			scene := scenes[0].(map[string]any)
			round := scene["rounds"].([]any)[0].(map[string]any)
			turn := round["turns"].([]any)[0].(map[string]any)
			reactions := turn["reactions"].([]any)
			if len(reactions) != 1 {
				t.Fatalf("len(reactions) = %d, want 1", len(reactions))
			}
			reaction := reactions[0].(map[string]any)

			if got := reaction["reactionKind"]; got != "dodge" {
				t.Fatalf("reactionKind = %v, want \"dodge\" — the label is the leak", got)
			}
			skills, _ := reaction["skills"].([]any)
			for _, s := range skills {
				skill := s.(map[string]any)
				if skill["skillName"] == enum.Evasion.String() {
					t.Fatal("the Evasion skill entry leaked to a third party")
				}
			}
		})
	}
}
