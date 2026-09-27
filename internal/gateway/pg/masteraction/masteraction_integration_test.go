//go:build integration

package pgmasteraction_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	pgmasteraction "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestMasterAction(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgmasteraction.NewRepository(pool)
	ctx := context.Background()

	setup := func(t *testing.T) (masterUUID, matchUUID, sceneUUID, roundUUID uuid.UUID) {
		t.Helper()
		masterStr := pgtest.InsertTestUser(t, pool, "gm-ma", "gm-ma@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "MA Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "MA Match")
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Free")
		return uuid.MustParse(masterStr), uuid.MustParse(matchStr), uuid.MustParse(sceneStr), uuid.MustParse(roundStr)
	}

	t.Run("insert with nil and non-nil turn_uuid, real master, views for two players — lists by happened_at", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID, matchUUID, sceneUUID, roundUUID := setup(t)

		player1 := uuid.New()
		player2 := uuid.New()

		moveContent, err := json.Marshal(masteraction.PieceContent{
			CharacterID: "char-1",
			PieceID:     "piece-1",
			From:        &[3]int{1, 2, 0},
			To:          &[3]int{3, 4, 0},
		})
		if err != nil {
			t.Fatalf("marshal content: %v", err)
		}

		turnUUID := uuid.New()

		withTurn := masteraction.Record{
			UUID:       uuid.New(),
			MatchUUID:  matchUUID,
			SceneUUID:  sceneUUID,
			RoundUUID:  roundUUID,
			MasterUUID: masterUUID,
			TurnUUID:   &turnUUID,
			Kind:       masteraction.KindMovePiece,
			Content:    moveContent,
			Views: map[uuid.UUID]masteraction.View{
				player1: masteraction.ViewFull,
				player2: masteraction.ViewLeft,
			},
			HappenedAt: time.Date(2026, 9, 27, 10, 0, 1, 0, time.UTC),
		}
		withoutTurn := masteraction.Record{
			UUID:       uuid.New(),
			MatchUUID:  matchUUID,
			SceneUUID:  sceneUUID,
			RoundUUID:  roundUUID,
			MasterUUID: masterUUID,
			TurnUUID:   nil,
			Kind:       masteraction.KindTurnNote,
			Content:    json.RawMessage(`{"note":"outside a turn"}`),
			Views:      map[uuid.UUID]masteraction.View{},
			HappenedAt: time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC),
		}

		if err := repo.Insert(ctx, withTurn); err != nil {
			t.Fatalf("insert withTurn: %v", err)
		}
		if err := repo.Insert(ctx, withoutTurn); err != nil {
			t.Fatalf("insert withoutTurn: %v", err)
		}

		got, err := repo.ListByMatch(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListByMatch: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 records, got %d", len(got))
		}
		// happened_at order: withoutTurn (10:00:00) then withTurn (10:00:01)
		if got[0].UUID != withoutTurn.UUID || got[1].UUID != withTurn.UUID {
			t.Fatalf("expected order withoutTurn,withTurn by happened_at, got %v", []uuid.UUID{got[0].UUID, got[1].UUID})
		}

		if got[0].TurnUUID != nil {
			t.Fatalf("expected nil TurnUUID for withoutTurn, got %v", got[0].TurnUUID)
		}
		if got[1].TurnUUID == nil || *got[1].TurnUUID != turnUUID {
			t.Fatalf("expected TurnUUID %s for withTurn, got %v", turnUUID, got[1].TurnUUID)
		}

		if len(got[1].Views) != 2 {
			t.Fatalf("expected 2 views on withTurn, got %d (%v)", len(got[1].Views), got[1].Views)
		}
		v1, ok := got[1].ViewFor(player1)
		if !ok || v1 != masteraction.ViewFull {
			t.Fatalf("player1 view: got (%q,%v) want (full,true)", v1, ok)
		}
		v2, ok := got[1].ViewFor(player2)
		if !ok || v2 != masteraction.ViewLeft {
			t.Fatalf("player2 view: got (%q,%v) want (left,true)", v2, ok)
		}

		var gotContent, wantContent masteraction.PieceContent
		if err := json.Unmarshal(got[1].Content, &gotContent); err != nil {
			t.Fatalf("unmarshal got content: %v", err)
		}
		if err := json.Unmarshal(moveContent, &wantContent); err != nil {
			t.Fatalf("unmarshal want content: %v", err)
		}
		if gotContent.CharacterID != wantContent.CharacterID || gotContent.PieceID != wantContent.PieceID ||
			gotContent.From == nil || wantContent.From == nil || *gotContent.From != *wantContent.From ||
			gotContent.To == nil || wantContent.To == nil || *gotContent.To != *wantContent.To {
			t.Fatalf("content mismatch: got %+v want %+v", gotContent, wantContent)
		}

		if len(got[0].Views) != 0 {
			t.Fatalf("expected 0 views on withoutTurn (turnNote never reaches the table), got %v", got[0].Views)
		}
	})

	t.Run("master_uuid that is not a user is a FK error", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		_, matchUUID, sceneUUID, roundUUID := setup(t)

		rec := masteraction.Record{
			UUID:       uuid.New(),
			MatchUUID:  matchUUID,
			SceneUUID:  sceneUUID,
			RoundUUID:  roundUUID,
			MasterUUID: uuid.New(), // not a real user
			Kind:       masteraction.KindTurnNote,
			Content:    json.RawMessage(`{}`),
			HappenedAt: time.Now(),
		}

		err := repo.Insert(ctx, rec)
		if err == nil {
			t.Fatalf("expected an FK violation error for a non-existent master_uuid")
		}
		if !strings.Contains(err.Error(), "23503") && !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			t.Fatalf("expected a foreign key violation error, got: %v", err)
		}
	})
}
