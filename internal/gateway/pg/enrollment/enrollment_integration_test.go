//go:build integration

package enrollment_test

import (
	"context"
	"errors"
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
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Gon")

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
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Killua")

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
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Kurapika")

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

func TestAcceptEnrollment(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Gon")
		enrollmentUUID := pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "pending")

		enrollID := uuid.MustParse(enrollmentUUID)

		if err := repo.AcceptEnrollment(ctx, enrollID); err != nil {
			t.Fatalf("AcceptEnrollment() error = %v, want nil", err)
		}

		status, _, err := repo.GetEnrollmentByUUID(ctx, enrollID)
		if err != nil {
			t.Fatalf("GetEnrollmentByUUID() error = %v", err)
		}
		if status != "accepted" {
			t.Errorf("GetEnrollmentByUUID() status = %q, want %q", status, "accepted")
		}
	})

	t.Run("not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		err := repo.AcceptEnrollment(ctx, uuid.New())
		if !errors.Is(err, enrollmentRepo.ErrEnrollmentNotFound) {
			t.Errorf("AcceptEnrollment() error = %v, want ErrEnrollmentNotFound", err)
		}
	})
}

func TestRejectEnrollment(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Killua")
		enrollmentUUID := pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "pending")

		enrollID := uuid.MustParse(enrollmentUUID)

		if err := repo.RejectEnrollment(ctx, enrollID); err != nil {
			t.Fatalf("RejectEnrollment() error = %v, want nil", err)
		}

		status, _, err := repo.GetEnrollmentByUUID(ctx, enrollID)
		if err != nil {
			t.Fatalf("GetEnrollmentByUUID() error = %v", err)
		}
		if status != "rejected" {
			t.Errorf("GetEnrollmentByUUID() status = %q, want %q", status, "rejected")
		}
	})

	t.Run("not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		err := repo.RejectEnrollment(ctx, uuid.New())
		if !errors.Is(err, enrollmentRepo.ErrEnrollmentNotFound) {
			t.Errorf("RejectEnrollment() error = %v, want ErrEnrollmentNotFound", err)
		}
	})
}

func TestGetEnrollmentByUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Kurapika")
		enrollmentUUID := pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "pending")

		enrollID := uuid.MustParse(enrollmentUUID)
		matchID := uuid.MustParse(matchUUID)

		status, gotMatchUUID, err := repo.GetEnrollmentByUUID(ctx, enrollID)
		if err != nil {
			t.Fatalf("GetEnrollmentByUUID() error = %v, want nil", err)
		}
		if status != "pending" {
			t.Errorf("GetEnrollmentByUUID() status = %q, want %q", status, "pending")
		}
		if gotMatchUUID != matchID {
			t.Errorf("GetEnrollmentByUUID() matchUUID = %v, want %v", gotMatchUUID, matchID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		_, _, err := repo.GetEnrollmentByUUID(ctx, uuid.New())
		if !errors.Is(err, enrollmentRepo.ErrEnrollmentNotFound) {
			t.Errorf("GetEnrollmentByUUID() error = %v, want ErrEnrollmentNotFound", err)
		}
	})
}

func TestRejectEnrollmentByPlayerAndMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Gon")
		enrollmentUUID := pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "accepted")

		playerID := uuid.MustParse(userUUID)
		matchID := uuid.MustParse(matchUUID)
		enrollID := uuid.MustParse(enrollmentUUID)

		if err := repo.RejectEnrollmentByPlayerAndMatch(ctx, playerID, matchID); err != nil {
			t.Fatalf("RejectEnrollmentByPlayerAndMatch() error = %v, want nil", err)
		}

		status, _, err := repo.GetEnrollmentByUUID(ctx, enrollID)
		if err != nil {
			t.Fatalf("GetEnrollmentByUUID() error = %v", err)
		}
		if status != "rejected" {
			t.Errorf("GetEnrollmentByUUID() status = %q, want %q", status, "rejected")
		}
	})

	t.Run("not found when no accepted enrollment", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Killua")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "pending")

		playerID := uuid.MustParse(userUUID)
		matchID := uuid.MustParse(matchUUID)

		err := repo.RejectEnrollmentByPlayerAndMatch(ctx, playerID, matchID)
		if !errors.Is(err, enrollmentRepo.ErrEnrollmentNotFound) {
			t.Errorf("RejectEnrollmentByPlayerAndMatch() error = %v, want ErrEnrollmentNotFound", err)
		}
	})

	t.Run("not found when player has no enrollment", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")

		matchID := uuid.MustParse(matchUUID)

		err := repo.RejectEnrollmentByPlayerAndMatch(ctx, uuid.New(), matchID)
		if !errors.Is(err, enrollmentRepo.ErrEnrollmentNotFound) {
			t.Errorf("RejectEnrollmentByPlayerAndMatch() error = %v, want ErrEnrollmentNotFound", err)
		}
	})
}

func TestIsPlayerEnrolledInMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("true when accepted", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Gon")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "accepted")

		playerID := uuid.MustParse(userUUID)
		matchID := uuid.MustParse(matchUUID)

		enrolled, err := repo.IsPlayerEnrolledInMatch(ctx, playerID, matchID)
		if err != nil {
			t.Fatalf("IsPlayerEnrolledInMatch() error = %v, want nil", err)
		}
		if !enrolled {
			t.Error("IsPlayerEnrolledInMatch() = false, want true")
		}
	})

	t.Run("false when pending", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		userUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, userUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, userUUID, campaignUUID, "Match 1")
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &userUUID, nil, nil, "Killua")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheetUUID, "pending")

		playerID := uuid.MustParse(userUUID)
		matchID := uuid.MustParse(matchUUID)

		enrolled, err := repo.IsPlayerEnrolledInMatch(ctx, playerID, matchID)
		if err != nil {
			t.Fatalf("IsPlayerEnrolledInMatch() error = %v, want nil", err)
		}
		if enrolled {
			t.Error("IsPlayerEnrolledInMatch() = true, want false")
		}
	})

	t.Run("false when no enrollment", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		enrolled, err := repo.IsPlayerEnrolledInMatch(ctx, uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("IsPlayerEnrolledInMatch() error = %v, want nil", err)
		}
		if enrolled {
			t.Error("IsPlayerEnrolledInMatch() = true, want false")
		}
	})
}

func TestListByMatchUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("lists all statuses ordered by created_at and includes joined sheet+player data", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match 1")

		sheet1 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, &campaignUUID, "Gon")
		sheet2 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, &campaignUUID, "Killua")
		sheet3 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, &campaignUUID, "Kurapika")

		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet1, "pending")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet2, "accepted")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet3, "rejected")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchUUID))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 3 {
			t.Fatalf("ListByMatchUUID() len = %d, want 3", len(got))
		}

		wantNicks := []string{"Gon", "Killua", "Kurapika"}
		wantStatuses := []string{"pending", "accepted", "rejected"}
		for i, e := range got {
			if e.CharacterSheet.NickName != wantNicks[i] {
				t.Errorf("row %d: nick = %q, want %q", i, e.CharacterSheet.NickName, wantNicks[i])
			}
			if e.Status != wantStatuses[i] {
				t.Errorf("row %d: status = %q, want %q", i, e.Status, wantStatuses[i])
			}
			if e.Player.Nick != "player1" {
				t.Errorf("row %d: player nick = %q, want %q", i, e.Player.Nick, "player1")
			}
			if e.Player.UUID.String() != playerUUID {
				t.Errorf("row %d: player uuid = %s, want %s", i, e.Player.UUID, playerUUID)
			}
			if e.CharacterSheet.UUID == uuid.Nil {
				t.Errorf("row %d: character sheet uuid is nil", i)
			}
			if e.CharacterSheet.CampaignUUID == nil {
				t.Errorf("row %d: campaign_uuid is nil, want set", i)
			}
		}
	})

	t.Run("returns empty slice when match has no enrollments", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match 1")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchUUID))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("ListByMatchUUID() len = %d, want 0", len(got))
		}
	})

	t.Run("does not include enrollments from other matches", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchA := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match A")
		matchB := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match B")

		sheetA := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Gon")
		sheetB := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, nil, "Killua")

		pgtest.InsertTestEnrollment(t, pool, matchA, sheetA, "pending")
		pgtest.InsertTestEnrollment(t, pool, matchB, sheetB, "accepted")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchA))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("ListByMatchUUID() len = %d, want 1", len(got))
		}
		if got[0].CharacterSheet.NickName != "Gon" {
			t.Errorf("got nick %q, want %q", got[0].CharacterSheet.NickName, "Gon")
		}
	})
}
