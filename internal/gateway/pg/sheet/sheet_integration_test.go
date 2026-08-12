//go:build integration

package sheet_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	domainsheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	pgcampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	pgsubmission "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildTestSheet(playerUUID *uuid.UUID) *domainsheet.CharacterSheet {
	factory := domainsheet.NewCharacterSheetFactory()
	profile := domainsheet.CharacterProfile{
		NickName:         "TestChar",
		FullName:         "Test Character",
		Alignment:        "Neutral",
		Description:      "A test character",
		BriefDescription: "Test",
		Birthday:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}

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
		Birthday:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}

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

	t.Run("happy path master-owned NPC sheet", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass")
		masterID := uuid.MustParse(masterStr)
		s := buildMasterTestSheet(&masterID)
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("expected no error for master-owned sheet, got: %v", err)
		}
		got, _, err := repo.GetCharacterSheetByUUID(ctx, s.UUID.String())
		if err != nil {
			t.Fatalf("expected sheet to be readable after create, got: %v", err)
		}
		if got.UUID != s.UUID {
			t.Fatalf("expected UUID %s, got %s", s.UUID, got.UUID)
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

func TestGetCharacterSheetBirthInfo(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "master1", "m@test.com", "pass")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, nil, &masterUUID, nil, "TestNick")

		birthday, age, err := repo.GetCharacterSheetBirthInfo(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if birthday.Year() != 0 {
			t.Errorf("expected year 0, got %d", birthday.Year())
		}
		if int(birthday.Month()) != 5 {
			t.Errorf("expected month 5, got %d", birthday.Month())
		}
		if birthday.Day() != 15 {
			t.Errorf("expected day 15, got %d", birthday.Day())
		}
		if age != 20 {
			t.Errorf("expected age 20, got %d", age)
		}
	})

	t.Run("not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		_, _, err := repo.GetCharacterSheetBirthInfo(ctx, uuid.New())
		if err == nil {
			t.Error("expected error for non-existent sheet UUID")
		}
	})
}

func TestGetCharacterSheetNormalizesStaleStatus(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()

	sheetRepo := sheet.NewRepository(pool)
	campaignRepo := pgcampaign.NewRepository(pool)
	submissionRepo := pgsubmission.NewRepository(pool)
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
			&sync.Map{}, factory, sheetRepo, campaignRepo, submissionRepo,
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

func TestUpdateCharacterSheetProfile(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	s := buildTestSheet(&playerUUID)
	if err := repo.CreateCharacterSheet(ctx, s); err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	sheetUUID := s.UUID

	avatarURL := "https://pub.r2.dev/avatar/abc.webp"
	coverURL := "https://pub.r2.dev/cover/abc.webp"

	err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, playerUUID, &avatarURL, &coverURL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verificar que os valores foram salvos
	got, _, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
	if err != nil {
		t.Fatalf("sheet not found after update: %v", err)
	}
	if got.GetProfile().AvatarURL == nil || *got.GetProfile().AvatarURL != avatarURL {
		t.Errorf("expected avatar_url %q, got %v", avatarURL, got.GetProfile().AvatarURL)
	}

	t.Run("wrong player returns error", func(t *testing.T) {
		err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, uuid.New(), &avatarURL, &coverURL, nil)
		if err == nil {
			t.Error("expected error for wrong player UUID, got nil")
		}
	})
}

