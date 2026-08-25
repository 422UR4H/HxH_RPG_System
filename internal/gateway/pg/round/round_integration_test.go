//go:build integration

package round_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: r, Turn: tRn, Action: act, MatchUUID: matchUUIDParsed,
		})
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

		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: r, Turn: tRn, Action: act, MatchUUID: matchUUIDParsed,
		}); err != nil {
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
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: r, Turn: tRn1, Action: act1, MatchUUID: matchUUIDParsed,
		}); err != nil {
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
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: r, Turn: tRn2, Action: act2, MatchUUID: matchUUIDParsed,
		}); err != nil {
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

// resolutionFixture is a match, a master, and two character sheets to attack and be
// attacked — everything TestPersistTurnCloseWritesTheSettledResolution and its neighbour
// need to build a turn whose action targets a real FK-satisfying victim.
type resolutionFixture struct {
	matchUUID                  uuid.UUID
	masterUUID                 uuid.UUID
	scene                      *sceneentity.Scene
	round                      *roundentity.Round
	attackerSheet, victimSheet uuid.UUID
}

func seedMatchAndSheets(t *testing.T, pool *pgxpool.Pool) resolutionFixture {
	t.Helper()
	masterUUID := pgtest.InsertTestUser(t, pool, "gm-resolution", "gm-resolution@test.com", "pass")
	masterUUIDParsed, err := uuid.Parse(masterUUID)
	if err != nil {
		t.Fatalf("parse master uuid: %v", err)
	}
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "CampResolution")
	matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "MatchResolution")
	matchUUIDParsed, err := uuid.Parse(matchUUID)
	if err != nil {
		t.Fatalf("parse match uuid: %v", err)
	}
	attackerUUID, err := uuid.Parse(
		pgtest.InsertTestCharacterSheet(t, pool, &masterUUID, nil, &campaignUUID, "attacker"))
	if err != nil {
		t.Fatalf("parse attacker uuid: %v", err)
	}
	// An NPC victim: a sheet with no player, same as the reaction test above.
	victimUUID, err := uuid.Parse(
		pgtest.InsertTestCharacterSheet(t, pool, nil, &masterUUID, &campaignUUID, "victim"))
	if err != nil {
		t.Fatalf("parse victim uuid: %v", err)
	}
	return resolutionFixture{
		matchUUID:     matchUUIDParsed,
		masterUUID:    masterUUIDParsed,
		scene:         sceneentity.NewScene(enum.Battle, "Arena"),
		round:         roundentity.NewRound(enum.Free),
		attackerSheet: attackerUUID,
		victimSheet:   victimUUID,
	}
}

func buildAttackAction(t *testing.T, attackerID, victimID uuid.UUID) *action.Action {
	t.Helper()
	return action.NewAction(
		attackerID, []uuid.UUID{victimID}, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil,
	)
}

