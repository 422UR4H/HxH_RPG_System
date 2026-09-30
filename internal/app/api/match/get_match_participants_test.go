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
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetMatchParticipantsHandler(t *testing.T) {
	requestUUID := uuid.New()
	matchUUID := uuid.New()
	ownerUUID := uuid.New()
	otherPlayerUUID := uuid.New()
	strangerUUID := uuid.New()
	now := time.Now()

	// Fixed order: [0] owned by ownerUUID, [1] owned by otherPlayerUUID, [2] NPC (no owner).
	makeFixture := func() []*matchEntity.Participant {
		return []*matchEntity.Participant{
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				Sheet: csEntity.Summary{
					UUID:       uuid.New(),
					PlayerUUID: &ownerUUID,
					NickName:   "Gon",
					FullName:   "Gon Freecss",
					Birthday:   now,
				},
				JoinedAt: now,
				LeftAt:   nil,
			},
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				Sheet: csEntity.Summary{
					UUID:       uuid.New(),
					PlayerUUID: &otherPlayerUUID,
					NickName:   "Killua",
					FullName:   "Killua Zoldyck",
					Birthday:   now,
				},
				JoinedAt: now,
				LeftAt:   nil,
			},
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				Sheet: csEntity.Summary{
					UUID:       uuid.New(),
					PlayerUUID: nil,
					NickName:   "Goblin",
					FullName:   "Goblin NPC",
					Birthday:   now,
				},
				JoinedAt: now,
				LeftAt:   nil,
			},
		}
	}

	tests := []struct {
		name            string
		ucFn            func(ctx context.Context, matchID, uid uuid.UUID) (*match.GetMatchParticipantsResult, error)
		wantStatus      int
		wantPrivateNils []bool // per participant, in fixture order; nil when status != 200
	}{
		{
			name: "master receives private on every participant, including the NPC",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return &match.GetMatchParticipantsResult{
					Participants:   makeFixture(),
					ViewerIsMaster: true,
					ViewerUUID:     uuid.New(), // master's own user UUID, irrelevant for the check
				}, nil
			},
			wantStatus:      http.StatusOK,
			wantPrivateNils: []bool{false, false, false},
		},
		{
			name: "owner receives private only on their own sheet",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return &match.GetMatchParticipantsResult{
					Participants:   makeFixture(),
					ViewerIsMaster: false,
					ViewerUUID:     ownerUUID,
				}, nil
			},
			wantStatus:      http.StatusOK,
			wantPrivateNils: []bool{false, true, true},
		},
		{
			name: "a player who owns none of the sheets receives private on none, not even the NPC",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return &match.GetMatchParticipantsResult{
					Participants:   makeFixture(),
					ViewerIsMaster: false,
					ViewerUUID:     strangerUUID,
				}, nil
			},
			wantStatus:      http.StatusOK,
			wantPrivateNils: []bool{true, true, true},
		},
		{
			name: "200 with empty list",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return &match.GetMatchParticipantsResult{
					Participants:   []*matchEntity.Participant{},
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 on ErrMatchNotFound",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return nil, match.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "403 on ErrInsufficientPermissions",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return nil, authUC.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "500 on generic error",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*match.GetMatchParticipantsResult, error) {
				return nil, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := apiMatch.GetMatchParticipantsHandler(&mockGetMatchParticipants{fn: tc.ucFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}/participants",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, requestUUID)
			resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/participants")

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
			participants, ok := body["participants"].([]any)
			if !ok {
				t.Fatal("response missing 'participants' array")
			}
			if len(participants) == 0 {
				return // empty-list case
			}
			if len(participants) != len(tc.wantPrivateNils) {
				t.Fatalf("len(participants) = %d, want %d", len(participants), len(tc.wantPrivateNils))
			}
			for i, wantNil := range tc.wantPrivateNils {
				row := participants[i].(map[string]any)
				sheet := row["characterSheet"].(map[string]any)
				privateField, present := sheet["private"]
				if !present {
					t.Fatalf("participant[%d]: character_sheet.private must be present (null or populated), not omitted", i)
				}
				if wantNil {
					if privateField != nil {
						t.Errorf("participant[%d]: character_sheet.private = %v, want null", i, privateField)
					}
				} else {
					if privateField == nil {
						t.Errorf("participant[%d]: character_sheet.private = null, want populated object", i)
					}
				}
			}
		})
	}
}