// seedSheetWithProfile creates a fresh player and a character sheet whose
// profile already has avatarUrl, coverUrl and briefDescription populated, so
// partial-PATCH tests can assert that omitted fields survive the update.
func seedSheetWithProfile(
	t *testing.T, pool *pgxpool.Pool, repo *sheet.Repository, ctx context.Context,
	avatarURL, coverURL, briefDescription string,
) (playerUUID, sheetUUID uuid.UUID) {
	t.Helper()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID = uuid.MustParse(playerStr)

	factory := domainsheet.NewCharacterSheetFactory()
	avatar := avatarURL
	cover := coverURL
	profile := domainsheet.CharacterProfile{
		NickName:         "TestChar",
		FullName:         "Test Character",
		Alignment:        "Neutral",
		Description:      "A test character",
		BriefDescription: briefDescription,
		AvatarURL:        &avatar,
		CoverURL:         &cover,
		Birthday:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	s, err := factory.Build(&playerUUID, nil, nil, profile, nil, nil, nil)
	if err != nil {
		t.Fatalf("seedSheetWithProfile: failed to build sheet: %v", err)
	}
	s.UUID = uuid.New()
	if err := repo.CreateCharacterSheet(ctx, s); err != nil {
		t.Fatalf("seedSheetWithProfile: failed to create sheet: %v", err)
	}
	return playerUUID, s.UUID
}

func TestUpdateCharacterSheetProfilePartialUpdate(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	const initialAvatar = "https://pub.r2.dev/avatar/initial.webp"
	const initialCover = "https://pub.r2.dev/cover/initial.webp"
	const initialBrief = "Initial brief description"

	t.Run("PATCH with only avatarUrl preserves coverUrl and briefDescription", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerUUID, sheetUUID := seedSheetWithProfile(
			t, pool, repo, ctx, initialAvatar, initialCover, initialBrief,
		)

		newAvatar := "https://pub.r2.dev/avatar/updated.webp"
		err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, playerUUID, &newAvatar, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
		if err != nil {
			t.Fatalf("sheet not found after update: %v", err)
		}
		profile := got.GetProfile()
		if profile.AvatarURL == nil || *profile.AvatarURL != newAvatar {
			t.Errorf("expected avatarUrl %q, got %v", newAvatar, profile.AvatarURL)
		}
		if profile.CoverURL == nil || *profile.CoverURL != initialCover {
			t.Errorf("expected coverUrl to be preserved as %q, got %v", initialCover, profile.CoverURL)
		}
		if profile.BriefDescription != initialBrief {
			t.Errorf("expected briefDescription to be preserved as %q, got %q", initialBrief, profile.BriefDescription)
		}
	})

	t.Run("PATCH with only coverUrl preserves avatarUrl and briefDescription", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerUUID, sheetUUID := seedSheetWithProfile(
			t, pool, repo, ctx, initialAvatar, initialCover, initialBrief,
		)

		newCover := "https://pub.r2.dev/cover/updated.webp"
		err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, playerUUID, nil, &newCover, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
		if err != nil {
			t.Fatalf("sheet not found after update: %v", err)
		}
		profile := got.GetProfile()
		if profile.CoverURL == nil || *profile.CoverURL != newCover {
			t.Errorf("expected coverUrl %q, got %v", newCover, profile.CoverURL)
		}
		if profile.AvatarURL == nil || *profile.AvatarURL != initialAvatar {
			t.Errorf("expected avatarUrl to be preserved as %q, got %v", initialAvatar, profile.AvatarURL)
		}
		if profile.BriefDescription != initialBrief {
			t.Errorf("expected briefDescription to be preserved as %q, got %q", initialBrief, profile.BriefDescription)
		}
	})

	t.Run("PATCH with all three fields updates all three", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerUUID, sheetUUID := seedSheetWithProfile(
			t, pool, repo, ctx, initialAvatar, initialCover, initialBrief,
		)

		newAvatar := "https://pub.r2.dev/avatar/all-updated.webp"
		newCover := "https://pub.r2.dev/cover/all-updated.webp"
		newBrief := "Updated brief description"
		err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, playerUUID, &newAvatar, &newCover, &newBrief)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
		if err != nil {
			t.Fatalf("sheet not found after update: %v", err)
		}
		profile := got.GetProfile()
		if profile.AvatarURL == nil || *profile.AvatarURL != newAvatar {
			t.Errorf("expected avatarUrl %q, got %v", newAvatar, profile.AvatarURL)
		}
		if profile.CoverURL == nil || *profile.CoverURL != newCover {
			t.Errorf("expected coverUrl %q, got %v", newCover, profile.CoverURL)
		}
		if profile.BriefDescription != newBrief {
			t.Errorf("expected briefDescription %q, got %q", newBrief, profile.BriefDescription)
		}
	})

	t.Run("PATCH with all nil leaves everything untouched", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerUUID, sheetUUID := seedSheetWithProfile(
			t, pool, repo, ctx, initialAvatar, initialCover, initialBrief,
		)

		err := repo.UpdateCharacterSheetProfile(ctx, sheetUUID, playerUUID, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, _, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
		if err != nil {
			t.Fatalf("sheet not found after update: %v", err)
		}
		profile := got.GetProfile()
		if profile.AvatarURL == nil || *profile.AvatarURL != initialAvatar {
			t.Errorf("expected avatarUrl to be preserved as %q, got %v", initialAvatar, profile.AvatarURL)
		}
		if profile.CoverURL == nil || *profile.CoverURL != initialCover {
			t.Errorf("expected coverUrl to be preserved as %q, got %v", initialCover, profile.CoverURL)
		}
		if profile.BriefDescription != initialBrief {
			t.Errorf("expected briefDescription to be preserved as %q, got %q", initialBrief, profile.BriefDescription)
		}
	})
}

func TestDeleteNPCCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	masterStr := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	masterUUID := uuid.MustParse(masterStr)

	t.Run("happy path - NPC deleted", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr2 := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		masterUUID2 := uuid.MustParse(masterStr2)

		s := buildMasterTestSheet(&masterUUID2)
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("setup: failed to create NPC sheet: %v", err)
		}
		if err := repo.DeleteNPCCharacterSheet(ctx, s.UUID, masterUUID2); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("not found - wrong uuid", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")

		err := repo.DeleteNPCCharacterSheet(ctx, uuid.New(), masterUUID)
		if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})

	t.Run("not found - wrong master uuid", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr2 := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		masterUUID2 := uuid.MustParse(masterStr2)

		s := buildMasterTestSheet(&masterUUID2)
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("setup: failed to create NPC sheet: %v", err)
		}
		err := repo.DeleteNPCCharacterSheet(ctx, s.UUID, uuid.New())
		if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})
}

func TestExistsMatchParticipantForSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	t.Run("not participated", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")

		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, nil, &masterStr, nil, "NPC1")
		exists, err := repo.ExistsMatchParticipantForSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("participated in match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "Match")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, nil, &masterStr, nil, "NPC1")
		pgtest.InsertTestMatchParticipant(t, pool, matchUUID, sheetUUID, time.Now())

		exists, err := repo.ExistsMatchParticipantForSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
	})
}
