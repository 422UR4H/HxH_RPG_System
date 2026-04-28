package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type MockCharacterSheetRepo struct {
	CreateCharacterSheetFn               func(ctx context.Context, sheet *model.CharacterSheet) error
	ExistsCharacterWithNickFn            func(ctx context.Context, nick string) (bool, error)
	CountCharactersByPlayerUUIDFn        func(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUIDFn        func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUIDFn            func(ctx context.Context, uuid string) (*model.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUIDFn    func(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error)
	UpdateNenHexagonValueFn              func(ctx context.Context, uuid string, val int) error
	GetCharacterSheetRelationshipUUIDsFn func(ctx context.Context, uuid uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error)
}

func (m *MockCharacterSheetRepo) CreateCharacterSheet(ctx context.Context, sheet *model.CharacterSheet) error {
	if m.CreateCharacterSheetFn != nil {
		return m.CreateCharacterSheetFn(ctx, sheet)
	}
	return nil
}

func (m *MockCharacterSheetRepo) ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error) {
	if m.ExistsCharacterWithNickFn != nil {
		return m.ExistsCharacterWithNickFn(ctx, nick)
	}
	return false, nil
}

func (m *MockCharacterSheetRepo) CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error) {
	if m.CountCharactersByPlayerUUIDFn != nil {
		return m.CountCharactersByPlayerUUIDFn(ctx, playerUUID)
	}
	return 0, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetPlayerUUID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if m.GetCharacterSheetPlayerUUIDFn != nil {
		return m.GetCharacterSheetPlayerUUIDFn(ctx, id)
	}
	return uuid.Nil, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetByUUID(ctx context.Context, id string) (*model.CharacterSheet, error) {
	if m.GetCharacterSheetByUUIDFn != nil {
		return m.GetCharacterSheetByUUIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {
	if m.ListCharacterSheetsByPlayerUUIDFn != nil {
		return m.ListCharacterSheetsByPlayerUUIDFn(ctx, playerUUID)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) UpdateNenHexagonValue(ctx context.Context, id string, val int) error {
	if m.UpdateNenHexagonValueFn != nil {
		return m.UpdateNenHexagonValueFn(ctx, id, val)
	}
	return nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetRelationshipUUIDs(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
	if m.GetCharacterSheetRelationshipUUIDsFn != nil {
		return m.GetCharacterSheetRelationshipUUIDsFn(ctx, id)
	}
	return model.CharacterSheetRelationshipUUIDs{}, nil
}
