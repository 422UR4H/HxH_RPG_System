//go:build integration

package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	entityMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func newTestMatch(masterUUID, campaignUUID uuid.UUID, title string, isPublic bool, gameScheduledAt time.Time) *entityMatch.Match {
	now := time.Now().Truncate(time.Microsecond)
	return &entityMatch.Match{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		CampaignUUID:            campaignUUID,
		Title:                   title,
		BriefInitialDescription: "Brief description for " + title,
		Description:             "Full description for " + title,
		IsPublic:                isPublic,
		GameScheduledAt:         gameScheduledAt.Truncate(time.Microsecond),
		StoryStartAt:            now,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

func TestCreateMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm1", "gm1@hunter.com", "pass"))
	campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Test Campaign"))

	t.Run("happy path", func(t *testing.T) {
		m := newTestMatch(masterUUID, campaignUUID, "First Session", true, time.Now().Add(24*time.Hour))

		if err := repo.CreateMatch(ctx, m); err != nil {
			t.Fatalf("CreateMatch() unexpected error: %v", err)
		}

		got, err := repo.GetMatch(ctx, m.UUID)
		if err != nil {
			t.Fatalf("GetMatch() after create: %v", err)
		}
		if got.UUID != m.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, m.UUID)
		}
		if got.Title != m.Title {
			t.Errorf("Title = %q, want %q", got.Title, m.Title)
		}
		if got.MasterUUID != masterUUID {
			t.Errorf("MasterUUID = %v, want %v", got.MasterUUID, masterUUID)
		}
		if got.CampaignUUID != campaignUUID {
			t.Errorf("CampaignUUID = %v, want %v", got.CampaignUUID, campaignUUID)
		}
		if got.IsPublic != true {
			t.Errorf("IsPublic = %v, want true", got.IsPublic)
		}
	})
}

func TestGetMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm2", "gm2@hunter.com", "pass"))
	campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Get Match Campaign"))

	t.Run("found", func(t *testing.T) {
		m := newTestMatch(masterUUID, campaignUUID, "Greed Island Session", true, time.Now().Add(48*time.Hour))

		if err := repo.CreateMatch(ctx, m); err != nil {
			t.Fatalf("CreateMatch() unexpected error: %v", err)
		}

		got, err := repo.GetMatch(ctx, m.UUID)
		if err != nil {
			t.Fatalf("GetMatch() unexpected error: %v", err)
		}
		if got.UUID != m.UUID {
			t.Errorf("UUID = %v, want %v", got.UUID, m.UUID)
		}
		if got.BriefInitialDescription != m.BriefInitialDescription {
			t.Errorf("BriefInitialDescription = %q, want %q", got.BriefInitialDescription, m.BriefInitialDescription)
		}
		if got.Description != m.Description {
			t.Errorf("Description = %q, want %q", got.Description, m.Description)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetMatch(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetMatch() expected error, got nil")
		}
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrMatchNotFound)
		}
	})
}

func TestGetMatchCampaignUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm3", "gm3@hunter.com", "pass"))
	campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "CampaignUUID Lookup"))

	t.Run("found", func(t *testing.T) {
		m := newTestMatch(masterUUID, campaignUUID, "Campaign Lookup Session", true, time.Now().Add(24*time.Hour))

		if err := repo.CreateMatch(ctx, m); err != nil {
			t.Fatalf("CreateMatch() unexpected error: %v", err)
		}

		got, err := repo.GetMatchCampaignUUID(ctx, m.UUID)
		if err != nil {
			t.Fatalf("GetMatchCampaignUUID() unexpected error: %v", err)
		}
		if got != campaignUUID {
			t.Errorf("campaignUUID = %v, want %v", got, campaignUUID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetMatchCampaignUUID(ctx, uuid.New())
		if err == nil {
			t.Fatal("GetMatchCampaignUUID() expected error, got nil")
		}
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrMatchNotFound)
		}
	})
}

func TestListMatchesByMasterUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm4", "gm4@hunter.com", "pass"))
	campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "List Matches Campaign"))

	t.Run("returns list", func(t *testing.T) {
		now := time.Now().Truncate(time.Microsecond)

		m1 := newTestMatch(masterUUID, campaignUUID, "Session Alpha", true, now.Add(24*time.Hour))
		m1.StoryStartAt = now.Add(-2 * time.Hour)

		m2 := newTestMatch(masterUUID, campaignUUID, "Session Beta", true, now.Add(48*time.Hour))
		m2.StoryStartAt = now.Add(-1 * time.Hour)

		if err := repo.CreateMatch(ctx, m1); err != nil {
			t.Fatalf("CreateMatch(m1) unexpected error: %v", err)
		}
		if err := repo.CreateMatch(ctx, m2); err != nil {
			t.Fatalf("CreateMatch(m2) unexpected error: %v", err)
		}

		list, err := repo.ListMatchesByMasterUUID(ctx, masterUUID)
		if err != nil {
			t.Fatalf("ListMatchesByMasterUUID() unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("list length = %d, want 2", len(list))
		}
		// ordered by story_start_at ASC
		if list[0].Title != "Session Alpha" {
			t.Errorf("list[0].Title = %q, want %q", list[0].Title, "Session Alpha")
		}
		if list[1].Title != "Session Beta" {
			t.Errorf("list[1].Title = %q, want %q", list[1].Title, "Session Beta")
		}
	})

	t.Run("empty", func(t *testing.T) {
		otherUUID := uuid.New()
		list, err := repo.ListMatchesByMasterUUID(ctx, otherUUID)
		if err != nil {
			t.Fatalf("ListMatchesByMasterUUID() unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("list length = %d, want 0", len(list))
		}
	})
}

func TestListPublicUpcomingMatches(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm5", "gm5@hunter.com", "pass"))
	otherMasterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm6", "gm6@hunter.com", "pass"))

	campaignOwn := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Own Campaign"))
	campaignOther := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, otherMasterUUID.String(), "Other Campaign"))

	now := time.Now().Truncate(time.Microsecond)

	t.Run("returns future public matches excluding own", func(t *testing.T) {
		// public + future + other master → should be returned
		m1 := newTestMatch(otherMasterUUID, campaignOther, "Public Future Other", true, now.Add(72*time.Hour))
		// public + future + own master → should be excluded
		m2 := newTestMatch(masterUUID, campaignOwn, "Public Future Own", true, now.Add(72*time.Hour))
		// private + future + other master → should be excluded
		m3 := newTestMatch(otherMasterUUID, campaignOther, "Private Future Other", false, now.Add(72*time.Hour))
		// public + past + other master → should be excluded
		m4 := newTestMatch(otherMasterUUID, campaignOther, "Public Past Other", true, now.Add(-72*time.Hour))

		for _, m := range []*entityMatch.Match{m1, m2, m3, m4} {
			if err := repo.CreateMatch(ctx, m); err != nil {
				t.Fatalf("CreateMatch(%s) unexpected error: %v", m.Title, err)
			}
		}

		list, err := repo.ListPublicUpcomingMatches(ctx, now, masterUUID)
		if err != nil {
			t.Fatalf("ListPublicUpcomingMatches() unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("list length = %d, want 1", len(list))
		}
		if list[0].Title != "Public Future Other" {
			t.Errorf("list[0].Title = %q, want %q", list[0].Title, "Public Future Other")
		}
	})

	t.Run("empty when no matching criteria", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		reqMasterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm7", "gm7@hunter.com", "pass"))

		list, err := repo.ListPublicUpcomingMatches(ctx, now, reqMasterUUID)
		if err != nil {
			t.Fatalf("ListPublicUpcomingMatches() unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("list length = %d, want 0", len(list))
		}
	})
}

func TestStartMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("atomically starts match, rejects pending, registers accepted", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_start", "gm_start@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Start Campaign"))
		player1UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p1_start", "p1_start@hunter.com", "pass"))
		player2UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p2_start", "p2_start@hunter.com", "pass"))

		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Start Session"))
		sheet1UUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{player1UUID.String()}[0], nil, nil, "Gon")
		sheet2UUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{player2UUID.String()}[0], nil, nil, "Killua")

		pgtest.InsertTestEnrollment(t, pool, matchUUID.String(), sheet1UUID, "accepted")
		pgtest.InsertTestEnrollment(t, pool, matchUUID.String(), sheet2UUID, "pending")

		gameStartAt := time.Now().UTC().Truncate(time.Microsecond)
		if err := repo.StartMatch(ctx, matchUUID, gameStartAt); err != nil {
			t.Fatalf("StartMatch() unexpected error: %v", err)
		}

		got, err := repo.GetMatch(ctx, matchUUID)
		if err != nil {
			t.Fatalf("GetMatch() after StartMatch: %v", err)
		}
		if got.GameStartAt == nil {
			t.Error("GameStartAt = nil, want non-nil")
		}

		var pendingCount int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM enrollments WHERE match_uuid = $1 AND status = 'pending'`, matchUUID,
		).Scan(&pendingCount); err != nil {
			t.Fatalf("pending count query: %v", err)
		}
		if pendingCount != 0 {
			t.Errorf("pending enrollments = %d, want 0", pendingCount)
		}

		var participantCount int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM match_participants WHERE match_uuid = $1`, matchUUID,
		).Scan(&participantCount); err != nil {
			t.Fatalf("participant count query: %v", err)
		}
		if participantCount != 1 {
			t.Errorf("participant count = %d, want 1 (only accepted)", participantCount)
		}

		var joinedAt time.Time
		if err := pool.QueryRow(ctx,
			`SELECT joined_at FROM match_participants WHERE match_uuid = $1`, matchUUID,
		).Scan(&joinedAt); err != nil {
			t.Fatalf("joined_at query: %v", err)
		}
		if !joinedAt.UTC().Truncate(time.Microsecond).Equal(gameStartAt) {
			t.Errorf("joined_at = %v, want %v", joinedAt.UTC().Truncate(time.Microsecond), gameStartAt)
		}
	})

	t.Run("already started returns error", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_dup", "gm_dup@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Dup Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Dup Session"))

		if err := repo.StartMatch(ctx, matchUUID, time.Now()); err != nil {
			t.Fatalf("StartMatch() first call unexpected error: %v", err)
		}

		err := repo.StartMatch(ctx, matchUUID, time.Now())
		if err == nil {
			t.Fatal("StartMatch() second call expected error, got nil")
		}
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrMatchNotFound)
		}
	})

	t.Run("non-existent match returns error", func(t *testing.T) {
		err := repo.StartMatch(ctx, uuid.New(), time.Now())
		if err == nil {
			t.Fatal("StartMatch() expected error, got nil")
		}
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrMatchNotFound)
		}
	})
}

func TestListParticipantsByMatchUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("returns participants with sheet data", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_list", "gm_list@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "List Campaign"))
		playerUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p_list", "p_list@hunter.com", "pass"))

		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "List Session"))
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{playerUUID.String()}[0], nil, nil, "Leorio")

		joinedAt := time.Now().UTC().Truncate(time.Microsecond)
		pgtest.InsertTestMatchParticipant(t, pool, matchUUID.String(), sheetUUID, joinedAt)

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("got %d participants, want 1", len(participants))
		}

		p := participants[0]
		if p.MatchUUID != matchUUID {
			t.Errorf("MatchUUID = %v, want %v", p.MatchUUID, matchUUID)
		}
		if p.Sheet.NickName != "Leorio" {
			t.Errorf("NickName = %q, want %q", p.Sheet.NickName, "Leorio")
		}
		if !p.JoinedAt.UTC().Truncate(time.Microsecond).Equal(joinedAt) {
			t.Errorf("JoinedAt = %v, want %v", p.JoinedAt, joinedAt)
		}
		if p.LeftAt != nil {
			t.Errorf("LeftAt = %v, want nil", p.LeftAt)
		}
	})

	t.Run("returns empty slice when no participants", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_empty", "gm_empty@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Empty Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Empty Session"))

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 0 {
			t.Errorf("got %d participants, want 0", len(participants))
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
