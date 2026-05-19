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
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Hero")

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
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Hero")

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
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Hero")

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
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Hero")

	err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Now())
	if err != nil {
		t.Fatalf("setup: failed to submit character sheet: %v", err)
	}

	t.Run("happy path", func(t *testing.T) {
		err := repo.AcceptCharacterSheetSubmission(ctx, uuid.MustParse(sheetUUID), uuid.MustParse(campaignUUID), time.Date(2000, 5, 15, 0, 0, 0, 0, time.UTC))
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

func TestExistsOtherCharacterWithNickInCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := submission.NewRepository(pool)
	ctx := context.Background()

	masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
	playerUUID := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")

	t.Run("no conflict — nick unique in campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID = pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID = pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
		playerUUID = pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")

		exists, err := repo.ExistsOtherCharacterWithNickInCampaign(ctx, "Gon", uuid.MustParse(campaignUUID), uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected false (no other character), got true")
		}
	})

	t.Run("conflict — nick taken by accepted character in campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID = pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID = pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
		playerUUID = pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
		player2UUID := pgtest.InsertTestUser(t, pool, "player2", "player2@test.com", "pass123")

		acceptedSheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &player2UUID, nil, &campaignUUID, "Gon")
		submittingSheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")

		exists, err := repo.ExistsOtherCharacterWithNickInCampaign(ctx, "Gon", uuid.MustParse(campaignUUID), uuid.MustParse(submittingSheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatalf("expected true (conflict with accepted sheet %s), got false", acceptedSheetUUID)
		}
	})

	t.Run("conflict — nick taken by pending submission in campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID = pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID = pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
		playerUUID = pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
		player2UUID := pgtest.InsertTestUser(t, pool, "player2", "player2@test.com", "pass123")

		pendingSheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &player2UUID, nil, nil, "Gon")
		submittingSheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")

		err := repo.SubmitCharacterSheet(ctx, uuid.MustParse(pendingSheetUUID), uuid.MustParse(campaignUUID), time.Now())
		if err != nil {
			t.Fatalf("setup: failed to submit pending sheet: %v", err)
		}

		exists, err := repo.ExistsOtherCharacterWithNickInCampaign(ctx, "Gon", uuid.MustParse(campaignUUID), uuid.MustParse(submittingSheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected true (conflict with pending submission), got false")
		}
	})

	t.Run("no conflict — same sheet excluded", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID = pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID = pgtest.InsertTestCampaign(t, pool, masterUUID, "TestCampaign")
		playerUUID = pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, &campaignUUID, "Gon")

		// The sheet itself is in the campaign but is excluded from the check
		exists, err := repo.ExistsOtherCharacterWithNickInCampaign(ctx, "Gon", uuid.MustParse(campaignUUID), uuid.MustParse(sheetUUID))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected false (only the excluded sheet has this nick), got true")
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
	sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Hero")

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
