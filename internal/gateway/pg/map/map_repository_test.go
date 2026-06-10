//go:build integration

package pgmap_test

import (
	"context"
	"errors"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/google/uuid"
)

func TestMapRepository_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_cag", "master_cag@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign CAG")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m := entity.NewTacticalMap(campaignID, "Forest", "A dark forest")
	if err := repo.CreateMap(ctx, m); err != nil {
		t.Fatalf("CreateMap: %v", err)
	}

	got, err := repo.GetMap(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if got.Name != "Forest" {
		t.Errorf("expected name Forest, got %s", got.Name)
	}
	if got.Grid.Cols != 25 {
		t.Errorf("expected cols 25, got %d", got.Grid.Cols)
	}
}

func TestMapRepository_ListByCampaign(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_lbc", "master_lbc@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign LBC")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m1 := entity.NewTacticalMap(campaignID, "Map A", "")
	m2 := entity.NewTacticalMap(campaignID, "Map B", "")
	_ = repo.CreateMap(ctx, m1)
	_ = repo.CreateMap(ctx, m2)

	maps, err := repo.ListMapsByCampaign(ctx, campaignID)
	if err != nil {
		t.Fatalf("ListMapsByCampaign: %v", err)
	}
	if len(maps) != 2 {
		t.Errorf("expected 2 maps, got %d", len(maps))
	}
}

func TestMapRepository_Update(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_upd", "master_upd@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign UPD")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m := entity.NewTacticalMap(campaignID, "Old Name", "")
	_ = repo.CreateMap(ctx, m)

	m.Name = "New Name"
	if err := repo.UpdateMap(ctx, m); err != nil {
		t.Fatalf("UpdateMap: %v", err)
	}

	got, err := repo.GetMap(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMap after update: %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("expected New Name, got %s", got.Name)
	}
}

func TestMapRepository_Delete(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_del", "master_del@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign DEL")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m := entity.NewTacticalMap(campaignID, "Temp", "")
	_ = repo.CreateMap(ctx, m)

	if err := repo.DeleteMap(ctx, m.ID); err != nil {
		t.Fatalf("DeleteMap: %v", err)
	}

	_, err = repo.GetMap(ctx, m.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
	if !errors.Is(err, pgmap.ErrMapNotFound) {
		t.Errorf("expected ErrMapNotFound, got %v", err)
	}
}

func TestMapRepository_GetMap_NotFound(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	_, err := repo.GetMap(ctx, uuid.New())
	if err == nil {
		t.Error("expected ErrMapNotFound")
	}
	if !errors.Is(err, pgmap.ErrMapNotFound) {
		t.Errorf("expected ErrMapNotFound, got %v", err)
	}
}

func TestMapRepository_WallsRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := pgmap.NewRepository(pool)

	masterStr := pgtest.InsertTestUser(t, pool, "master_wrt", "master_wrt@hunter.com", "pass")
	campaignStr := pgtest.InsertTestCampaign(t, pool, masterStr, "Test Campaign WRT")
	campaignID, err := uuid.Parse(campaignStr)
	if err != nil {
		t.Fatalf("parse campaign uuid: %v", err)
	}

	m := entity.NewTacticalMap(campaignID, "Walled Room", "")
	if err := repo.CreateMap(ctx, m); err != nil {
		t.Fatalf("CreateMap: %v", err)
	}

	doorSubtype := entity.DoorSubtypeBasic
	m.Walls = []entity.WallSegment{
		{
			ID:         "wall-uuid-0001",
			P1:         [2]float64{0, 0},
			P2:         [2]float64{64, 0},
			WallType:   entity.WallTypeWall,
			Material:   entity.WallMaterialStone,
			Move:       true,
			Sense:      entity.SenseFull,
			Direction:  entity.WallDirectionBoth,
			HP:         100,
			MaxHP:      100,
			Resistance: 5,
		},
		{
			ID:          "wall-uuid-0002",
			P1:          [2]float64{64, 0},
			P2:          [2]float64{64, 64},
			WallType:    entity.WallTypeDoor,
			Material:    entity.WallMaterialWood,
			DoorSubtype: &doorSubtype,
			Move:        true,
			Sense:       entity.SenseFull,
			Direction:   entity.WallDirectionBoth,
			Open:        false,
			Locked:      true,
			HP:          40,
			MaxHP:       40,
			Resistance:  2,
		},
	}
	if err := repo.UpdateMap(ctx, m); err != nil {
		t.Fatalf("UpdateMap with walls: %v", err)
	}

	got, err := repo.GetMap(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMap: %v", err)
	}
	if len(got.Walls) != 2 {
		t.Fatalf("expected 2 walls, got %d", len(got.Walls))
	}
	if got.Walls[0].WallType != entity.WallTypeWall {
		t.Errorf("expected WallTypeWall, got %s", got.Walls[0].WallType)
	}
	if got.Walls[1].WallType != entity.WallTypeDoor {
		t.Errorf("expected WallTypeDoor, got %s", got.Walls[1].WallType)
	}
	if got.Walls[1].DoorSubtype == nil || *got.Walls[1].DoorSubtype != entity.DoorSubtypeBasic {
		t.Errorf("expected DoorSubtypeBasic, got %v", got.Walls[1].DoorSubtype)
	}
	if !got.Walls[1].Locked {
		t.Error("expected wall[1].Locked = true")
	}
}
