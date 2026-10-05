//go:build integration

package fog_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestPlayerMemory(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := fog.NewPlayerMemoryRepository(pool)
	ctx := context.Background()

	setupMatchAndMap := func(t *testing.T) (matchID, mapID uuid.UUID) {
		t.Helper()
		masterUUID := pgtest.InsertTestUser(t, pool, "gm-fog", "gm-fog@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Fog Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Fog Match")
		mapUUID := pgtest.InsertTestMap(t, pool, campaignUUID, "Fog Map")
		return uuid.MustParse(matchUUID), uuid.MustParse(mapUUID)
	}

	t.Run("upsert of two memories of the same (match, map) round-trips with the right Seen", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchID, mapID := setupMatchAndMap(t)
		player1 := uuid.New()
		player2 := uuid.New()

		mem1 := fogentity.PlayerMemory{
			PlayerID: player1,
			MatchID:  matchID,
			MapID:    mapID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-1"}: {},
			},
		}
		mem2 := fogentity.PlayerMemory{
			PlayerID: player2,
			MatchID:  matchID,
			MapID:    mapID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-1"}: {},
				{Kind: fogentity.FeatureWall, ID: "wall-2"}: {},
			},
		}

		if err := repo.Upsert(ctx, mem1); err != nil {
			t.Fatalf("upsert mem1: %v", err)
		}
		if err := repo.Upsert(ctx, mem2); err != nil {
			t.Fatalf("upsert mem2: %v", err)
		}

		got, err := repo.FindByMatchMap(ctx, matchID, mapID)
		if err != nil {
			t.Fatalf("FindByMatchMap: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 memories, got %d", len(got))
		}

		byPlayer := map[uuid.UUID]fogentity.PlayerMemory{}
		for _, m := range got {
			byPlayer[m.PlayerID] = m
		}

		g1, ok := byPlayer[player1]
		if !ok {
			t.Fatalf("missing memory for player1")
		}
		if len(g1.Seen) != 1 {
			t.Fatalf("player1: expected 1 seen feature, got %d (%v)", len(g1.Seen), g1.Seen)
		}
		if !g1.Has(fogentity.FeatureWall, "wall-1") {
			t.Fatalf("player1: expected to have seen wall-1")
		}

		g2, ok := byPlayer[player2]
		if !ok {
			t.Fatalf("missing memory for player2")
		}
		if len(g2.Seen) != 2 {
			t.Fatalf("player2: expected 2 seen features, got %d (%v)", len(g2.Seen), g2.Seen)
		}
		if !g2.Has(fogentity.FeatureWall, "wall-1") || !g2.Has(fogentity.FeatureWall, "wall-2") {
			t.Fatalf("player2: expected to have seen wall-1 and wall-2, got %v", g2.Seen)
		}
	})

	t.Run("upsert of an existing memory replaces Seen", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchID, mapID := setupMatchAndMap(t)
		player := uuid.New()

		first := fogentity.PlayerMemory{
			PlayerID: player,
			MatchID:  matchID,
			MapID:    mapID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-1"}: {},
			},
		}
		if err := repo.Upsert(ctx, first); err != nil {
			t.Fatalf("upsert first: %v", err)
		}

		second := fogentity.PlayerMemory{
			PlayerID: player,
			MatchID:  matchID,
			MapID:    mapID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-2"}: {},
				{Kind: fogentity.FeatureWall, ID: "wall-3"}: {},
			},
		}
		if err := repo.Upsert(ctx, second); err != nil {
			t.Fatalf("upsert second: %v", err)
		}

		got, err := repo.FindByMatchMap(ctx, matchID, mapID)
		if err != nil {
			t.Fatalf("FindByMatchMap: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 memory after replace-upsert, got %d", len(got))
		}
		if len(got[0].Seen) != 2 {
			t.Fatalf("expected replaced Seen to have 2 features, got %d (%v)", len(got[0].Seen), got[0].Seen)
		}
		if got[0].Has(fogentity.FeatureWall, "wall-1") {
			t.Fatalf("expected wall-1 to be gone after replace-upsert")
		}
		if !got[0].Has(fogentity.FeatureWall, "wall-2") || !got[0].Has(fogentity.FeatureWall, "wall-3") {
			t.Fatalf("expected wall-2 and wall-3 after replace-upsert, got %v", got[0].Seen)
		}
	})

	t.Run("DeleteByMatch clears every memory of the match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		matchID, mapID := setupMatchAndMap(t)
		player := uuid.New()

		mem := fogentity.PlayerMemory{
			PlayerID: player,
			MatchID:  matchID,
			MapID:    mapID,
			Seen: map[fogentity.FeatureRef]struct{}{
				{Kind: fogentity.FeatureWall, ID: "wall-1"}: {},
			},
		}
		if err := repo.Upsert(ctx, mem); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		if err := repo.DeleteByMatch(ctx, matchID); err != nil {
			t.Fatalf("DeleteByMatch: %v", err)
		}

		got, err := repo.FindByMatchMap(ctx, matchID, mapID)
		if err != nil {
			t.Fatalf("FindByMatchMap after delete: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no memories after DeleteByMatch, got %d", len(got))
		}
	})
}
