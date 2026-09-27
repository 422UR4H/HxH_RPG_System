//go:build integration

package pgmatchboard_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	pgmatchboard "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchboard"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestMatchBoard(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgmatchboard.NewRepository(pool)
	ctx := context.Background()

	setupMatchAndMap := func(t *testing.T) (matchUUID, mapUUID uuid.UUID) {
		t.Helper()
		masterUUID := pgtest.InsertTestUser(t, pool, "gm-board", "gm-board@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Board Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Board Match")
		mapStr := pgtest.InsertTestMap(t, pool, campaignUUID, "Board Map")
		return uuid.MustParse(matchStr), uuid.MustParse(mapStr)
	}

	t.Run("get without row returns nil", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchUUID, _ := setupMatchAndMap(t)

		got, err := repo.Get(ctx, matchUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil board without a row, got %+v", got)
		}
	})

	t.Run("save then get round-trips pieces walls grid and nil bg", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchUUID, mapUUID := setupMatchAndMap(t)

		board := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Bg:        nil,
			Pieces: []mapentity.Piece{
				{
					ID:          "piece-1",
					CharacterID: "char-1",
					Coord: mapentity.PieceCoord{
						Slot: map[string]any{"kind": "square", "col": 3.0, "row": 4.0},
						Z:    0,
					},
					Visible: true,
				},
			},
			Walls: []mapentity.WallSegment{
				{
					ID:       "wall-1",
					P1:       [2]float64{0, 0},
					P2:       [2]float64{10, 0},
					WallType: mapentity.WallTypeWall,
					Material: mapentity.WallMaterialStone,
					Sense:    mapentity.SenseFull,
					HP:       10,
					MaxHP:    10,
				},
			},
		}

		if err := repo.Save(ctx, board); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := repo.Get(ctx, matchUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got == nil {
			t.Fatalf("expected a board after Save, got nil")
		}
		if got.MatchUUID != matchUUID {
			t.Fatalf("MatchUUID: got %s want %s", got.MatchUUID, matchUUID)
		}
		if got.MapUUID != mapUUID {
			t.Fatalf("MapUUID: got %s want %s", got.MapUUID, mapUUID)
		}
		if got.Bg != nil {
			t.Fatalf("expected nil Bg, got %+v", got.Bg)
		}
		if !reflect.DeepEqual(got.Grid, board.Grid) {
			t.Fatalf("Grid: got %+v want %+v", got.Grid, board.Grid)
		}
		if !reflect.DeepEqual(got.Pieces, board.Pieces) {
			t.Fatalf("Pieces: got %+v want %+v", got.Pieces, board.Pieces)
		}
		if !reflect.DeepEqual(got.Walls, board.Walls) {
			t.Fatalf("Walls: got %+v want %+v", got.Walls, board.Walls)
		}
	})

	t.Run("save twice upserts", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchUUID, mapUUID := setupMatchAndMap(t)

		board := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces: []mapentity.Piece{
				{ID: "piece-1", CharacterID: "char-1", Coord: mapentity.PieceCoord{
					Slot: map[string]any{"kind": "square", "col": 1.0, "row": 1.0}, Z: 0,
				}, Visible: true},
			},
			Walls: []mapentity.WallSegment{},
		}
		if err := repo.Save(ctx, board); err != nil {
			t.Fatalf("first Save: %v", err)
		}

		board2 := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces: []mapentity.Piece{
				{ID: "piece-1", CharacterID: "char-1", Coord: mapentity.PieceCoord{
					Slot: map[string]any{"kind": "square", "col": 9.0, "row": 9.0}, Z: 0,
				}, Visible: true},
			},
			Walls: []mapentity.WallSegment{},
		}
		if err := repo.Save(ctx, board2); err != nil {
			t.Fatalf("second Save: %v", err)
		}

		got, err := repo.Get(ctx, matchUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got == nil {
			t.Fatalf("expected a board, got nil")
		}
		if len(got.Pieces) != 1 {
			t.Fatalf("expected exactly 1 board row (upsert), got %d pieces snapshot mismatch", len(got.Pieces))
		}
		if !reflect.DeepEqual(got.Pieces, board2.Pieces) {
			t.Fatalf("expected the SECOND save to win: got %+v want %+v", got.Pieces, board2.Pieces)
		}
	})

	t.Run("delete removes", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchUUID, mapUUID := setupMatchAndMap(t)

		board := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces:    []mapentity.Piece{},
			Walls:     []mapentity.WallSegment{},
		}
		if err := repo.Save(ctx, board); err != nil {
			t.Fatalf("Save: %v", err)
		}

		if err := repo.Delete(ctx, matchUUID); err != nil {
			t.Fatalf("Delete: %v", err)
		}

		got, err := repo.Get(ctx, matchUUID)
		if err != nil {
			t.Fatalf("Get after delete: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil board after Delete, got %+v", got)
		}
	})
}
