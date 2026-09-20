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
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
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

// TestGetMatchHistoryCarriesSystemBias is the last leg of the SystemBias round trip: the
// column and the decode put it back on the domain Action, and this handler is what lets the
// front read it.
//
// It is exposed for the reason this surface already exposes RollCheckResponse.attempts to
// every viewer: the bias is public by omission. Both dice sets AND the result already travel,
// so which set the engine read is derivable — withholding the field only forces the client
// into the algebra that ReactionTotal exists to spare it.
//
// RollCondition is NOT the same call and stays internal: the master's intervention has its
// own surface in overridden_action_values.
func TestGetMatchHistoryCarriesSystemBias(t *testing.T) {
	userUUID, matchUUID, actorID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	mainAction := action.NewAction(actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{}, nil, nil, nil, nil, nil, nil,
		&action.Interact{Kind: action.InteractOpen},
	)
	reaction := action.NewAction(actorID, nil, mainAction.GetID(), nil,
		action.ActionSpeed{}, nil, nil, nil, nil, &action.Dodge{}, nil, nil,
	)
	reaction.ReactionKind = action.ReactDodge
	reaction.SystemBias = -1

	scenes := []match.HistoryScene{{
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

	_, api := humatest.New(t)
	handler := apiMatch.GetMatchHistoryHandler(&mockGetMatchHistory{
		fn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
			return &match.GetMatchHistoryResult{Scenes: scenes}, nil
		},
	})
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/matches/{uuid}/history",
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/history")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d. Body: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					Action struct {
						SystemBias int `json:"systemBias"`
						Interact   *struct {
							Kind string `json:"kind"`
						} `json:"interact"`
					} `json:"action"`
					Reactions []struct {
						SystemBias int `json:"systemBias"`
					} `json:"reactions"`
				} `json:"turns"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v — body: %s", err, resp.Body.String())
	}

	turn := body.Scenes[0].Rounds[0].Turns[0]
	if turn.Action.Interact == nil || turn.Action.Interact.Kind != string(action.InteractOpen) {
		t.Fatalf("the interaction is missing from the response: %s", resp.Body.String())
	}
	if len(turn.Reactions) != 1 {
		t.Fatalf("reactions = %d, want 1", len(turn.Reactions))
	}
	if turn.Reactions[0].SystemBias != -1 {
		t.Errorf("reaction systemBias = %d, want -1 — the Disadvantage it was charged under",
			turn.Reactions[0].SystemBias)
	}
	// The action itself was charged nothing, and must read back as nothing rather than
	// inheriting its reaction's bias.
	if turn.Action.SystemBias != 0 {
		t.Errorf("the action's systemBias = %d, want 0", turn.Action.SystemBias)
	}
}

// TestGetMatchHistoryCarriesPayouts is the REST half of the same pair as the WebSocket's
// TestResolutionUpdatedPayloadCarriesPayouts: the deny-list lives in
// service.ProjectResolution (already applied upstream by the use case) and the field that
// makes it observable lives in this DTO.
//
// The scenes here are what a THIRD PARTY's use case already returned, i.e. post-projection:
// the repel keeps its label and its payout, the closed dodge arrives demoted to "dodge" with
// its reserve already stripped. This handler must serialize that as-is — no second filter.
func TestGetMatchHistoryCarriesPayouts(t *testing.T) {
	userUUID, matchUUID := uuid.New(), uuid.New()
	repelTarget, attacker := uuid.New(), uuid.New()
	now := time.Now()

	act := action.NewAction(attacker, []uuid.UUID{repelTarget}, uuid.Nil, nil,
		action.ActionSpeed{}, nil, nil, &action.Attack{}, nil, nil, nil, nil)

	scenes := []match.HistoryScene{{
		UUID: uuid.New(), Category: "combat", CreatedAt: now,
		Rounds: []match.HistoryRound{{
			UUID: uuid.New(), Mode: "combat", CreatedAt: now,
			Turns: []match.HistoryTurn{{
				UUID: uuid.New(), CreatedAt: now, FinishedAt: now,
				Action: *act,
				Resolution: &service.TurnResolution{
					IsSettled: true,
					CharacterResults: []service.CharacterResult{{
						TargetID: repelTarget, ReactionKind: string(action.ReactRepel),
						Ladder: service.LadderOutcome{Rung: service.RungNearMiss, Difference: 3},
						Payouts: []domainMatch.Modifier{{
							Amount: -3, Applies: domainMatch.DimActionSpeed,
							Source: domainMatch.SourceSystem, Against: domainMatch.ScopeAnyone(),
							ExpiresAt: domainMatch.LifetimeNextTurn,
							Reason:    "repel: near miss penalty",
						}},
					}},
				},
			}},
		}},
	}}

	_, api := humatest.New(t)
	handler := apiMatch.GetMatchHistoryHandler(&mockGetMatchHistory{
		fn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchHistoryResult, error) {
			return &match.GetMatchHistoryResult{Scenes: scenes}, nil
		},
	})
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/matches/{uuid}/history",
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/history")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d. Body: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					Resolution struct {
						Targets []struct {
							Payouts []struct {
								Amount      int    `json:"amount"`
								Applies     string `json:"applies"`
								Source      string `json:"source"`
								AgainstKind string `json:"againstKind"`
								ExpiresAt   string `json:"expiresAt"`
								Reason      string `json:"reason"`
							} `json:"payouts"`
						} `json:"targets"`
					} `json:"resolution"`
				} `json:"turns"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v — body: %s", err, resp.Body.String())
	}

	targets := body.Scenes[0].Rounds[0].Turns[0].Resolution.Targets
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	if len(targets[0].Payouts) != 1 {
		t.Fatalf("the repel penalty is missing from the history: %s", resp.Body.String())
	}
	got := targets[0].Payouts[0]
	if got.Amount != -3 || got.Applies != string(domainMatch.DimActionSpeed) ||
		got.Source != string(domainMatch.SourceSystem) ||
		got.AgainstKind != domainMatch.ScopeAnyone().Kind() ||
		got.ExpiresAt != string(domainMatch.LifetimeNextTurn) ||
		got.Reason != "repel: near miss penalty" {
		t.Errorf("payout = %+v, want the near-miss penalty verbatim", got)
	}
}
