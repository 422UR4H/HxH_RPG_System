// internal/app/api/map/mocks_test.go
package mapapi_test

import (
	"context"

	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type mockCreateMap struct {
	result *entity.TacticalMap
	err    error
}

func (m *mockCreateMap) CreateMap(_ context.Context, _ *mapuc.CreateMapInput) (*entity.TacticalMap, error) {
	return m.result, m.err
}
