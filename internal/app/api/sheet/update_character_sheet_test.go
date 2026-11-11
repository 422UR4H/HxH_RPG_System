package sheet_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockUpdateCharacterSheet struct{ called bool }

func (m *mockUpdateCharacterSheet) UpdateCharacterSheet(
	_ context.Context, _, _ uuid.UUID, _ *charactersheet.CreateCharacterSheetInput,
) error {
	m.called = true
	return nil
}

// Os erros de castRequest nascem no handler, antes do use case: corpo real, UC não chamado.
func TestUpdateCharacterSheetHandler_CastRequestErrors(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(body, profile map[string]any)
		wantStatus int
	}{
		{"emoji in nickname", func(_, p map[string]any) { p["nickname"] = "Gon😀" }, http.StatusUnprocessableEntity},
		{"emoji in fullname", func(_, p map[string]any) { p["fullname"] = "Gon Freecss 😀" }, http.StatusUnprocessableEntity},
		{"nickname too short", func(_, p map[string]any) { p["nickname"] = "G" }, http.StatusUnprocessableEntity},
		{"invalid enum", func(b, _ map[string]any) { b["attributePoints"] = map[string]any{"Stamina": 1} }, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			uc := &mockUpdateCharacterSheet{}
			huma.Register(api, huma.Operation{
				Method: http.MethodPut,
				Path:   "/charactersheets/{uuid}",
			}, sheet.UpdateCharacterSheetHandler(uc, nil))

			profile := map[string]any{
				"nickname":         "Gon",
				"fullname":         "Gon Freecss",
				"alignment":        "Chaotic-Good",
				"description":      "A young hunter",
				"briefDescription": "Hunter boy",
				"age":              12,
				"birthday":         "0000-05-15T00:00:00Z",
			}
			body := map[string]any{
				"profile":           profile,
				"characterClass":    "Hunter",
				"skillsExps":        map[string]any{},
				"proficienciesExps": map[string]any{},
				"attributePoints":   map[string]any{"Resistance": 1, "Agility": 1, "Flexibility": 1},
			}
			tt.mutate(body, profile)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
			resp := api.PutCtx(ctx, "/charactersheets/"+uuid.New().String(), body)
			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if uc.called {
				t.Error("use case must not be called")
			}
		})
	}
}
