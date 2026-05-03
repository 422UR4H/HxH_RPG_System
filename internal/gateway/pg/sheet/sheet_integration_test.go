//go:build integration

package sheet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

func buildTestSheet(playerUUID *uuid.UUID) *model.CharacterSheet {
	now := time.Now()
	birthday := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	sheetUUID := uuid.New()
	profileUUID := uuid.New()
	return &model.CharacterSheet{
		UUID:         sheetUUID,
		PlayerUUID:   playerUUID,
		CategoryName: "Reinforcement",
		CurrHexValue: nil,
		CreatedAt:    now,
		UpdatedAt:    now,
		Profile: model.CharacterProfile{
			UUID:             profileUUID,
			NickName:         "TestChar",
			FullName:         "Test Character",
			Alignment:        "Neutral",
			CharacterClass:   "Swordsman",
			Description:      "A test character",
			BriefDescription: "Test",
			Birthday:         &birthday,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		Proficiencies: []model.Proficiency{
			{Weapon: "Sword", Exp: 100},
		},
	}
}

func TestCreateCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	t.Run("happy path player-owned with proficiencies", func(t *testing.T) {
		s := buildTestSheet(&playerUUID)
		err := repo.CreateCharacterSheet(ctx, s)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.ID == 0 {
			t.Fatal("expected sheet ID to be set after create")
		}
	})

	t.Run("happy path master-owned requires player_uuid nil", func(t *testing.T) {
		// NOTE: CreateCharacterSheet only inserts player_uuid, not master_uuid.
		// The XOR constraint (chk_exclusive_owner) requires exactly one of player_uuid/master_uuid.
		// Creating master-owned sheets is not supported by this repository method.
		s := &model.CharacterSheet{
			UUID:         uuid.New(),
			PlayerUUID:   nil,
			MasterUUID:   &playerUUID,
			CategoryName: "Reinforcement",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			Profile: model.CharacterProfile{
				UUID:             uuid.New(),
				NickName:         "MasterNPC",
				FullName:         "Master NPC Character",
				Alignment:        "Evil",
				CharacterClass:   "Swordsman",
				Description:      "An NPC",
				BriefDescription: "NPC",
				Birthday:         func() *time.Time { t := time.Date(1995, 6, 15, 0, 0, 0, 0, time.UTC); return &t }(),
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
		}
		err := repo.CreateCharacterSheet(ctx, s)
		if err == nil {
			t.Fatal("expected error for master-owned sheet (repo only inserts player_uuid), got nil")
		}
	})
}

func TestGetCharacterSheetByUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	created := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, created)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetCharacterSheetByUUID(ctx, created.UUID.String())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.UUID != created.UUID {
			t.Fatalf("expected UUID %s, got %s", created.UUID, got.UUID)
		}
		if got.Profile.NickName != created.Profile.NickName {
			t.Fatalf("expected nickname %q, got %q", created.Profile.NickName, got.Profile.NickName)
		}
		if got.CategoryName != "Reinforcement" {
			t.Fatalf("expected category Reinforcement, got %q", got.CategoryName)
		}
		if len(got.Proficiencies) != 1 {
			t.Fatalf("expected 1 proficiency, got %d", len(got.Proficiencies))
		}
		if got.Proficiencies[0].Weapon != "Sword" {
			t.Fatalf("expected proficiency weapon Sword, got %q", got.Proficiencies[0].Weapon)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCharacterSheetByUUID(ctx, uuid.New().String())
		if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})
}

func TestGetCharacterSheetPlayerUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	created := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, created)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetCharacterSheetPlayerUUID(ctx, created.UUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != playerUUID {
			t.Fatalf("expected player UUID %s, got %s", playerUUID, got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCharacterSheetPlayerUUID(ctx, uuid.New())
		if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})
}

func TestGetCharacterSheetRelationshipUUIDs(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	created := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, created)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetCharacterSheetRelationshipUUIDs(ctx, created.UUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.PlayerUUID == nil || *got.PlayerUUID != playerUUID {
			t.Fatalf("expected player UUID %s, got %v", playerUUID, got.PlayerUUID)
		}
		if got.CampaignUUID != nil {
			t.Fatalf("expected nil campaign UUID, got %v", got.CampaignUUID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCharacterSheetRelationshipUUIDs(ctx, uuid.New())
		if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})
}

func TestExistsCharacterWithNick(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	created := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, created)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("true", func(t *testing.T) {
		exists, err := repo.ExistsCharacterWithNick(ctx, created.Profile.NickName)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
	})

	t.Run("false", func(t *testing.T) {
		exists, err := repo.ExistsCharacterWithNick(ctx, "NonExistentNick")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
	})
}

func TestCountCharactersByPlayerUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	t.Run("count zero", func(t *testing.T) {
		count, err := repo.CountCharactersByPlayerUUID(ctx, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0, got %d", count)
		}
	})

	t.Run("count greater than zero", func(t *testing.T) {
		s := buildTestSheet(&playerUUID)
		err := repo.CreateCharacterSheet(ctx, s)
		if err != nil {
			t.Fatalf("setup: failed to create sheet: %v", err)
		}

		count, err := repo.CountCharactersByPlayerUUID(ctx, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if count < 1 {
			t.Fatalf("expected count >= 1, got %d", count)
		}
	})
}

func TestListCharacterSheetsByPlayerUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	s := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, s)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("returns list", func(t *testing.T) {
		list, err := repo.ListCharacterSheetsByPlayerUUID(ctx, playerStr)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(list) == 0 {
			t.Fatal("expected at least one sheet in list")
		}
		if list[0].NickName != s.Profile.NickName {
			t.Fatalf("expected nickname %q, got %q", s.Profile.NickName, list[0].NickName)
		}
		if list[0].UUID == uuid.Nil {
			t.Fatal("expected non-nil UUID in summary")
		}
	})
}

func TestUpdateNenHexagonValue(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	s := buildTestSheet(&playerUUID)
	err := repo.CreateCharacterSheet(ctx, s)
	if err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		newVal := 42
		err := repo.UpdateNenHexagonValue(ctx, s.UUID.String(), newVal)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		got, err := repo.GetCharacterSheetByUUID(ctx, s.UUID.String())
		if err != nil {
			t.Fatalf("expected no error fetching sheet, got %v", err)
		}
		if got.CurrHexValue == nil || *got.CurrHexValue != newVal {
			t.Fatalf("expected hex value %d, got %v", newVal, got.CurrHexValue)
		}
	})
}

func TestUpdateStatusBars(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerStr, nil, nil, "Gon")

		health := model.StatusBar{Min: 0, Curr: 17, Max: 20}
		stamina := model.StatusBar{Min: 0, Curr: 0, Max: 0}
		aura := model.StatusBar{Min: 0, Curr: 0, Max: 0}

		err := repo.UpdateStatusBars(ctx, sheetUUID, health, stamina, aura)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var healthMin, healthCurr, healthMax int
		err = pool.QueryRow(ctx,
			`SELECT health_min_pts, health_curr_pts, health_max_pts FROM character_sheets WHERE uuid = $1`,
			sheetUUID,
		).Scan(&healthMin, &healthCurr, &healthMax)
		if err != nil {
			t.Fatalf("failed to fetch sheet: %v", err)
		}
		if healthCurr != 17 {
			t.Errorf("health curr = %d, want 17", healthCurr)
		}
		if healthMax != 20 {
			t.Errorf("health max = %d, want 20", healthMax)
		}
		if healthMin != 0 {
			t.Errorf("health min = %d, want 0", healthMin)
		}
	})

	t.Run("sheet not found is a no-op", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		health := model.StatusBar{Min: 0, Curr: 10, Max: 20}
		stamina := model.StatusBar{Min: 0, Curr: 5, Max: 10}
		aura := model.StatusBar{Min: 0, Curr: 0, Max: 0}

		err := repo.UpdateStatusBars(ctx, uuid.New().String(), health, stamina, aura)
		if err != nil {
			t.Errorf("expected no error for missing sheet, got: %v", err)
		}
	})
}

func TestExistsSheetInCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := sheet.NewRepository(pool)

	t.Run("true when player has a sheet in the campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")

		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")
		if _, err := pool.Exec(ctx,
			`UPDATE character_sheets SET campaign_uuid = $1 WHERE uuid = $2`,
			campaignUUID, sheetUUID,
		); err != nil {
			t.Fatalf("update campaign_uuid: %v", err)
		}

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignUUID),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if !got {
			t.Error("ExistsSheetInCampaign() = false, want true")
		}
	})

	t.Run("false when player has no sheet in the campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignUUID),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if got {
			t.Error("ExistsSheetInCampaign() = true, want false")
		}
	})

	t.Run("false when player has sheets in other campaigns only", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignA := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")
		campaignB := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign B")

		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")
		if _, err := pool.Exec(ctx,
			`UPDATE character_sheets SET campaign_uuid = $1 WHERE uuid = $2`,
			campaignA, sheetUUID,
		); err != nil {
			t.Fatalf("update campaign_uuid: %v", err)
		}

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignB),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if got {
			t.Error("ExistsSheetInCampaign() = true, want false")
		}
	})
}
