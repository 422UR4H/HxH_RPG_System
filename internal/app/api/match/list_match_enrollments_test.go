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
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListMatchEnrollmentsHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	makeFixture := func() []*enrollmentEntity.Enrollment {
		return []*enrollmentEntity.Enrollment{
			{
				UUID:      uuid.New(),
				Status:    "pending",
				CreatedAt: now,
				CharacterSheet: model.CharacterSheetSummary{
					UUID:     uuid.New(),
					NickName: "Gon",
					FullName: "Gon Freecss",
					Birthday: now,
				},
				Player: enrollmentEntity.PlayerRef{UUID: uuid.New(), Nick: "tiago"},
			},
		}
	}

	tests := []struct {
		name           string
		ucFn           func(ctx context.Context, matchID, uid uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error)
		wantStatus     int
		wantPrivateNil bool // when status==200, asserts the first row's character_sheet.private nullness
	}{
		{
			name: "200 with private populated when ViewerIsMaster",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    makeFixture(),
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: false,
		},
		{
			name: "200 with private null when not master",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    makeFixture(),
					ViewerIsMaster: false,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: true,
		},
		{
			name: "200 with empty list",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    []*enrollmentEntity.Enrollment{},
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 on ErrMatchNotFound",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "403 on ErrInsufficientPermissions",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "500 on generic error",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := apiMatch.ListMatchEnrollmentsHandler(&mockListMatchEnrollments{fn: tc.ucFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}/enrollments",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/enrollments")

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
			enrollments, ok := body["enrollments"].([]any)
			if !ok {
				t.Fatal("response missing 'enrollments' array")
			}
			if len(enrollments) == 0 {
				return // empty-list case
			}
			row := enrollments[0].(map[string]any)
			sheet := row["character_sheet"].(map[string]any)
			privateField, present := sheet["private"]
			if !present {
				t.Fatal("character_sheet.private must be present (null or populated), not omitted")
			}
			if tc.wantPrivateNil {
				if privateField != nil {
					t.Errorf("character_sheet.private = %v, want null", privateField)
				}
			} else {
				if privateField == nil {
					t.Error("character_sheet.private = null, want populated object")
				}
			}
		})
	}
}
