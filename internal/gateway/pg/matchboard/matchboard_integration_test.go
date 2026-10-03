//go:build integration

package pgmatchboard_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/fog"
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

// TestMatchBoardCopy covers B16 (spec §4.3): a match can inherit the board — and the fog —
// of another match that ended with it, as one INSERT…SELECT per table inside one
// transaction (pgmatchboard.Repository.Copy, which reaches into fog.CopyMatch for the
// player_memories half — see copy.go).
func TestMatchBoardCopy(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgmatchboard.NewRepository(pool)
	fogRepo := fog.NewPlayerMemoryRepository(pool)
	ctx := context.Background()

	t.Run("copies board and player memories from src to dst, replacing dst's own", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm-copy", "gm-copy@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Copy Campaign")
		srcMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Src Match"))
		dstMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Dst Match"))
		mapUUID := uuid.MustParse(pgtest.InsertTestMap(t, pool, campaignUUID, "Shared Map"))

		srcBoard := &matchboard.Board{
			MatchUUID: srcMatch,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces: []mapentity.Piece{
				{
					ID:          "piece-src",
					CharacterID: "char-src",
					Coord: mapentity.PieceCoord{
						Slot: map[string]any{"kind": "square", "col": 5.0, "row": 6.0},
						Z:    0,
					},
					Visible: true,
				},
			},
			Walls: []mapentity.WallSegment{
				{
					ID:       "wall-src",
					P1:       [2]float64{0, 0},
					P2:       [2]float64{5, 0},
					WallType: mapentity.WallTypeWall,
					Material: mapentity.WallMaterialStone,
					Sense:    mapentity.SenseFull,
					HP:       7,
					MaxHP:    10,
				},
			},
		}
		if err := repo.Save(ctx, srcBoard); err != nil {
			t.Fatalf("save src board: %v", err)
		}

		// dst already has its OWN board (and, implicitly, could have memories) — Copy must
		// replace it entirely, not merge.
		dstOldBoard := &matchboard.Board{
			MatchUUID: dstMatch,
			MapUUID:   mapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces:    []mapentity.Piece{{ID: "piece-old-dst", CharacterID: "char-old", Coord: mapentity.PieceCoord{Slot: map[string]any{"kind": "square", "col": 1.0, "row": 1.0}, Z: 0}, Visible: true}},
			Walls:     []mapentity.WallSegment{},
		}
		if err := repo.Save(ctx, dstOldBoard); err != nil {
			t.Fatalf("save dst old board: %v", err)
		}

		player1, player2 := uuid.New(), uuid.New()
		mem1 := fogentity.PlayerMemory{
			PlayerID: player1, MatchID: srcMatch, MapID: mapUUID,
			Seen: map[fogentity.FeatureRef]struct{}{{Kind: fogentity.FeatureWall, ID: "wall-src"}: {}},
		}
		mem2 := fogentity.PlayerMemory{
			PlayerID: player2, MatchID: srcMatch, MapID: mapUUID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-src"}:   {},
				{Kind: fogentity.FeatureWall, ID: "wall-other"}: {},
			},
		}
		if err := fogRepo.Upsert(ctx, mem1); err != nil {
			t.Fatalf("upsert mem1: %v", err)
		}
		if err := fogRepo.Upsert(ctx, mem2); err != nil {
			t.Fatalf("upsert mem2: %v", err)
		}

		if err := repo.Copy(ctx, srcMatch, dstMatch); err != nil {
			t.Fatalf("Copy: %v", err)
		}

		got, err := repo.Get(ctx, dstMatch)
		if err != nil {
			t.Fatalf("Get dst after Copy: %v", err)
		}
		if got == nil {
			t.Fatalf("expected a board on dst after Copy, got nil")
		}
		if got.MatchUUID != dstMatch {
			t.Fatalf("MatchUUID: got %s want %s", got.MatchUUID, dstMatch)
		}
		if got.MapUUID != mapUUID {
			t.Fatalf("MapUUID: got %s want %s", got.MapUUID, mapUUID)
		}
		if !reflect.DeepEqual(got.Pieces, srcBoard.Pieces) {
			t.Fatalf("Pieces: got %+v want src's %+v (replaced, not merged)", got.Pieces, srcBoard.Pieces)
		}
		if !reflect.DeepEqual(got.Walls, srcBoard.Walls) {
			t.Fatalf("Walls: got %+v want src's %+v", got.Walls, srcBoard.Walls)
		}

		// src's own board must be untouched.
		srcGot, err := repo.Get(ctx, srcMatch)
		if err != nil {
			t.Fatalf("Get src after Copy: %v", err)
		}
		if srcGot == nil || !reflect.DeepEqual(srcGot.Pieces, srcBoard.Pieces) {
			t.Fatalf("expected src board untouched by Copy, got %+v", srcGot)
		}

		gotMems, err := fogRepo.FindByMatchMap(ctx, dstMatch, mapUUID)
		if err != nil {
			t.Fatalf("FindByMatchMap dst after Copy: %v", err)
		}
		if len(gotMems) != 2 {
			t.Fatalf("expected 2 player memories copied to dst, got %d", len(gotMems))
		}
		byPlayer := map[uuid.UUID]fogentity.PlayerMemory{}
		for _, m := range gotMems {
			if m.MatchID != dstMatch {
				t.Fatalf("expected copied memory's MatchID to be dst, got %s", m.MatchID)
			}
			byPlayer[m.PlayerID] = m
		}
		g1, ok := byPlayer[player1]
		if !ok || !g1.Has(fogentity.FeatureWall, "wall-src") || len(g1.Seen) != 1 {
			t.Fatalf("player1 memory not copied correctly: %+v", g1)
		}
		g2, ok := byPlayer[player2]
		if !ok || !g2.Has(fogentity.FeatureWall, "wall-src") || !g2.Has(fogentity.FeatureWall, "wall-other") || len(g2.Seen) != 2 {
			t.Fatalf("player2 memory not copied correctly: %+v", g2)
		}

		// src's own memories must be untouched.
		srcMems, err := fogRepo.FindByMatchMap(ctx, srcMatch, mapUUID)
		if err != nil {
			t.Fatalf("FindByMatchMap src after Copy: %v", err)
		}
		if len(srcMems) != 2 {
			t.Fatalf("expected src's own 2 memories untouched, got %d", len(srcMems))
		}
	})

	t.Run("errors when source has no board row", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm-copy-2", "gm-copy-2@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Copy Campaign 2")
		srcMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Src Match No Board"))
		dstMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Dst Match No Board"))

		if err := repo.Copy(ctx, srcMatch, dstMatch); err == nil {
			t.Fatalf("expected an error copying from a match with no board")
		}
	})

	// Fix round 1 (review finding): src is a PAST match and may hold player_memories rows
	// for some OTHER map it was once attached to (e.g. an earlier board, since detached).
	// Copy must only bring over the memories of the map the source's CURRENT board is on —
	// a stray memory on another map must NOT ride along into dst.
	t.Run("does not copy source's player memories from a different map", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm-copy-3", "gm-copy-3@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Copy Campaign 3")
		srcMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Src Match"))
		dstMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Dst Match"))
		currentMapUUID := uuid.MustParse(pgtest.InsertTestMap(t, pool, campaignUUID, "Current Map"))
		otherMapUUID := uuid.MustParse(pgtest.InsertTestMap(t, pool, campaignUUID, "Other (Stale) Map"))

		// src's board is on currentMapUUID.
		srcBoard := &matchboard.Board{
			MatchUUID: srcMatch,
			MapUUID:   currentMapUUID,
			Grid:      mapentity.DefaultGrid(),
			Pieces:    []mapentity.Piece{},
			Walls:     []mapentity.WallSegment{},
		}
		if err := repo.Save(ctx, srcBoard); err != nil {
			t.Fatalf("save src board: %v", err)
		}

		player := uuid.New()
		// A memory for the CURRENT map — must be copied.
		currentMem := fogentity.PlayerMemory{
			PlayerID: player, MatchID: srcMatch, MapID: currentMapUUID,
			Seen: map[fogentity.FeatureRef]struct{}{{Kind: fogentity.FeatureWall, ID: "wall-current"}: {}},
		}
		// A stray memory for a map src is no longer on — must NOT be copied.
		staleMem := fogentity.PlayerMemory{
			PlayerID: player, MatchID: srcMatch, MapID: otherMapUUID,
			Seen: map[fogentity.FeatureRef]struct{}{{Kind: fogentity.FeatureWall, ID: "wall-stale"}: {}},
		}
		if err := fogRepo.Upsert(ctx, currentMem); err != nil {
			t.Fatalf("upsert currentMem: %v", err)
		}
		if err := fogRepo.Upsert(ctx, staleMem); err != nil {
			t.Fatalf("upsert staleMem: %v", err)
		}

		if err := repo.Copy(ctx, srcMatch, dstMatch); err != nil {
			t.Fatalf("Copy: %v", err)
		}

		gotCurrent, err := fogRepo.FindByMatchMap(ctx, dstMatch, currentMapUUID)
		if err != nil {
			t.Fatalf("FindByMatchMap dst/currentMap: %v", err)
		}
		if len(gotCurrent) != 1 || !gotCurrent[0].Has(fogentity.FeatureWall, "wall-current") {
			t.Fatalf("expected the current-map memory to be copied, got %+v", gotCurrent)
		}

		gotOther, err := fogRepo.FindByMatchMap(ctx, dstMatch, otherMapUUID)
		if err != nil {
			t.Fatalf("FindByMatchMap dst/otherMap: %v", err)
		}
		if len(gotOther) != 0 {
			t.Fatalf("expected the OTHER map's stray memory NOT to be copied, got %+v", gotOther)
		}
	})
}
