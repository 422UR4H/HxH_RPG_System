//go:build integration

package sheet_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainsheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	pgcampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

func buildTestSheet(playerUUID *uuid.UUID) *domainsheet.CharacterSheet {
	factory := domainsheet.NewCharacterSheetFactory()
	profile := domainsheet.CharacterProfile{
		NickName:         "TestChar",
		FullName:         "Test Character",
		Alignment:        "Neutral",
		Description:      "A test character",
		BriefDescription: "Test",
	}
	birthday := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	profile.Birthday = &birthday

	s, err := factory.Build(playerUUID, nil, nil, profile, nil, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("buildTestSheet: %v", err))
	}
	s.UUID = uuid.New()
	return s
}

func buildMasterTestSheet(masterUUID *uuid.UUID) *domainsheet.CharacterSheet {
	factory := domainsheet.NewCharacterSheetFactory()
	profile := domainsheet.CharacterProfile{
		NickName:         "MasterChar",
		FullName:         "Master Character",
		Alignment:        "Neutral",
		Description:      "A master-owned test character",
		BriefDescription: "Master",
	}
	birthday := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	profile.Birthday = &birthday

	s, err := factory.Build(nil, masterUUID, nil, profile, nil, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("buildMasterTestSheet: %v", err))
	}
	s.UUID = uuid.New()
	return s
}

// testBar is a minimal status.IStatusBar for use in integration tests.
type testBar struct{ min, curr, max int }

func (b testBar) GetMin() int              { return b.min }
func (b testBar) GetCurrent() int          { return b.curr }
func (b testBar) GetMax() int              { return b.max }
func (b testBar) IncreaseAt(int) int       { return b.curr }
func (b testBar) DecreaseAt(int) int       { return b.curr }
func (b testBar) Upgrade()                 {}
func (b testBar) SetCurrent(int) error     { return nil }

var _ status.IStatusBar = testBar{}

func TestCreateCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	t.Run("happy path player-owned with proficiencies", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		s := buildTestSheet(&playerUUID)
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		got, _, err := repo.GetCharacterSheetByUUID(ctx, s.UUID.String())
		if err != nil {
			t.Fatalf("expected sheet to be readable after create, got: %v", err)
		}
		if got.UUID != s.UUID {
			t.Fatalf("expected UUID %s, got %s", s.UUID, got.UUID)
		}
	})

	t.Run("master-owned sheet not insertable via player-only repo", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		// NOTE: CreateCharacterSheet only inserts player_uuid, not master_uuid.
		// A master-owned sheet (playerUUID=nil, masterUUID=non-nil) hits the DB XOR constraint.
		masterStr := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass")
		masterID := uuid.MustParse(masterStr)
		s := buildMasterTestSheet(&masterID)
		err := repo.CreateCharacterSheet(ctx, s)
		if err == nil {
			t.Fatal("expected error for master-owned sheet, got nil")
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
	if err := repo.CreateCharacterSheet(ctx, created); err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, _, err := repo.GetCharacterSheetByUUID(ctx, created.UUID.String())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.UUID != created.UUID {
			t.Fatalf("expected UUID %s, got %s", created.UUID, got.UUID)
		}
		if got.GetProfile().NickName != "TestChar" {
			t.Fatalf("expected nickname %q, got %q", "TestChar", got.GetProfile().NickName)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, _, err := repo.GetCharacterSheetByUUID(ctx, uuid.New().String())
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
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
	if err := repo.CreateCharacterSheet(ctx, created); err != nil {
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
	if err := repo.CreateCharacterSheet(ctx, created); err != nil {
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
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
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
	if err := repo.CreateCharacterSheet(ctx, created); err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("true", func(t *testing.T) {
		exists, err := repo.ExistsCharacterWithNick(ctx, created.GetProfile().NickName)
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
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
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
	if err := repo.CreateCharacterSheet(ctx, s); err != nil {
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
		if list[0].NickName != s.GetProfile().NickName {
			t.Fatalf("expected nickname %q, got %q", s.GetProfile().NickName, list[0].NickName)
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
	if err := repo.CreateCharacterSheet(ctx, s); err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		newVal := 42
		if err := repo.UpdateNenHexagonValue(ctx, s.UUID.String(), newVal); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		got, _, err := repo.GetCharacterSheetByUUID(ctx, s.UUID.String())
		if err != nil {
			t.Fatalf("expected no error fetching sheet, got %v", err)
		}
		if got.GetCurrHexValue() == nil || *got.GetCurrHexValue() != newVal {
			t.Fatalf("expected hex value %d, got %v", newVal, got.GetCurrHexValue())
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

		health := testBar{min: 0, curr: 17, max: 20}
		stamina := testBar{min: 0, curr: 0, max: 0}
		aura := testBar{min: 0, curr: 0, max: 0}

		if err := repo.UpdateStatusBars(ctx, sheetUUID, health, stamina, aura); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		var healthMin, healthCurr, healthMax int
		err := pool.QueryRow(ctx,
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
		health := testBar{min: 0, curr: 10, max: 20}
		stamina := testBar{min: 0, curr: 5, max: 10}
		aura := testBar{min: 0, curr: 0, max: 0}

		if err := repo.UpdateStatusBars(ctx, uuid.New().String(), health, stamina, aura); err != nil {
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

func TestGetCharacterSheetNormalizesStaleStatus(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()

	sheetRepo := sheet.NewRepository(pool)
	campaignRepo := pgcampaign.NewRepository(pool)
	factory := domainsheet.NewCharacterSheetFactory()

	t.Run("normalizes stale curr in returned entity", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass")
		playerUUID := uuid.MustParse(playerStr)

		s := buildTestSheet(&playerUUID)
		if err := sheetRepo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("setup: failed to create sheet: %v", err)
		}

		// Simulate stale data: curr=25, max=30 (persisted under old rules).
		// Base health max for a sheet with no XP is 20.
		// normalizeStatus(25, 30, 20, 0) → round(20*25/30) = 17.
		if _, err := pool.Exec(ctx,
			`UPDATE character_sheets SET health_curr_pts = 25, health_max_pts = 30 WHERE uuid = $1`,
			s.UUID,
		); err != nil {
			t.Fatalf("failed to inject stale health values: %v", err)
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			&sync.Map{}, factory, sheetRepo, campaignRepo,
		)

		result, err := uc.GetCharacterSheet(ctx, s.UUID, playerUUID)
		if err != nil {
			t.Fatalf("GetCharacterSheet() error = %v", err)
		}

		allBars := result.GetAllStatusBar()
		if got := allBars[enum.Health].GetCurrent(); got != 17 {
			t.Errorf("domain health curr = %d, want 17", got)
		}
		// NOTE: async DB persist is not triggered from use case (TODO: expose wasCorrected).
		// Only the in-memory normalization is verified here.
	})
}
