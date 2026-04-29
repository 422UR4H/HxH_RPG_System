package sheet_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestUpdateNenHexagonValueHandler(t *testing.T) {
	userUUID := uuid.New()
	sheetUUID := uuid.New()

	tests := []struct {
		name        string
		getSheetFn  func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error)
		updateHexFn func(ctx context.Context, cs *domainSheet.CharacterSheet, method string) (*spiritual.NenHexagonUpdateResult, error)
		wantStatus  int
	}{
		{
			name: "success",
			getSheetFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				cs := buildTestCharacterSheetWithHex(t)
				cs.UUID = id
				return cs, nil
			},
			updateHexFn: func(ctx context.Context, cs *domainSheet.CharacterSheet, method string) (*spiritual.NenHexagonUpdateResult, error) {
				return &spiritual.NenHexagonUpdateResult{
					PercentList:   map[enum.CategoryName]float64{enum.Reinforcement: 100.0},
					CategoryName:  enum.Reinforcement,
					CurrentHexVal: 4,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "sheet_not_found",
			getSheetFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				return nil, charactersheet.ErrCharacterSheetNotFound
			},
			updateHexFn: func(ctx context.Context, cs *domainSheet.CharacterSheet, method string) (*spiritual.NenHexagonUpdateResult, error) {
				t.Fatal("UpdateNenHexagonValue should not be called when GetCharacterSheet fails")
				return nil, nil
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal_server_error",
			getSheetFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				return nil, errors.New("db connection failed")
			},
			updateHexFn: func(ctx context.Context, cs *domainSheet.CharacterSheet, method string) (*spiritual.NenHexagonUpdateResult, error) {
				t.Fatal("UpdateNenHexagonValue should not be called when GetCharacterSheet fails")
				return nil, nil
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			getSheetMock := &mockGetCharacterSheet{fn: tt.getSheetFn}
			updateHexMock := &mockUpdateNenHexagonValue{fn: tt.updateHexFn}
			handler := sheet.UpdateNenHexagonValueHandler(updateHexMock, getSheetMock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/charactersheets/{character_sheet_uuid}/nen-hexagon/{method}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/charactersheets/"+sheetUUID.String()+"/nen-hexagon/increase")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
