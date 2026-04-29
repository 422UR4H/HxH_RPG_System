package sheet_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestListClassesHandler(t *testing.T) {
	tests := []struct {
		name          string
		listClassesFn func() []cc.CharacterClass
		listSheetsFn  func() []domainSheet.HalfSheet
		wantStatus    int
	}{
		{
			name: "success_empty_list",
			listClassesFn: func() []cc.CharacterClass {
				return []cc.CharacterClass{}
			},
			listSheetsFn: func() []domainSheet.HalfSheet {
				return []domainSheet.HalfSheet{}
			},
			wantStatus: http.StatusOK,
		},
		{
			// The handler has no error path — always returns 200.
			// This test verifies success with populated data.
			name: "success_with_data",
			listClassesFn: func() []cc.CharacterClass {
				return []cc.CharacterClass{{}}
			},
			listSheetsFn: func() []domainSheet.HalfSheet {
				hs := buildTestHalfSheet(t)
				return []domainSheet.HalfSheet{hs}
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListCharacterClasses{
				listClassesFn: tt.listClassesFn,
				listSheetsFn:  tt.listSheetsFn,
			}
			handler := sheet.ListClassesHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/classes",
			}, handler)

			resp := api.GetCtx(context.Background(), "/classes")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
