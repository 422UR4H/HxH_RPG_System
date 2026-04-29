//go:build integration

package enrollment_test

import (
	"context"
	"testing"

	enrollmentRepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/google/uuid"
)

func TestEnrollCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, "Gon")

		matchID := uuid.MustParse(matchUUID)
		sheetID := uuid.MustParse(sheetUUID)

		if err := repo.EnrollCharacterSheet(ctx, matchID, sheetID); err != nil {
			t.Fatalf("EnrollCharacterSheet() error = %v, want nil", err)
		}

		exists, err := repo.ExistsEnrolledCharacterSheet(ctx, sheetID, matchID)
		if err != nil {
			t.Fatalf("ExistsEnrolledCharacterSheet() error = %v", err)
		}
		if !exists {
			t.Error("ExistsEnrolledCharacterSheet() = false after enrollment, want true")
		}
	})

	t.Run("duplicate enrollment returns error", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, "Killua")

		matchID := uuid.MustParse(matchUUID)
		sheetID := uuid.MustParse(sheetUUID)

		if err := repo.EnrollCharacterSheet(ctx, matchID, sheetID); err != nil {
			t.Fatalf("first EnrollCharacterSheet() error = %v", err)
		}

		if err := repo.EnrollCharacterSheet(ctx, matchID, sheetID); err == nil {
			t.Fatal("second EnrollCharacterSheet() error = nil, want UNIQUE constraint error")
		}
	})
}

func TestExistsEnrolledCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("true when enrolled", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, "Kurapika")

		matchID := uuid.MustParse(matchUUID)
		sheetID := uuid.MustParse(sheetUUID)

		if err := repo.EnrollCharacterSheet(ctx, matchID, sheetID); err != nil {
			t.Fatalf("EnrollCharacterSheet() error = %v", err)
		}

		exists, err := repo.ExistsEnrolledCharacterSheet(ctx, sheetID, matchID)
		if err != nil {
			t.Fatalf("ExistsEnrolledCharacterSheet() error = %v", err)
		}
		if !exists {
			t.Error("ExistsEnrolledCharacterSheet() = false, want true")
		}
	})

	t.Run("false when not enrolled", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		exists, err := repo.ExistsEnrolledCharacterSheet(ctx, uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("ExistsEnrolledCharacterSheet() error = %v", err)
		}
		if exists {
			t.Error("ExistsEnrolledCharacterSheet() = true, want false")
		}
	})
}
