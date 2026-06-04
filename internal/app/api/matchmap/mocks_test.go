package matchmapapi_test

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/google/uuid"
)

type mockAttachMatchMap struct {
	fn func(ctx context.Context, input *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error)
}

func (m *mockAttachMatchMap) Attach(ctx context.Context, input *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
	return m.fn(ctx, input)
}

type mockGetMatchMap struct {
	fn func(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
}

func (m *mockGetMatchMap) Get(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	return m.fn(ctx, matchUUID)
}

type mockDetachMatchMap struct {
	fn func(ctx context.Context, input *matchmapuc.DetachMatchMapInput) error
}

func (m *mockDetachMatchMap) Detach(ctx context.Context, input *matchmapuc.DetachMatchMapInput) error {
	return m.fn(ctx, input)
}
