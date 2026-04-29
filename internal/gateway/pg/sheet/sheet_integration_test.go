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

	t.Run("happy path master-owned minimal", func(t *testing.T) {
		now := time.Now()
		s := &model.CharacterSheet{
			UUID:         uuid.New(),
			PlayerUUID:   nil,
			CategoryName: "Reinforcement",
			CreatedAt:    now,
			UpdatedAt:    now,
			Profile: model.CharacterProfile{
				UUID:             uuid.New(),
				NickName:         "MasterNPC",
				FullName:         "Master NPC Character",
				Alignment:        "Evil",
				CharacterClass:   "Swordsman",
				Description:      "An NPC",
				BriefDescription: "NPC",
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		}
		err := repo.CreateCharacterSheet(ctx, s)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if s.ID == 0 {
			t.Fatal("expected sheet ID to be set after create")
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
