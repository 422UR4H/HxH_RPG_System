//go:build integration

package scenario_test

import (
	"context"
	"errors"
	"testing"
	"time"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	scenarioRepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/google/uuid"
)

func TestCreateScenario(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := scenarioRepo.NewRepository(pool)

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "creator", "creator@test.com", "pass123")

		s, err := scenarioEntity.NewScenario(
			uuid.MustParse(userUUID),
			"Greed Island",
			"A game world",
			"Full description of Greed Island scenario",
		)
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}

		if err := repo.CreateScenario(ctx, s); err != nil {
			t.Fatalf("CreateScenario() error = %v, want nil", err)
		}

		exists, err := repo.ExistsScenario(ctx, s.UUID)
		if err != nil {
			t.Fatalf("ExistsScenario() error = %v", err)
		}
		if !exists {
			t.Error("ExistsScenario() = false after CreateScenario, want true")
		}
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "creator", "creator@test.com", "pass123")
		uid := uuid.MustParse(userUUID)

		s1, err := scenarioEntity.NewScenario(uid, "NGL", "Brief 1", "Desc 1")
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s1); err != nil {
			t.Fatalf("first CreateScenario() error = %v", err)
		}

		s2, err := scenarioEntity.NewScenario(uid, "NGL", "Brief 2", "Desc 2")
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s2); err == nil {
			t.Fatal("CreateScenario() with duplicate name: expected error, got nil")
		}
	})
}

