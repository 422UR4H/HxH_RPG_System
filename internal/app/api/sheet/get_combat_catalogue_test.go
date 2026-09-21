package sheet_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

// newSheetWithProficiencies builds a character sheet whose common weapon proficiencies sit at
// the given levels. The factory (character_sheet_factory.go) always starts a sheet with an
// empty proficiency manager, so tests that need a specific level have to seed exp points that
// land exactly on it: IncreasePoints by the table's aggregate exp for that level.
func newSheetWithProficiencies(t *testing.T, levels map[enum.WeaponName]int) *sheetEntity.CharacterSheet {
	t.Helper()
	charSheet := buildTestCharacterSheet(t)

	expTable := experience.NewExpTable(sheetEntity.PHYSICAL_SKILLS_COEFF)
	for name, level := range levels {
		exp := experience.NewExperience(expTable)
		exp.IncreasePoints(expTable.GetAggregateExpByLvl(level))
		// physSkillsExp is nil: these tests only read GetLevel(), never trigger
		// CascadeUpgradeTrigger, so the cascade reference is never dereferenced.
		domainProf := proficiency.NewProficiency(name, *exp, nil)
		if err := charSheet.AddCommonProficiency(name, domainProf); err != nil {
			t.Fatalf("AddCommonProficiency(%s): %v", name, err)
		}
	}
	return charSheet
}

// mustUnmarshalJSON keeps test bodies focused on assertions instead of error boilerplate.
func mustUnmarshalJSON(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
}

// authContext puts a user UUID in context the way the real apiAuth middleware would after a
// successful JWT check. Handler tests in this package (see get_character_sheet_test.go) bypass
// the middleware chain and call GetCtx directly with the context value the handler reads.
func authContext() context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
}

// registerCombatCatalogueRoute registers only the route under test, as the neighboring handler
// tests do, instead of pulling in the full routes.go registration.
func registerCombatCatalogueRoute(t *testing.T, mock *mockGetCharacterSheet) (context.Context, humatest.TestAPI) {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	handler := sheet.GetCombatCatalogueHandler(mock)

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/charactersheets/{uuid}/combat-catalogue",
	}, handler)

	return authContext(), api
}

// As armas de uma action não são o catálogo do sistema: são as que ESTE personagem sabe usar,
// mais o golpe corporal, sempre. Quando existir inventário isto vira "o que ele carrega";
// hoje proficiência é a melhor aproximação — é o que ele sabe empunhar.
func TestCombatCatalogueListsProficientWeaponsPlusFist(t *testing.T) {
	testSheet := newSheetWithProficiencies(t, map[enum.WeaponName]int{
		enum.Sword:  4,
		enum.Dagger: 2,
	})

	mock := &mockGetCharacterSheet{
		fn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
			testSheet.UUID = id
			return testSheet, nil
		},
	}
	ctx, api := registerCombatCatalogueRoute(t, mock)

	resp := api.GetCtx(ctx, "/charactersheets/"+testSheet.UUID.String()+"/combat-catalogue")
	if resp.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body sheet.GetCombatCatalogueResponseBody
	mustUnmarshalJSON(t, resp.Body.Bytes(), &body)

	names := map[string]int{}
	for _, w := range body.Weapons {
		names[w.Name] = w.ProficiencyLevel
	}
	if len(body.Weapons) != 3 {
		t.Fatalf("got %d weapons, want 3 (Sword, Dagger, Fist): %v", len(body.Weapons), names)
	}
	if _, ok := names["Fist"]; !ok {
		t.Fatal("Fist is missing; the bare-handed blow does not depend on training")
	}
	if names["Sword"] != 4 {
		t.Fatalf("Sword came back at level %d, want 4", names["Sword"])
	}
	if len(body.Skills) == 0 {
		t.Fatal("skills came back empty; the front would go on inventing strings like combat_strength")
	}
}

// Uma ficha que JÁ tem proficiência em Fist devolve Fist uma vez só, com o nível real — não
// duplicado pelo acréscimo incondicional.
func TestCombatCatalogueDoesNotDuplicateFist(t *testing.T) {
	testSheet := newSheetWithProficiencies(t, map[enum.WeaponName]int{enum.Fist: 3})

	mock := &mockGetCharacterSheet{
		fn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
			testSheet.UUID = id
			return testSheet, nil
		},
	}
	ctx, api := registerCombatCatalogueRoute(t, mock)

	resp := api.GetCtx(ctx, "/charactersheets/"+testSheet.UUID.String()+"/combat-catalogue")
	if resp.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var body sheet.GetCombatCatalogueResponseBody
	mustUnmarshalJSON(t, resp.Body.Bytes(), &body)

	fists := 0
	level := -1
	for _, w := range body.Weapons {
		if w.Name == "Fist" {
			fists++
			level = w.ProficiencyLevel
		}
	}
	if fists != 1 {
		t.Fatalf("Fist appeared %d times, want exactly 1", fists)
	}
	if level != 3 {
		t.Fatalf("Fist came back at level %d, want the real 3", level)
	}
}
