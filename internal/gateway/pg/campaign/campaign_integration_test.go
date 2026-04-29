//go:build integration

package campaign_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	entityCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func newTestCampaign(masterUUID uuid.UUID, scenarioUUID *uuid.UUID, name string) *entityCampaign.Campaign {
	now := time.Now().Truncate(time.Microsecond)
	return &entityCampaign.Campaign{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		ScenarioUUID:            scenarioUUID,
		Name:                    name,
		BriefInitialDescription: "A brief description for " + name,
		Description:             "Full description for " + name,
		IsPublic:                true,
		CallLink:                "https://meet.example.com/" + name,
		StoryStartAt:            now,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

func TestCreateCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master1", "master1@hunter.com", "pass"))

	t.Run("happy path with scenario_uuid", func(t *testing.T) {
		scenUUID := mustParseUUID(t, pgtest.InsertTestScenario(t, pool, masterUUID.String(), "Greed Island"))
		c := newTestCampaign(masterUUID, &scenUUID, "Campaign With Scenario")

		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatalf("CreateCampaign() unexpected error: %v", err)
		}

		got, err := repo.GetCampaign(ctx, c.UUID)
		if err != nil {
			t.Fatalf("GetCampaign() after create: %v", err)
		}
		if got.UUID != c.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, c.UUID)
		}
		if got.Name != c.Name {
			t.Errorf("Name = %q, want %q", got.Name, c.Name)
		}
		if got.ScenarioUUID == nil || *got.ScenarioUUID != scenUUID {
			t.Errorf("ScenarioUUID = %v, want %v", got.ScenarioUUID, scenUUID)
		}
	})

	t.Run("happy path without scenario_uuid", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID2 := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master2", "master2@hunter.com", "pass"))

		c := newTestCampaign(masterUUID2, nil, "Campaign Without Scenario")

		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatalf("CreateCampaign() unexpected error: %v", err)
		}

		got, err := repo.GetCampaign(ctx, c.UUID)
		if err != nil {
			t.Fatalf("GetCampaign() after create: %v", err)
		}
		if got.ScenarioUUID != nil {
			t.Errorf("ScenarioUUID = %v, want nil", got.ScenarioUUID)
		}
	})
}

func TestGetCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	t.Run("found without sheets or matches", func(t *testing.T) {
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm1", "gm1@hunter.com", "pass"))
		c := newTestCampaign(masterUUID, nil, "Yorknew City Arc")

		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatalf("CreateCampaign() unexpected error: %v", err)
		}

		got, err := repo.GetCampaign(ctx, c.UUID)
		if err != nil {
			t.Fatalf("GetCampaign() unexpected error: %v", err)
		}
		if got.UUID != c.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, c.UUID)
		}
		if got.MasterUUID != masterUUID {
			t.Errorf("MasterUUID = %v, want %v", got.MasterUUID, masterUUID)
		}
		if got.Name != "Yorknew City Arc" {
			t.Errorf("Name = %q, want %q", got.Name, "Yorknew City Arc")
		}
		if got.BriefInitialDescription != c.BriefInitialDescription {
			t.Errorf("BriefInitialDescription = %q, want %q", got.BriefInitialDescription, c.BriefInitialDescription)
		}
		if got.IsPublic != true {
			t.Errorf("IsPublic = %v, want true", got.IsPublic)
		}
		if len(got.CharacterSheets) != 0 {
			t.Errorf("CharacterSheets length = %d, want 0", len(got.CharacterSheets))
		}
		if len(got.PendingSheets) != 0 {
			t.Errorf("PendingSheets length = %d, want 0", len(got.PendingSheets))
		}
		if len(got.Matches) != 0 {
			t.Errorf("Matches length = %d, want 0", len(got.Matches))
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCampaign(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetCampaign() expected error, got nil")
		}
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("error = %v, want %v", err, pgCampaign.ErrCampaignNotFound)
		}
	})
}

func TestListCampaignsByMasterUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master3", "master3@hunter.com", "pass"))
	scenUUID := mustParseUUID(t, pgtest.InsertTestScenario(t, pool, masterUUID.String(), "HxH World"))

	t.Run("returns list", func(t *testing.T) {
		c1 := newTestCampaign(masterUUID, &scenUUID, "Alpha Campaign")
		c2 := newTestCampaign(masterUUID, &scenUUID, "Beta Campaign")

		if err := repo.CreateCampaign(ctx, c1); err != nil {
			t.Fatalf("CreateCampaign(c1) unexpected error: %v", err)
		}
		if err := repo.CreateCampaign(ctx, c2); err != nil {
			t.Fatalf("CreateCampaign(c2) unexpected error: %v", err)
		}

		list, err := repo.ListCampaignsByMasterUUID(ctx, masterUUID)
		if err != nil {
			t.Fatalf("ListCampaignsByMasterUUID() unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("list length = %d, want 2", len(list))
		}
		// ordered by name ASC
		if list[0].Name != "Alpha Campaign" {
			t.Errorf("list[0].Name = %q, want %q", list[0].Name, "Alpha Campaign")
		}
		if list[1].Name != "Beta Campaign" {
			t.Errorf("list[1].Name = %q, want %q", list[1].Name, "Beta Campaign")
		}
	})

	t.Run("empty", func(t *testing.T) {
		otherUUID := uuid.New()
		list, err := repo.ListCampaignsByMasterUUID(ctx, otherUUID)
		if err != nil {
			t.Fatalf("ListCampaignsByMasterUUID() unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("list length = %d, want 0", len(list))
		}
	})
}

func TestGetCampaignMasterUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master4", "master4@hunter.com", "pass"))
		c := newTestCampaign(masterUUID, nil, "Master Lookup Campaign")

		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatalf("CreateCampaign() unexpected error: %v", err)
		}

		got, err := repo.GetCampaignMasterUUID(ctx, c.UUID)
		if err != nil {
			t.Fatalf("GetCampaignMasterUUID() unexpected error: %v", err)
		}
		if got != masterUUID {
			t.Errorf("masterUUID = %v, want %v", got, masterUUID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCampaignMasterUUID(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetCampaignMasterUUID() expected error, got nil")
		}
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("error = %v, want %v", err, pgCampaign.ErrCampaignNotFound)
		}
	})
}

func TestGetCampaignStoryDates(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master5", "master5@hunter.com", "pass"))
		c := newTestCampaign(masterUUID, nil, "Story Dates Campaign")

		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatalf("CreateCampaign() unexpected error: %v", err)
		}

		got, err := repo.GetCampaignStoryDates(ctx, c.UUID)
		if err != nil {
			t.Fatalf("GetCampaignStoryDates() unexpected error: %v", err)
		}
		if got.UUID != c.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, c.UUID)
		}
		if got.MasterUUID != masterUUID {
			t.Errorf("MasterUUID = %v, want %v", got.MasterUUID, masterUUID)
		}
		if got.StoryStartAt.IsZero() {
			t.Error("StoryStartAt is zero, expected non-zero")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCampaignStoryDates(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetCampaignStoryDates() expected error, got nil")
		}
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Errorf("error = %v, want %v", err, pgCampaign.ErrCampaignNotFound)
		}
	})
}

func TestCountCampaignsByMasterUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master6", "master6@hunter.com", "pass"))

	t.Run("count zero", func(t *testing.T) {
		count, err := repo.CountCampaignsByMasterUUID(ctx, masterUUID)
		if err != nil {
			t.Fatalf("CountCampaignsByMasterUUID() unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("count greater than zero", func(t *testing.T) {
		c1 := newTestCampaign(masterUUID, nil, "Count Campaign A")
		c2 := newTestCampaign(masterUUID, nil, "Count Campaign B")

		if err := repo.CreateCampaign(ctx, c1); err != nil {
			t.Fatalf("CreateCampaign(c1) unexpected error: %v", err)
		}
		if err := repo.CreateCampaign(ctx, c2); err != nil {
			t.Fatalf("CreateCampaign(c2) unexpected error: %v", err)
		}

		count, err := repo.CountCampaignsByMasterUUID(ctx, masterUUID)
		if err != nil {
			t.Fatalf("CountCampaignsByMasterUUID() unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})
}

func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse UUID %q: %v", s, err)
	}
	return id
}