func TestGetScenario(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := scenarioRepo.NewRepository(pool)

	t.Run("found without campaigns", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "reader", "reader@test.com", "pass123")

		s, err := scenarioEntity.NewScenario(
			uuid.MustParse(userUUID),
			"York New City",
			"Auction arc",
			"Full description of York New City",
		)
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s); err != nil {
			t.Fatalf("CreateScenario() error = %v", err)
		}

		got, err := repo.GetScenario(ctx, s.UUID)
		if err != nil {
			t.Fatalf("GetScenario() error = %v, want nil", err)
		}
		if got.UUID != s.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, s.UUID)
		}
		if got.UserUUID != s.UserUUID {
			t.Errorf("UserUUID = %v, want %v", got.UserUUID, s.UserUUID)
		}
		if got.Name != s.Name {
			t.Errorf("Name = %q, want %q", got.Name, s.Name)
		}
		if got.BriefDescription != s.BriefDescription {
			t.Errorf("BriefDescription = %q, want %q", got.BriefDescription, s.BriefDescription)
		}
		if got.Description != s.Description {
			t.Errorf("Description = %q, want %q", got.Description, s.Description)
		}
		if len(got.Campaigns) != 0 {
			t.Errorf("Campaigns length = %d, want 0", len(got.Campaigns))
		}
	})

	t.Run("found with campaigns", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")

		s, err := scenarioEntity.NewScenario(
			uuid.MustParse(userUUID),
			"Heavens Arena",
			"Fighting tower",
			"Full description of Heavens Arena",
		)
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s); err != nil {
			t.Fatalf("CreateScenario() error = %v", err)
		}

		campaignUUID := uuid.New()
		now := time.Now()
		_, err = pool.Exec(ctx,
			`INSERT INTO campaigns
				(uuid, master_uuid, scenario_uuid, name,
				 brief_initial_description, description,
				 is_public, call_link, story_start_at,
				 created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			campaignUUID, userUUID, s.UUID, "Campaign Alpha",
			"Brief init desc", "Campaign description",
			true, "https://meet.example.com", now,
			now, now,
		)
		if err != nil {
			t.Fatalf("insert campaign error = %v", err)
		}

		got, err := repo.GetScenario(ctx, s.UUID)
		if err != nil {
			t.Fatalf("GetScenario() error = %v, want nil", err)
		}
		if len(got.Campaigns) != 1 {
			t.Fatalf("Campaigns length = %d, want 1", len(got.Campaigns))
		}
		if got.Campaigns[0].UUID != campaignUUID {
			t.Errorf("Campaign UUID = %v, want %v", got.Campaigns[0].UUID, campaignUUID)
		}
		if got.Campaigns[0].Name != "Campaign Alpha" {
			t.Errorf("Campaign Name = %q, want %q", got.Campaigns[0].Name, "Campaign Alpha")
		}
	})

	t.Run("not found returns ErrScenarioNotFound", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		_, err := repo.GetScenario(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetScenario() error = nil, want ErrScenarioNotFound")
		}
		if !errors.Is(err, scenarioRepo.ErrScenarioNotFound) {
			t.Errorf("GetScenario() error = %v, want ErrScenarioNotFound", err)
		}
	})
}

func TestListScenariosByUserUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := scenarioRepo.NewRepository(pool)

	t.Run("returns list ordered by name", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "lister", "lister@test.com", "pass123")
		uid := uuid.MustParse(userUUID)

		names := []string{"Zoldyck Estate", "Greed Island", "NGL"}
		for _, name := range names {
			s, err := scenarioEntity.NewScenario(uid, name, "Brief for "+name, "Description for "+name)
			if err != nil {
				t.Fatalf("NewScenario(%q) error = %v", name, err)
			}
			if err := repo.CreateScenario(ctx, s); err != nil {
				t.Fatalf("CreateScenario(%q) error = %v", name, err)
			}
		}

		list, err := repo.ListScenariosByUserUUID(ctx, uid)
		if err != nil {
			t.Fatalf("ListScenariosByUserUUID() error = %v", err)
		}
		if len(list) != 3 {
			t.Fatalf("list length = %d, want 3", len(list))
		}

		expectedOrder := []string{"Greed Island", "NGL", "Zoldyck Estate"}
		for i, want := range expectedOrder {
			if list[i].Name != want {
				t.Errorf("list[%d].Name = %q, want %q", i, list[i].Name, want)
			}
		}
	})

	t.Run("empty list for unknown user", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		list, err := repo.ListScenariosByUserUUID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("ListScenariosByUserUUID() error = %v", err)
		}
		if len(list) != 0 {
			t.Errorf("list length = %d, want 0", len(list))
		}
	})
}

func TestExistsScenarioWithName(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := scenarioRepo.NewRepository(pool)

	t.Run("true when exists", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "namer", "namer@test.com", "pass123")

		s, err := scenarioEntity.NewScenario(
			uuid.MustParse(userUUID),
			"Dark Continent",
			"Expedition",
			"Full description",
		)
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s); err != nil {
			t.Fatalf("CreateScenario() error = %v", err)
		}

		exists, err := repo.ExistsScenarioWithName(ctx, "Dark Continent")
		if err != nil {
			t.Fatalf("ExistsScenarioWithName() error = %v", err)
		}
		if !exists {
			t.Error("ExistsScenarioWithName() = false, want true")
		}
	})

	t.Run("false when not exists", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		exists, err := repo.ExistsScenarioWithName(ctx, "Nonexistent Scenario")
		if err != nil {
			t.Fatalf("ExistsScenarioWithName() error = %v", err)
		}
		if exists {
			t.Error("ExistsScenarioWithName() = true, want false")
		}
	})
}

func TestExistsScenario(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := scenarioRepo.NewRepository(pool)

	t.Run("true when exists", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		userUUID := pgtest.InsertTestUser(t, pool, "checker", "checker@test.com", "pass123")

		s, err := scenarioEntity.NewScenario(
			uuid.MustParse(userUUID),
			"Hunter Exam",
			"Phase 1",
			"Full description",
		)
		if err != nil {
			t.Fatalf("NewScenario() error = %v", err)
		}
		if err := repo.CreateScenario(ctx, s); err != nil {
			t.Fatalf("CreateScenario() error = %v", err)
		}

		exists, err := repo.ExistsScenario(ctx, s.UUID)
		if err != nil {
			t.Fatalf("ExistsScenario() error = %v", err)
		}
		if !exists {
			t.Error("ExistsScenario() = false, want true")
		}
	})

	t.Run("false when not exists", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		exists, err := repo.ExistsScenario(ctx, uuid.New())
		if err != nil {
			t.Fatalf("ExistsScenario() error = %v", err)
		}
		if exists {
			t.Error("ExistsScenario() = true, want false")
		}
	})
}
