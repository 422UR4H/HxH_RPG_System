package matchboarduc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	matchmapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
)

type fakeMatchMapRepo struct {
	mm  *matchmapentity.MatchMap
	err error
}

func (f *fakeMatchMapRepo) GetMatchMap(_ context.Context, _ uuid.UUID) (*matchmapentity.MatchMap, error) {
	return f.mm, f.err
}

type fakeBoardRepo struct {
	board *matchboard.Board
	err   error
	calls int
}

func (f *fakeBoardRepo) Get(_ context.Context, _ uuid.UUID) (*matchboard.Board, error) {
	f.calls++
	return f.board, f.err
}

type fakeMapRepo struct {
	m     *mapentity.TacticalMap
	err   error
	calls int
}

func (f *fakeMapRepo) GetMap(_ context.Context, _ uuid.UUID) (*mapentity.TacticalMap, error) {
	f.calls++
	return f.m, f.err
}

func TestLoadMatchBoardUC_Load(t *testing.T) {
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	t.Run("no map attached returns nil, nil", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{err: pgmatchmap.ErrMatchMapNotFound}
		boardRepo := &fakeBoardRepo{}
		mapRepo := &fakeMapRepo{}
		uc := matchboarduc.NewLoadMatchBoardUC(boardRepo, matchMapRepo, mapRepo)

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got != nil {
			t.Fatalf("Load = %+v, want nil (no map attached)", got)
		}
		if boardRepo.calls != 0 {
			t.Fatalf("board repo was called %d time(s), want 0 — nothing to look up without an "+
				"attachment", boardRepo.calls)
		}
		if mapRepo.calls != 0 {
			t.Fatalf("map repo was called %d time(s), want 0", mapRepo.calls)
		}
	})

	t.Run("attached and a match_boards row exists returns the row without reading the map", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		saved := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   mapUUID,
			Grid:      mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 5, Rows: 5, CellSize: 32},
			Bg:        &mapentity.BgImage{URL: "own"},
			Pieces:    []mapentity.Piece{{ID: "p1"}},
			Walls:     []mapentity.WallSegment{{ID: "w1"}},
		}
		boardRepo := &fakeBoardRepo{board: saved}
		mapRepo := &fakeMapRepo{m: mapentity.NewTacticalMap(uuid.New(), "should not be read", "")}
		uc := matchboarduc.NewLoadMatchBoardUC(boardRepo, matchMapRepo, mapRepo)

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got != saved {
			t.Fatalf("Load = %+v, want the exact saved row %+v", got, saved)
		}
		if mapRepo.calls != 0 {
			t.Fatalf("map repo was called %d time(s), want 0 — a saved row must win over the map",
				mapRepo.calls)
		}
	})

	t.Run("attached and no row falls back to a fresh snapshot of the map", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		boardRepo := &fakeBoardRepo{board: nil}
		m := mapentity.NewTacticalMap(uuid.New(), "Dungeon", "")
		m.ID = mapUUID
		m.Grid = mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 64}
		m.Pieces = []mapentity.Piece{{ID: "p1", CharacterID: "c1"}}
		m.Walls = []mapentity.WallSegment{{ID: "w1"}}
		bg := &mapentity.BgImage{URL: "map-bg", X: 1, Y: 2, Width: 3, Height: 4, Rotation: 5, Opacity: 0.5}
		want := *bg
		m.Bg = bg
		mapRepo := &fakeMapRepo{m: m}
		uc := matchboarduc.NewLoadMatchBoardUC(boardRepo, matchMapRepo, mapRepo)

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got == nil {
			t.Fatal("Load = nil, want a fresh snapshot of the attached map")
		}
		if got.MatchUUID != matchUUID {
			t.Fatalf("MatchUUID = %v, want %v", got.MatchUUID, matchUUID)
		}
		if got.MapUUID != mapUUID {
			t.Fatalf("MapUUID = %v, want the attached map's %v", got.MapUUID, mapUUID)
		}
		if got.Grid != m.Grid {
			t.Fatalf("Grid = %+v, want the map's own %+v", got.Grid, m.Grid)
		}
		if len(got.Pieces) != 1 || got.Pieces[0].ID != "p1" {
			t.Fatalf("Pieces = %+v, want the map's own", got.Pieces)
		}
		if len(got.Walls) != 1 || got.Walls[0].ID != "w1" {
			t.Fatalf("Walls = %+v, want the map's own", got.Walls)
		}
		// The background is part of the portrait: copied by value, so editing the map later
		// does not move the board's background.
		if got.Bg == nil || *got.Bg != want {
			t.Fatalf("Bg = %+v, want a copy of the map's %+v", got.Bg, want)
		}
		if got.Bg == bg {
			t.Fatal("Bg shares the map's pointer, want a value copy")
		}
		m.Bg.URL = "changed after"
		if got.Bg.URL == "changed after" {
			t.Fatal("changing the map's Bg changed the returned board")
		}
	})

	// F1: detach + attach another map deletes nothing (AttachMatchMapUC only clears the board
	// when the PREVIOUS attachment was a different map, and a detach leaves no previous one),
	// so a row saved for the old map can still be sitting there when the new one is attached.
	// It is a portrait of a map that is no longer this match's — it must read as absent.
	t.Run("a saved row for a map other than the attached one is ignored for the map snapshot", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		stale := &matchboard.Board{
			MatchUUID: matchUUID,
			MapUUID:   uuid.New(), // the OLD map
			Pieces:    []mapentity.Piece{{ID: "stale"}},
		}
		boardRepo := &fakeBoardRepo{board: stale}
		m := mapentity.NewTacticalMap(uuid.New(), "New map", "")
		m.ID = mapUUID
		m.Pieces = []mapentity.Piece{{ID: "fresh"}}
		mapRepo := &fakeMapRepo{m: m}
		uc := matchboarduc.NewLoadMatchBoardUC(boardRepo, matchMapRepo, mapRepo)

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got == nil {
			t.Fatal("Load = nil, want a fresh snapshot of the attached map")
		}
		if got.MapUUID != mapUUID {
			t.Fatalf("MapUUID = %v, want the ATTACHED map %v — the stale row of another map won",
				got.MapUUID, mapUUID)
		}
		if len(got.Pieces) != 1 || got.Pieces[0].ID != "fresh" {
			t.Fatalf("Pieces = %+v, want the attached map's own", got.Pieces)
		}
		if mapRepo.calls != 1 {
			t.Fatalf("map repo was called %d time(s), want 1", mapRepo.calls)
		}
	})

	t.Run("fresh snapshot of a map without background has nil Bg", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		m := mapentity.NewTacticalMap(uuid.New(), "Plain", "")
		m.ID = mapUUID
		uc := matchboarduc.NewLoadMatchBoardUC(&fakeBoardRepo{}, matchMapRepo, &fakeMapRepo{m: m})

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got.Bg != nil {
			t.Fatalf("Bg = %+v, want nil", got.Bg)
		}
	})

	// Legacy rows were saved with bg NULL ("inherit"). They resolve to the map's current
	// background at read time; the next board save writes that value and freezes the row.
	t.Run("saved row with nil Bg resolves to the map's current background", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		saved := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID, Pieces: []mapentity.Piece{{ID: "p1"}}}
		m := mapentity.NewTacticalMap(uuid.New(), "Dungeon", "")
		m.ID = mapUUID
		m.Bg = &mapentity.BgImage{URL: "map-bg", Width: 10}
		mapRepo := &fakeMapRepo{m: m}
		uc := matchboarduc.NewLoadMatchBoardUC(&fakeBoardRepo{board: saved}, matchMapRepo, mapRepo)

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got.Bg == nil || *got.Bg != *m.Bg {
			t.Fatalf("Bg = %+v, want the map's %+v", got.Bg, m.Bg)
		}
		if got.Bg == m.Bg {
			t.Fatal("Bg shares the map's pointer, want a value copy")
		}
		if len(got.Pieces) != 1 || got.Pieces[0].ID != "p1" {
			t.Fatalf("Pieces = %+v, want the saved row's own", got.Pieces)
		}
	})

	t.Run("saved row with nil Bg and a map without background stays nil", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{
			mm: &matchmapentity.MatchMap{MatchUUID: matchUUID.String(), MapUUID: mapUUID.String()},
		}
		saved := &matchboard.Board{MatchUUID: matchUUID, MapUUID: mapUUID}
		m := mapentity.NewTacticalMap(uuid.New(), "Plain", "")
		m.ID = mapUUID
		uc := matchboarduc.NewLoadMatchBoardUC(&fakeBoardRepo{board: saved}, matchMapRepo, &fakeMapRepo{m: m})

		got, err := uc.Load(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Load returned an error: %v", err)
		}
		if got.Bg != nil {
			t.Fatalf("Bg = %+v, want nil", got.Bg)
		}
	})

	t.Run("match map repository error other than not-found propagates", func(t *testing.T) {
		matchMapRepo := &fakeMatchMapRepo{err: context.DeadlineExceeded}
		uc := matchboarduc.NewLoadMatchBoardUC(&fakeBoardRepo{}, matchMapRepo, &fakeMapRepo{})

		if _, err := uc.Load(context.Background(), matchUUID); err == nil {
			t.Fatal("Load returned no error for a real match-map repository failure")
		}
	})
}