func TestPersistTurnCloseWritesTheSettledResolution(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	// hitOutcome/dodgeOutcome/defenseOutcome carry real, distinct non-zero values in every
	// field — not just the flat damage numbers — because the round trip below has to prove
	// the derived roll math (Bias/Modifier/Passive/DiceTotal) survives, not merely that the
	// JSONB column is non-NULL.
	hitOutcome := service.RollOutcome{
		SkillName: "Strength", SkillValue: 5, Dice: []int{6, 4}, DiceTotal: 10,
		Bias: 1, Modifier: 2, Total: 13,
	}
	dodgeOutcome := service.RollOutcome{
		SkillName: "Legerity", SkillValue: 4, Dice: []int{3, 2}, DiceTotal: 5,
		Bias: -1, Passive: true, Total: 4,
	}
	defenseOutcome := service.RollOutcome{
		SkillName: "Fortitude", SkillValue: 2, DiceTotal: 0, Modifier: 1, Passive: true, Total: 3,
	}
	// The payout's scope is ScopeOnly, not ScopeAnyone: the earlier version of this test only
	// ever drove ScopeAnyone through the real persistence path, leaving ScopeOnly/ScopeAllBut
	// covered at the unit level only.
	payoutScope := match.ScopeOnly(fx.attackerSheet)

	res := &service.TurnResolution{
		IsSettled:    true,
		ActionResult: service.RollResult{SkillName: "Legerity", Total: 19, DiceRolled: []int{10, 9}},
		CharacterResults: []service.CharacterResult{{
			TargetID: fx.victimSheet, RawDamage: 11, DefenseApplied: 3, EffectiveDamage: 8,
			ReactionKind: string(action.ReactRepel),
			Hit:          hitOutcome, Dodge: dodgeOutcome, Defense: defenseOutcome,
			Ladder: service.LadderOutcome{Rung: service.RungNearMiss, Margin: -4, Difference: 4},
			Payouts: []match.Modifier{{
				Amount: -4, Applies: match.DimActionSpeed, Source: match.SourceSystem,
				Against: payoutScope, ExpiresAt: match.LifetimeEndOfRound,
				Reason: "parry penalty",
			}},
		}},
		WallResults: []service.WallResult{{
			UpdatedWall:     mapentity.WallSegment{ID: "wall-north-1"},
			EffectiveDamage: 7, ReboundDamage: 2, Kind: service.WallResultKindAttack,
		}},
	}

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Resolution: res,
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("turns.resolution is NULL — the collision was not persisted")
	}

	got := roundrepo.DecodeResolution(raw)
	if got == nil || len(got.CharacterResults) != 1 {
		t.Fatalf("round trip lost the character results: %+v", got)
	}
	cr := got.CharacterResults[0]
	if cr.EffectiveDamage != 8 || cr.RawDamage != 11 {
		t.Fatalf("damage did not survive: raw=%d effective=%d", cr.RawDamage, cr.EffectiveDamage)
	}
	if cr.Ladder.Rung != service.RungNearMiss || cr.Ladder.Difference != 4 {
		t.Fatalf("the ladder did not survive: %+v", cr.Ladder)
	}
	if len(cr.Payouts) != 1 {
		t.Fatalf("expected 1 payout, got %+v", cr.Payouts)
	}
	// A non-anyone scope, through the real persistence path: Kind AND ID both have to
	// survive, or a ScopeOnly modifier would read back applying to nobody or to everybody.
	if cr.Payouts[0].Against.Kind() != payoutScope.Kind() || cr.Payouts[0].Against.ID() != fx.attackerSheet {
		t.Fatalf("the payout's non-anyone scope did not survive: %+v", cr.Payouts[0].Against)
	}
	// RollOutcome carries a []int (Dice), so it is not comparable with == — reflect.DeepEqual
	// is the straightforward way to assert every field round-tripped, dice slice included.
	if !reflect.DeepEqual(cr.Hit, hitOutcome) {
		t.Fatalf("the hit roll's derived math did not survive: got %+v, want %+v", cr.Hit, hitOutcome)
	}
	if !reflect.DeepEqual(cr.Dodge, dodgeOutcome) {
		t.Fatalf("the dodge roll's derived math did not survive: got %+v, want %+v", cr.Dodge, dodgeOutcome)
	}
	if !reflect.DeepEqual(cr.Defense, defenseOutcome) {
		t.Fatalf("the defense roll's derived math did not survive: got %+v, want %+v", cr.Defense, defenseOutcome)
	}
	if len(got.WallResults) != 1 {
		t.Fatalf("expected 1 wall result, got %+v", got.WallResults)
	}
	wr := got.WallResults[0]
	if wr.UpdatedWall.ID != "wall-north-1" || wr.EffectiveDamage != 7 || wr.ReboundDamage != 2 ||
		wr.Kind != service.WallResultKindAttack {
		t.Fatalf("the wall result did not survive: %+v", wr)
	}
}

func TestPersistTurnCloseAcceptsANilResolution(t *testing.T) {
	// A turn with nothing resolvable still closes. NULL, not an error, and not a zero-value
	// record that would read back as "a collision that produced nothing".
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Resolution: nil,
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if raw != nil {
		t.Fatalf("expected SQL NULL for a nil resolution, got %d bytes", len(raw))
	}
}

func TestPersistTurnCloseWritesOverriddenValues(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Overrides: []match.OverriddenValue{{
			ActionID: act.GetID(), Field: "skills", Origin: match.OriginPlayer,
			MasterUUID: fx.masterUUID, At: time.Now(),
			Original: []action.Skill{{SkillName: "Acrobatics"}},
		}},
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM overridden_action_values WHERE action_uuid = $1`,
		act.GetID()).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("wrote %d rows, want 1 — one row per field", n)
	}

	// A nil Original must land as SQL NULL — "there was no value" — not the JSON string
	// 'null', which would claim there was a value and it was null.
	var isNull bool
	if err := pool.QueryRow(ctx,
		`SELECT original_value IS NULL FROM overridden_action_values
		 WHERE action_uuid = $1 AND field = 'skills'`,
		act.GetID()).Scan(&isNull); err != nil {
		t.Fatalf("read original_value: %v", err)
	}
	if isNull {
		t.Fatal("original_value is NULL, want the marshaled []action.Skill")
	}
}

func TestPersistTurnCloseWritesNoRowForAnUneditedTurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Overrides: nil,
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM overridden_action_values WHERE action_uuid = $1`,
		act.GetID()).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("wrote %d rows, want 0 — a turn the master never touched leaves no trace", n)
	}
}

func TestPersistTurnCloseWritesANullOriginalAsSQLNull(t *testing.T) {
	// The commonest capture: a RollCondition the player never sent. Original is a bare nil
	// (no type at all), and the honest row says NULL, not the JSON string 'null'.
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Overrides: []match.OverriddenValue{{
			ActionID: act.GetID(), Field: "rollCondition", Origin: match.OriginSystem,
			MasterUUID: fx.masterUUID, At: time.Now(), Original: nil,
		}},
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var isNull bool
	if err := pool.QueryRow(ctx,
		`SELECT original_value IS NULL FROM overridden_action_values
		 WHERE action_uuid = $1 AND field = 'rollCondition'`,
		act.GetID()).Scan(&isNull); err != nil {
		t.Fatalf("read original_value: %v", err)
	}
	if !isNull {
		t.Fatal("original_value is not SQL NULL for a nil Original")
	}
}

