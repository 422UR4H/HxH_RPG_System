// internal/app/api/map/mocks_test.go
package mapapi_test

import (
	"context"

	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type mockCreateMap struct {
	result *entity.TacticalMap
	err    error
}

func (m *mockCreateMap) CreateMap(_ context.Context, _ *mapuc.CreateMapInput) (*entity.TacticalMap, error) {
	return m.result, m.err
}

type mockListMaps struct {
	result []*entity.TacticalMap
	err    error
}

func (m *mockListMaps) ListMaps(_ context.Context, _, _ uuid.UUID) ([]*entity.TacticalMap, error) {
	return m.result, m.err
}
