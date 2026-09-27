//go:build integration

package pgmatchevent_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	pgmatchevent "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchevent"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestMatchEvent(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgmatchevent.NewRepository(pool)
	ctx := context.Background()

	t.Run("insert of three events out of insertion order lists them by created_at", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "gm-event", "gm-event@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Event Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Event Match")
		matchUUID := uuid.MustParse(matchStr)
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		sceneUUID := uuid.MustParse(sceneStr)
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Free")
		roundUUID := uuid.MustParse(roundStr)

		mkPayload := func(from, to string) json.RawMessage {
			b, _ := json.Marshal(map[string]string{"from": from, "to": to})
			return b
		}

		second := matchevent.Event{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			SceneUUID: sceneUUID,
			RoundUUID: roundUUID,
			Kind:      matchevent.KindRoundModeChanged,
			Payload:   mkPayload("Free", "Race"),
			CreatedAt: mustParseTime(t, "2026-09-27T10:00:01Z"),
		}
		third := matchevent.Event{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			SceneUUID: sceneUUID,
			RoundUUID: roundUUID,
			Kind:      matchevent.KindRoundModeChanged,
			Payload:   mkPayload("Race", "Free"),
			CreatedAt: mustParseTime(t, "2026-09-27T10:00:02Z"),
		}
		first := matchevent.Event{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			SceneUUID: sceneUUID,
			RoundUUID: roundUUID,
			Kind:      matchevent.KindRoundModeChanged,
			Payload:   mkPayload("Free", "Free"),
			CreatedAt: mustParseTime(t, "2026-09-27T10:00:00Z"),
		}

		// Inserted out of chronological order on purpose (second, third, first).
		if err := repo.Insert(ctx, second); err != nil {
			t.Fatalf("insert second: %v", err)
		}
		if err := repo.Insert(ctx, third); err != nil {
			t.Fatalf("insert third: %v", err)
		}
		if err := repo.Insert(ctx, first); err != nil {
			t.Fatalf("insert first: %v", err)
		}

		got, err := repo.ListByMatch(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListByMatch: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 events, got %d", len(got))
		}
		if got[0].UUID != first.UUID || got[1].UUID != second.UUID || got[2].UUID != third.UUID {
			t.Fatalf("expected order first,second,third by created_at, got %v", []uuid.UUID{got[0].UUID, got[1].UUID, got[2].UUID})
		}
		if got[0].Kind != matchevent.KindRoundModeChanged {
			t.Fatalf("expected Kind roundModeChanged, got %q", got[0].Kind)
		}
		var gotPayload, wantPayload map[string]string
		if err := json.Unmarshal(got[0].Payload, &gotPayload); err != nil {
			t.Fatalf("unmarshal got payload: %v", err)
		}
		if err := json.Unmarshal(first.Payload, &wantPayload); err != nil {
			t.Fatalf("unmarshal want payload: %v", err)
		}
		if gotPayload["from"] != wantPayload["from"] || gotPayload["to"] != wantPayload["to"] {
			t.Fatalf("payload mismatch: got %v want %v", gotPayload, wantPayload)
		}
	})
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return parsed
}
