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

	// received is the input the handler actually built from the request body — set on
	// every call so a test can assert on it directly, in addition to (or instead of)
	// asserting on the HTTP response.
	received *mapuc.CreateMapInput
}

func (m *mockCreateMap) CreateMap(_ context.Context, input *mapuc.CreateMapInput) (*entity.TacticalMap, error) {
	m.received = input
	if m.err != nil {
		return nil, m.err
	}
	// Mirror what the real CreateMapUC does with Grid/Bg/Pieces, so a bug in the
	// request->entity converters (toEntityGridShape/toEntityBgImage/toEntityPiece)
	// shows up in the HTTP response the test asserts against, not just in `received`.
	res := *m.result
	if input.Grid != nil {
		res.Grid = *input.Grid
	}
	res.Bg = input.Bg
	if len(input.Pieces) > 0 {
		res.Pieces = input.Pieces
	}
	return &res, nil
}

// mockUpdateMap satisfies mapuc.IUpdateMap and records the input the handler built from
// the request body, since UpdateMapHandler returns 204 with no body — the only way to
// assert the request->entity converters (toEntityGridShape/toEntityBgImage/toEntityPiece/
// toEntityWallSegment) did their job is to inspect what reached the use case.
type mockUpdateMap struct {
	err      error
	received *mapuc.UpdateMapInput
}

func (m *mockUpdateMap) UpdateMap(_ context.Context, input *mapuc.UpdateMapInput) error {
	m.received = input
	return m.err
}
