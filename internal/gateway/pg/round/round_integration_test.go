//go:build integration

package round_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
	"github.com/google/uuid"
)

func TestPersistTurnClose(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path — persists scene, round, turn, action atomically", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)
		// The actor is the CHARACTER SHEET, not the player — that is what Action.actorID has
		// carried since phase 2, and what a real match hands this repository.
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &masterUUID, nil, &campaignUUID, "hero1")
		actorUUIDParsed, _ := uuid.Parse(sheetUUID)

		sc := sceneentity.NewScene(enum.Battle, "Arena")
		r := roundentity.NewRound(enum.Free)

		act := action.NewAction(
			actorUUIDParsed,
			nil,
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil,
		)
		actCopy := *act
		tRn := turnentity.NewTurn(actCopy)
		tRn.Close(time.Now())

		err := repo.PersistTurnClose(ctx, sc, r, tRn, act, matchUUIDParsed)
		if err != nil {
			t.Fatalf("PersistTurnClose error: %v", err)
		}

		var sceneCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes WHERE uuid = $1`, sc.GetID()).Scan(&sceneCount) //nolint:errcheck
		if sceneCount != 1 {
			t.Errorf("expected 1 scene row, got %d", sceneCount)
		}

		var roundCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM rounds WHERE uuid = $1`, r.GetID()).Scan(&roundCount) //nolint:errcheck
		if roundCount != 1 {
			t.Errorf("expected 1 round row, got %d", roundCount)
		}

		var turnCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM turns WHERE uuid = $1`, tRn.GetID()).Scan(&turnCount) //nolint:errcheck
		if turnCount != 1 {
			t.Errorf("expected 1 turn row, got %d", turnCount)
		}

		var actionCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM actions WHERE actor_uuid = $1`, actorUUIDParsed).Scan(&actionCount) //nolint:errcheck
		if actionCount != 1 {
			t.Errorf("expected 1 action row, got %d", actionCount)
		}
	})

	t.Run("persists the turn's reactions, with their declared kind", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm3", "gm3@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp3")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match3")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)

		attackerUUID, _ := uuid.Parse(
			pgtest.InsertTestCharacterSheet(t, pool, &masterUUID, nil, &campaignUUID, "attacker"))
		// An NPC: a sheet with no player. The FK has to accept it, or the master could never
		// persist a turn of theirs.
		npcUUID, _ := uuid.Parse(
			pgtest.InsertTestCharacterSheet(t, pool, nil, &masterUUID, &campaignUUID, "npc"))

		sc := sceneentity.NewScene(enum.Battle, "Arena")
		r := roundentity.NewRound(enum.Race)

		act := action.NewAction(
			attackerUUID, []uuid.UUID{npcUUID}, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil,
		)

		// The NPC repels. Its kind is declared; nothing about the shape says "repel".
		repel := action.NewAction(
			npcUUID, nil, act.GetID(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil,
		)
		repel.ReactionKind = action.ReactRepel
		repel.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: "Sword", Result: 17}}

		actCopy := *act
		tRn := turnentity.NewTurn(actCopy)
		tRn.AddReaction(repel)
		tRn.Close(time.Now())

		if err := repo.PersistTurnClose(ctx, sc, r, tRn, act, matchUUIDParsed); err != nil {
			t.Fatalf("PersistTurnClose error: %v", err)
		}

		var rowCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM actions WHERE turn_uuid = $1`, tRn.GetID()).Scan(&rowCount) //nolint:errcheck
		if rowCount != 2 {
			t.Fatalf("expected the action and its reaction, got %d rows", rowCount)
		}

		var kind, rowType string
		var reactTo uuid.UUID
		var repelJSON []byte
		err := pool.QueryRow(ctx,
			`SELECT type, reaction_kind, react_to_uuid, repel FROM actions WHERE uuid = $1`,
			repel.GetID(),
		).Scan(&rowType, &kind, &reactTo, &repelJSON)
		if err != nil {
			t.Fatalf("reading the reaction row: %v", err)
		}
		if rowType != "reaction" {
			t.Errorf("expected type %q, got %q", "reaction", rowType)
		}
		if kind != string(action.ReactRepel) {
			t.Errorf("expected reaction_kind %q, got %q", action.ReactRepel, kind)
		}
		if reactTo != act.GetID() {
			t.Errorf("expected react_to_uuid %s, got %s", act.GetID(), reactTo)
		}
		if len(repelJSON) == 0 {
			t.Error("expected the repel component to be persisted, got SQL NULL")
		}
	})

	t.Run("ON CONFLICT DO NOTHING — second call with same scene/round UUIDs is idempotent", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm2", "gm2@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp2")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match2")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &masterUUID, nil, &campaignUUID, "hero2")
		actorUUIDParsed, _ := uuid.Parse(sheetUUID)

		sc := sceneentity.NewScene(enum.Roleplay, "Inn")
		r := roundentity.NewRound(enum.Free)

		// First call
		act1 := action.NewAction(
			actorUUIDParsed,
			nil,
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil,
		)
		act1Copy := *act1
		tRn1 := turnentity.NewTurn(act1Copy)
		tRn1.Close(time.Now())
		if err := repo.PersistTurnClose(ctx, sc, r, tRn1, act1, matchUUIDParsed); err != nil {
			t.Fatalf("first PersistTurnClose error: %v", err)
		}

		// Second call with same scene/round UUIDs — only new turn+action should insert
		act2 := action.NewAction(
			actorUUIDParsed,
			nil,
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil,
		)
		act2Copy := *act2
		tRn2 := turnentity.NewTurn(act2Copy)
		tRn2.Close(time.Now())
		if err := repo.PersistTurnClose(ctx, sc, r, tRn2, act2, matchUUIDParsed); err != nil {
			t.Fatalf("second PersistTurnClose error: %v", err)
		}

		var sceneCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes WHERE uuid = $1`, sc.GetID()).Scan(&sceneCount) //nolint:errcheck
		if sceneCount != 1 {
			t.Errorf("expected exactly 1 scene row after two calls, got %d", sceneCount)
		}

		var roundCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM rounds WHERE uuid = $1`, r.GetID()).Scan(&roundCount) //nolint:errcheck
		if roundCount != 1 {
			t.Errorf("expected exactly 1 round row after two calls, got %d", roundCount)
		}

		var turnCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM turns`).Scan(&turnCount) //nolint:errcheck
		if turnCount != 2 {
			t.Errorf("expected 2 turn rows after two calls, got %d", turnCount)
		}
	})
}

