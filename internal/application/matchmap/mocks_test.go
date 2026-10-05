package matchmapuc_test

import (
	"context"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

type mockRepository struct {
	attachMapFn   func(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error)
	getMatchMapFn func(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
	detachMapFn   func(ctx context.Context, matchUUID uuid.UUID) error
}

func (m *mockRepository) AttachMap(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error) {
	return m.attachMapFn(ctx, matchUUID, mapUUID)
}

func (m *mockRepository) GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	return m.getMatchMapFn(ctx, matchUUID)
}

func (m *mockRepository) DetachMap(ctx context.Context, matchUUID uuid.UUID) error {
	return m.detachMapFn(ctx, matchUUID)
}

type mockMatchRepository struct {
	getMatchInfoFn func(ctx context.Context, matchUUID uuid.UUID) (*matchmapuc.MatchInfo, error)
}

func (m *mockMatchRepository) GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
	return m.getMatchInfoFn(ctx, matchUUID)
}

type mockMatchBoardRepository struct {
	getFn    func(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error)
	copyFn   func(ctx context.Context, src, dst uuid.UUID) error
	deleteFn func(ctx context.Context, matchUUID uuid.UUID) error

	copyCalled   bool
	copySrc      uuid.UUID
	copyDst      uuid.UUID
	deleteCalled bool
	deleteUUID   uuid.UUID
}

func (m *mockMatchBoardRepository) Get(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error) {
	return m.getFn(ctx, matchUUID)
}

func (m *mockMatchBoardRepository) Copy(ctx context.Context, src, dst uuid.UUID) error {
	m.copyCalled = true
	m.copySrc = src
	m.copyDst = dst
	if m.copyFn != nil {
		return m.copyFn(ctx, src, dst)
	}
	return nil
}

func (m *mockMatchBoardRepository) Delete(ctx context.Context, matchUUID uuid.UUID) error {
	m.deleteCalled = true
	m.deleteUUID = matchUUID
	if m.deleteFn != nil {
		return m.deleteFn(ctx, matchUUID)
	}
	return nil
}
