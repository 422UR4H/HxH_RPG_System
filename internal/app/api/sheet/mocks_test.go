package sheet_test

import (
	"context"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

// mockCreateCharacterSheet implements charactersheet.ICreateCharacterSheet
type mockCreateCharacterSheet struct {
	fn func(ctx context.Context, input *charactersheet.CreateCharacterSheetInput) (*domainSheet.CharacterSheet, error)
}

func (m *mockCreateCharacterSheet) CreateCharacterSheet(
	ctx context.Context, input *charactersheet.CreateCharacterSheetInput,
) (*domainSheet.CharacterSheet, error) {
	return m.fn(ctx, input)
}

// mockGetCharacterSheet implements charactersheet.IGetCharacterSheet
type mockGetCharacterSheet struct {
	fn func(ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID) (*domainSheet.CharacterSheet, error)
}

func (m *mockGetCharacterSheet) GetCharacterSheet(
	ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
) (*domainSheet.CharacterSheet, error) {
	return m.fn(ctx, charSheetId, playerId)
}

// mockListCharacterSheets implements charactersheet.IListCharacterSheets
type mockListCharacterSheets struct {
	fn func(ctx context.Context, playerId uuid.UUID) ([]model.CharacterSheetSummary, error)
}

func (m *mockListCharacterSheets) ListCharacterSheets(
	ctx context.Context, playerId uuid.UUID,
) ([]model.CharacterSheetSummary, error) {
	return m.fn(ctx, playerId)
}

// mockListCharacterClasses implements charactersheet.IListCharacterClasses
type mockListCharacterClasses struct {
	listClassesFn func() []cc.CharacterClass
	listSheetsFn  func() []domainSheet.HalfSheet
}

func (m *mockListCharacterClasses) ListCharacterClasses() []cc.CharacterClass {
	return m.listClassesFn()
}

func (m *mockListCharacterClasses) ListClassSheets() []domainSheet.HalfSheet {
	return m.listSheetsFn()
}

// mockGetCharacterClass implements charactersheet.IGetCharacterClass
type mockGetCharacterClass struct {
	getClassFn func(name string) (cc.CharacterClass, error)
	getSheetFn func(name string) (domainSheet.HalfSheet, error)
}

func (m *mockGetCharacterClass) GetCharacterClass(name string) (cc.CharacterClass, error) {
	return m.getClassFn(name)
}

func (m *mockGetCharacterClass) GetClassSheet(name string) (domainSheet.HalfSheet, error) {
	return m.getSheetFn(name)
}

// mockUpdateNenHexagonValue implements charactersheet.IUpdateNenHexagonValue
type mockUpdateNenHexagonValue struct {
	fn func(ctx context.Context, charSheet *domainSheet.CharacterSheet, method string) (*spiritual.NenHexagonUpdateResult, error)
}

func (m *mockUpdateNenHexagonValue) UpdateNenHexagonValue(
	ctx context.Context, charSheet *domainSheet.CharacterSheet, method string,
) (*spiritual.NenHexagonUpdateResult, error) {
	return m.fn(ctx, charSheet, method)
}

// Test helpers

func buildTestCharacterSheet(t *testing.T) *domainSheet.CharacterSheet {
	t.Helper()
	factory := domainSheet.NewCharacterSheetFactory()
	playerUUID := uuid.New()
	profile := domainSheet.CharacterProfile{
		NickName:  "Gon",
		FullName:  "Gon Freecss",
		Alignment: "Chaotic-Good",
	}
	cs, err := factory.Build(&playerUUID, nil, nil, profile, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to build test character sheet: %v", err)
	}
	cs.UUID = uuid.New()
	return cs
}

func buildTestCharacterSheetWithHex(t *testing.T) *domainSheet.CharacterSheet {
	t.Helper()
	factory := domainSheet.NewCharacterSheetFactory()
	playerUUID := uuid.New()
	hexValue := 3
	profile := domainSheet.CharacterProfile{
		NickName:  "Gon",
		FullName:  "Gon Freecss",
		Alignment: "Chaotic-Good",
	}
	cs, err := factory.Build(&playerUUID, nil, nil, profile, &hexValue, nil, nil)
	if err != nil {
		t.Fatalf("failed to build test character sheet with hex: %v", err)
	}
	cs.UUID = uuid.New()
	return cs
}

func buildTestHalfSheet(t *testing.T) domainSheet.HalfSheet {
	t.Helper()
	factory := domainSheet.NewCharacterSheetFactory()
	profile := domainSheet.CharacterProfile{
		NickName:  "Hunter",
		FullName:  "Hunter Class",
	}
	hs, err := factory.BuildHalfSheet(profile, nil)
	if err != nil {
		t.Fatalf("failed to build test half sheet: %v", err)
	}
	return *hs
}
