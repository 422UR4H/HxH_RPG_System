package matchsession_test

import (
	"reflect"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestVisibilityFrom_MatchesRecomputeForSinglePiece(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &playerUUID},
	}
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1}
	wall := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{100, 0}, P2: [2]float64{100, 100},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	s.SyncMapState([]mapentity.WallSegment{wall}, grid)
	s.SyncPlayerMemories(nil, fog.FogModeExplored)

	pos := service.Point2D{X: 50, Y: 50}
	src := &fakePieceSource{}
	src.setPosition(playerUUID, pos)
	s.SetPieceSource(src)

	// Pure: called before any RecomputeVisibility, so the cache is still nil.
	got := s.VisibilityFrom(pos)
	if s.GetVisibility(playerUUID) != nil {
		t.Error("VisibilityFrom must not write the visibility cache")
	}
	if mem, ok := s.GetPlayerMemory(playerUUID); ok && len(mem.Seen) != 0 {
		t.Errorf("VisibilityFrom must not write memory, got %d entries", len(mem.Seen))
	}

	polys, err := s.RecomputeVisibility(playerUUID)
	if err != nil || len(polys) != 1 {
		t.Fatalf("unexpected recompute result: %v, %d polys", err, len(polys))
	}
	if !reflect.DeepEqual(got, polys[0]) {
		t.Errorf("VisibilityFrom differs from RecomputeVisibility:\n got %+v\nwant %+v", got, polys[0])
	}
}
