//go:build integration

package submission_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
)

func TestSubmitCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Hero")

	t.Run("happy path", func(t *testing.T) {
		err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		exists, err := repo.ExistsSubmittedCharacterSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error checking existence, got %v", err)
		}
		if !exists {
			t.Fatal("expected submission to exist after submit")
		}
	})

	t.Run("duplicate submission", func(t *testing.T) {
		err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
		if err == nil {
			t.Fatal("expected error for duplicate submission, got nil")
		}
	})
}

func TestGetSubmissionCampaignUUIDBySheetUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Hero")

	err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
	if err != nil {
		t.Fatalf("setup: failed to submit character sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetSubmissionCampaignUUIDBySheetUUID(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != uuid.MustParse(campaignUUID) {
			t.Fatalf("expected campaign UUID %s, got %s", campaignUUID, got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetSubmissionCampaignUUIDBySheetUUID(ctx, uuid.New())
		if !errors.Is(err, submission.ErrSubmissionNotFound) {
			t.Fatalf("expected ErrSubmissionNotFound, got %v", err)
		}
	})
}

func TestExistsSubmittedCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Hero")

	err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
	if err != nil {
		t.Fatalf("setup: failed to submit character sheet: %v", err)
	}

	t.Run("true", func(t *testing.T) {
		exists, err := repo.ExistsSubmittedCharacterSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
	})

	t.Run("false", func(t *testing.T) {
		exists, err := repo.ExistsSubmittedCharacterSheet(ctx, uuid.New())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
	})
}

func TestAcceptCharacterSheetSubmission(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Hero")

	err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
	if err != nil {
		t.Fatalf("setup: failed to submit character sheet: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		err := repo.AcceptCharacterSheetSubmission(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Submission should be deleted
		exists, err := repo.ExistsSubmittedCharacterSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error checking existence, got %v", err)
		}
		if exists {
			t.Fatal("expected submission to be deleted after accept")
		}

		// Sheet should have campaign_uuid set
		var gotCampaignUUID *string
		err = pool.QueryRow(ctx,
			`SELECT campaign_uuid FROM character_sheets WHERE uuid = $1`, sheetUUID,
		).Scan(&gotCampaignUUID)
		if err != nil {
			t.Fatalf("expected no error querying sheet, got %v", err)
		}
		if gotCampaignUUID == nil || *gotCampaignUUID != campaignUUID {
			t.Fatalf("expected sheet campaign_uuid %s, got %v", campaignUUID, gotCampaignUUID)
		}
	})
}

func TestRejectCharacterSheetSubmission(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Hero")

	err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
	if err != nil {
		t.Fatalf("setup: failed to submit character sheet: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		err := repo.RejectCharacterSheetSubmission(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		exists, err := repo.ExistsSubmittedCharacterSheet(ctx, uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("expected no error checking existence, got %v", err)
		}
		if exists {
			t.Fatal("expected submission to be deleted after reject")
		}
	})
}
