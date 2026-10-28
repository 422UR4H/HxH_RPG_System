package sheet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestGetClassHandler(t *testing.T) {
	tests := []struct {
		name       string
		getClassFn func(name string) (cc.CharacterClass, error)
		getSheetFn func(name string) (*sheetEntity.CharacterSheet, error)
		wantStatus int
	}{
		{
			name: "success",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, nil
			},
			getSheetFn: func(name string) (*sheetEntity.CharacterSheet, error) {
				return buildTestClassSheet(t), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found_invalid_class_name",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				// erro real do enum para nome que não é classe
				_, err := enum.CharacterClassNameFrom("Nope")
				return cc.CharacterClass{}, err
			},
			getSheetFn: func(name string) (*sheetEntity.CharacterSheet, error) {
				t.Fatal("GetClassSheet should not be called when GetCharacterClass fails")
				return nil, nil
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_found_class_not_registered",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, charactersheet.ErrCharacterClassNotFound
			},
			getSheetFn: func(name string) (*sheetEntity.CharacterSheet, error) {
				t.Fatal("GetClassSheet should not be called when GetCharacterClass fails")
				return nil, nil
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal_error_class_lookup_failure",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, errors.New("unexpected error")
			},
			getSheetFn: func(name string) (*sheetEntity.CharacterSheet, error) {
				t.Fatal("GetClassSheet should not be called when GetCharacterClass fails")
				return nil, nil
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "internal_error_class_sheet_failure",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, nil
			},
			getSheetFn: func(name string) (*sheetEntity.CharacterSheet, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCharacterClass{
				getClassFn: tt.getClassFn,
				getSheetFn: tt.getSheetFn,
			}
			handler := sheet.GetClassHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/classes/{name}",
			}, handler)

			resp := api.GetCtx(context.Background(), "/classes/Hunter")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]json.RawMessage
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if _, ok := result["characterClass"]; !ok {
					t.Fatalf("response missing top-level 'characterClass' key. Body: %s", resp.Body.String())
				}
				if _, ok := result["CharacterClass"]; ok {
					t.Fatal("response must not carry PascalCase 'CharacterClass' key")
				}
			}
		})
	}
}

func TestGetClassHandler_spiritualEntries(t *testing.T) {
	spiritClass := &cc.CharacterClass{
		Profile:    *cc.NewClassProfile(enum.Monk, "", "monk", ""),
		SkillsExps: map[enum.SkillName]int{enum.Focus: 210, enum.Mop: 127},
	}

	tests := []struct {
		name         string
		charClass    *cc.CharacterClass
		wantSpirit   bool
		wantAttrName string
		wantSkill    string
	}{
		{"with_spiritual_exp", spiritClass, true, "Flame", "Focus"},
		{"without_spiritual_exp", nil, false, "Flame", "Focus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			mock := &mockGetCharacterClass{
				getClassFn: func(string) (cc.CharacterClass, error) { return cc.CharacterClass{}, nil },
				getSheetFn: func(string) (*sheetEntity.CharacterSheet, error) {
					return buildClassSheetFor(t, tt.charClass), nil
				},
			}
			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/classes/{name}",
			}, sheet.GetClassHandler(mock))

			resp := api.GetCtx(context.Background(), "/classes/Monk")
			if resp.Code != http.StatusOK {
				t.Fatalf("status %d, body: %s", resp.Code, resp.Body.String())
			}
			var result struct {
				CharacterClass struct {
					Abilities  map[string]any `json:"abilities"`
					Attributes map[string]any `json:"attributes"`
					Skills     map[string]any `json:"skills"`
				}
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			cls := result.CharacterClass
			_, hasAttr := cls.Attributes[tt.wantAttrName]
			_, hasSkill := cls.Skills[tt.wantSkill]
			_, hasAbility := cls.Abilities["Spirituals"]
			if hasAttr != tt.wantSpirit || hasSkill != tt.wantSpirit || hasAbility != tt.wantSpirit {
				t.Errorf("attr=%v skill=%v ability=%v, want all %v", hasAttr, hasSkill, hasAbility, tt.wantSpirit)
			}
		})
	}
}