func TestFindActiveSession(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("returns nil when no active session exists", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)

		data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil, got %+v", data)
		}
	})

	t.Run("returns active session when scene and round are open", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm2", "gm2@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp2")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match2")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)

		sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
		roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

		data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data == nil {
			t.Fatal("expected non-nil ActiveSessionData")
		}
		if data.SceneID.String() != sceneUUID {
			t.Errorf("expected SceneID %s, got %s", sceneUUID, data.SceneID)
		}
		if data.RoundID.String() != roundUUID {
			t.Errorf("expected RoundID %s, got %s", roundUUID, data.RoundID)
		}
		if data.Category != "Battle" {
			t.Errorf("expected category Battle, got %q", data.Category)
		}
		if data.Mode != "Free" {
			t.Errorf("expected mode Free, got %q", data.Mode)
		}
	})

	t.Run("ignores finished scenes", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm3", "gm3@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp3")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match3")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)

		sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Roleplay")
		pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

		// Close the scene
		pool.Exec(ctx, `UPDATE scenes SET finished_at = $1 WHERE uuid = $2`, time.Now(), sceneUUID) //nolint:errcheck

		data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil for finished scene, got %+v", data)
		}
	})
}

func TestCloseSceneAndRound(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path — sets finished_at on both scene and round", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")

		sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
		roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Free")

		sceneUUIDParsed, _ := uuid.Parse(sceneUUID)
		roundUUIDParsed, _ := uuid.Parse(roundUUID)
		at := time.Now().UTC().Truncate(time.Microsecond)

		err := repo.CloseSceneAndRound(ctx, sceneUUIDParsed, roundUUIDParsed, at)
		if err != nil {
			t.Fatalf("CloseSceneAndRound error: %v", err)
		}

		var sceneFinishedAt, roundFinishedAt time.Time
		pool.QueryRow(ctx, `SELECT finished_at FROM scenes WHERE uuid = $1`, sceneUUID).Scan(&sceneFinishedAt) //nolint:errcheck
		pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, roundUUID).Scan(&roundFinishedAt) //nolint:errcheck

		if !sceneFinishedAt.Truncate(time.Microsecond).Equal(at) {
			t.Errorf("scene finished_at: got %v, want %v", sceneFinishedAt, at)
		}
		if !roundFinishedAt.Truncate(time.Microsecond).Equal(at) {
			t.Errorf("round finished_at: got %v, want %v", roundFinishedAt, at)
		}
	})
}

func TestCloseRound(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path — sets finished_at on round only", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")

		sceneUUID := pgtest.InsertTestScene(t, pool, matchUUID, "Battle")
		roundUUID := pgtest.InsertTestRound(t, pool, sceneUUID, "Race")

		roundUUIDParsed, _ := uuid.Parse(roundUUID)
		at := time.Now().UTC().Truncate(time.Microsecond)

		err := repo.CloseRound(ctx, roundUUIDParsed, at)
		if err != nil {
			t.Fatalf("CloseRound error: %v", err)
		}

		var roundFinishedAt time.Time
		pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, roundUUID).Scan(&roundFinishedAt) //nolint:errcheck

		if !roundFinishedAt.Truncate(time.Microsecond).Equal(at) {
			t.Errorf("round finished_at: got %v, want %v", roundFinishedAt, at)
		}

		// Scene must NOT be closed
		var sceneFinishedAt *time.Time
		pool.QueryRow(ctx, `SELECT finished_at FROM scenes WHERE uuid = $1`, sceneUUID).Scan(&sceneFinishedAt) //nolint:errcheck
		if sceneFinishedAt != nil {
			t.Error("expected scene finished_at to remain NULL")
		}
	})
}