// TestFindMatchHistoryIsNested proves the read side of Task 11: a match's closed turns come
// back as Scene -> Round -> Turn -> Action, not a flat list, with reactions attached to the
// right turn and the settled resolution round-tripped alongside it.
func TestFindMatchHistoryIsNested(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	// Turn 1: a plain action, no reaction, no resolution — closes first.
	act1 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn1 := turnentity.NewTurn(*act1)
	tn1.Close(time.Now())
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn1, Action: act1, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 1: %v", err)
	}

	// Turn 2: same scene, same round — a reaction and a settled resolution, closes after.
	act2 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	reaction := action.NewAction(
		fx.victimSheet, nil, act2.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	reaction.ReactionKind = action.ReactRepel
	reaction.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: "Sword", Result: 17}}

	tn2 := turnentity.NewTurn(*act2)
	tn2.AddReaction(reaction)
	tn2.Close(time.Now().Add(time.Second)) // strictly after tn1, so ordering is provable

	res := &service.TurnResolution{
		IsSettled:    true,
		ActionResult: service.RollResult{SkillName: "Legerity", Total: 19, DiceRolled: []int{10, 9}},
		CharacterResults: []service.CharacterResult{{
			TargetID: fx.victimSheet, RawDamage: 11, DefenseApplied: 3, EffectiveDamage: 8,
		}},
	}
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn2, Action: act2,
		MatchUUID: fx.matchUUID, Resolution: res,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 2: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 || len(scenes[0].Rounds[0].Turns) != 2 {
		t.Fatalf("the tree is wrong: %d scenes", len(scenes))
	}
	turns := scenes[0].Rounds[0].Turns
	if turns[0].FinishedAt.After(turns[1].FinishedAt) {
		t.Fatal("turns came back out of order")
	}
	withReaction := turns[1]
	if len(withReaction.Reactions) != 1 {
		t.Fatalf("reactions = %d, want 1 — they are persisted since PR #69",
			len(withReaction.Reactions))
	}
	if withReaction.Reactions[0].ReactionKind == "" {
		t.Fatal("reaction_kind did not come back")
	}
	if withReaction.Resolution == nil || len(withReaction.Resolution.CharacterResults) == 0 {
		t.Fatal("the settled resolution did not come back")
	}
	// Not just "non-nil" — a real value inside it, proving the round trip, not just presence.
	if withReaction.Resolution.CharacterResults[0].EffectiveDamage != 8 {
		t.Fatalf("resolution's character result did not survive: %+v",
			withReaction.Resolution.CharacterResults[0])
	}
	// Turn 1 resolved nothing — must come back nil, not a zero-value record.
	if turns[0].Resolution != nil {
		t.Fatalf("turn 1 should have no resolution, got %+v", turns[0].Resolution)
	}
}

// TestFindMatchHistoryOfAMatchWithNoTurns proves the empty case: an empty slice, not an error
// and not nil-that-marshals-to-null — Task 12 will put this straight on the wire.
func TestFindMatchHistoryOfAMatchWithNoTurns(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)

	masterUUID := pgtest.InsertTestUser(t, pool, "gm-empty", "gm-empty@test.com", "pass")
	campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "CampEmpty")
	matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "MatchEmpty")
	matchUUIDParsed, err := uuid.Parse(matchUUID)
	if err != nil {
		t.Fatalf("parse match uuid: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, matchUUIDParsed)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if scenes == nil {
		t.Fatal("expected a non-nil empty slice, got nil")
	}
	if len(scenes) != 0 {
		t.Fatalf("expected 0 scenes for a match with no closed turns, got %d", len(scenes))
	}
}

func TestPersistTurnCloseOverridesUniqueConstraintKeepsOneRowPerField(t *testing.T) {
	// Two captures naming the same (action_uuid, field) — the unique constraint plus ON
	// CONFLICT DO NOTHING must keep exactly one row, not error and not duplicate.
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Overrides: []match.OverriddenValue{
			{
				ActionID: act.GetID(), Field: "skills", Origin: match.OriginPlayer,
				MasterUUID: fx.masterUUID, At: time.Now(),
				Original: []action.Skill{{SkillName: "Acrobatics"}},
			},
			{
				ActionID: act.GetID(), Field: "skills", Origin: match.OriginPlayer,
				MasterUUID: fx.masterUUID, At: time.Now(),
				Original: []action.Skill{{SkillName: "Persuasion"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM overridden_action_values WHERE action_uuid = $1 AND field = 'skills'`,
		act.GetID()).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("wrote %d rows for the same (action_uuid, field), want 1", n)
	}
}
