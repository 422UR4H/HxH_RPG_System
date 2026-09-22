//go:build integration

package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestAddNPCParticipant(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("adds npc", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_npc_add", "gm_npc_add@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "NPC Add Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "NPC Add Session"))

		sheetUUIDStr := pgtest.InsertTestCharacterSheet(t, pool, nil, &[]string{masterUUID.String()}[0], &[]string{campaignUUID.String()}[0], "Goblin")
		sheetUUID := mustParseUUID(t, sheetUUIDStr)

		joinedAt := time.Now().UTC().Truncate(time.Microsecond)
		p, err := repo.AddNPCParticipant(ctx, matchUUID, sheetUUID, joinedAt)
		if err != nil {
			t.Fatalf("AddNPCParticipant() unexpected error: %v", err)
		}
		if p.Sheet.UUID != sheetUUID {
			t.Errorf("Sheet.UUID = %v, want %v", p.Sheet.UUID, sheetUUID)
		}
		if p.MatchUUID != matchUUID {
			t.Errorf("MatchUUID = %v, want %v", p.MatchUUID, matchUUID)
		}

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("got %d participants, want 1", len(participants))
		}
		got := participants[0]
		if got.Sheet.PlayerUUID != nil {
			t.Errorf("Sheet.PlayerUUID = %v, want nil", got.Sheet.PlayerUUID)
		}
		if got.Sheet.MasterUUID == nil || *got.Sheet.MasterUUID != masterUUID {
			t.Errorf("Sheet.MasterUUID = %v, want %v", got.Sheet.MasterUUID, masterUUID)
		}
	})

	t.Run("duplicate returns ErrNPCAlreadyInMatch", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_npc_dup", "gm_npc_dup@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "NPC Dup Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "NPC Dup Session"))

		sheetUUIDStr := pgtest.InsertTestCharacterSheet(t, pool, nil, &[]string{masterUUID.String()}[0], &[]string{campaignUUID.String()}[0], "Slime")
		sheetUUID := mustParseUUID(t, sheetUUIDStr)

		joinedAt := time.Now().UTC().Truncate(time.Microsecond)
		if _, err := repo.AddNPCParticipant(ctx, matchUUID, sheetUUID, joinedAt); err != nil {
			t.Fatalf("AddNPCParticipant() first call unexpected error: %v", err)
		}

		_, err := repo.AddNPCParticipant(ctx, matchUUID, sheetUUID, joinedAt)
		if !errors.Is(err, pgMatch.ErrNPCAlreadyInMatch) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrNPCAlreadyInMatch)
		}

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("got %d participants, want 1", len(participants))
		}
	})

	t.Run("player sheet is rejected", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_npc_rej", "gm_npc_rej@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "NPC Reject Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "NPC Reject Session"))
		playerUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p_npc_rej", "p_npc_rej@hunter.com", "pass"))

		sheetUUIDStr := pgtest.InsertTestCharacterSheet(t, pool, &[]string{playerUUID.String()}[0], nil, nil, "Gon")
		sheetUUID := mustParseUUID(t, sheetUUIDStr)

		_, err := repo.AddNPCParticipant(ctx, matchUUID, sheetUUID, time.Now().UTC().Truncate(time.Microsecond))
		if !errors.Is(err, pgMatch.ErrSheetNotEligibleNPC) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrSheetNotEligibleNPC)
		}

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 0 {
			t.Errorf("got %d participants, want 0", len(participants))
		}
	})
}

func TestRemoveNPCParticipant(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("removes npc", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_npc_rm", "gm_npc_rm@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "NPC Remove Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "NPC Remove Session"))

		sheetUUIDStr := pgtest.InsertTestCharacterSheet(t, pool, nil, &[]string{masterUUID.String()}[0], &[]string{campaignUUID.String()}[0], "Orc")
		sheetUUID := mustParseUUID(t, sheetUUIDStr)

		joinedAt := time.Now().UTC().Truncate(time.Microsecond)
		if _, err := repo.AddNPCParticipant(ctx, matchUUID, sheetUUID, joinedAt); err != nil {
			t.Fatalf("AddNPCParticipant() unexpected error: %v", err)
		}

		if err := repo.RemoveNPCParticipant(ctx, matchUUID, sheetUUID); err != nil {
			t.Fatalf("RemoveNPCParticipant() unexpected error: %v", err)
		}

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 0 {
			t.Errorf("got %d participants, want 0", len(participants))
		}

		err = repo.RemoveNPCParticipant(ctx, matchUUID, sheetUUID)
		if !errors.Is(err, pgMatch.ErrNPCNotInMatch) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrNPCNotInMatch)
		}
	})

	t.Run("does not delete player participant", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_npc_guard", "gm_npc_guard@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "NPC Guard Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "NPC Guard Session"))
		playerUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p_npc_guard", "p_npc_guard@hunter.com", "pass"))

		sheetUUIDStr := pgtest.InsertTestCharacterSheet(t, pool, &[]string{playerUUID.String()}[0], nil, nil, "Killua")
		sheetUUID := mustParseUUID(t, sheetUUIDStr)

		joinedAt := time.Now().UTC().Truncate(time.Microsecond)
		pgtest.InsertTestMatchParticipant(t, pool, matchUUID.String(), sheetUUIDStr, joinedAt)

		err := repo.RemoveNPCParticipant(ctx, matchUUID, sheetUUID)
		if !errors.Is(err, pgMatch.ErrNPCNotInMatch) {
			t.Errorf("error = %v, want %v", err, pgMatch.ErrNPCNotInMatch)
		}

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("got %d participants, want 1 (player must remain)", len(participants))
		}
		if participants[0].Sheet.UUID != sheetUUID {
			t.Errorf("remaining participant Sheet.UUID = %v, want %v", participants[0].Sheet.UUID, sheetUUID)
		}
	})
}
