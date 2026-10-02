//go:build integration

package round_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	pgmatchboard "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchboard"
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

	t.Run("ON CONFLICT upsert — second call with same scene/round UUIDs is idempotent", func(t *testing.T) {
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

// B13: an escape's verdict — and where the master put a failed one — is part of the settled
// resolution, so the history can say why a piece moved (or did not) a year later. Three
// shapes: no escape at all (nil must stay nil, not become a zero verdict), one that escaped,
// and one that failed with the master's landing.
func TestPersistTurnCloseKeepsTheEscape(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	landing := [3]int{7, 6, 0}
	noEscape, escaped, landed := uuid.New(), uuid.New(), uuid.New()
	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{
			{TargetID: noEscape},
			{TargetID: escaped, ReactionKind: string(action.ReactEscape),
				Escape: &service.EscapeResult{MovePassed: true, DodgePassed: true, Escaped: true}},
			{TargetID: landed, ReactionKind: string(action.ReactEscapeGuard),
				Escape: &service.EscapeResult{DodgePassed: true, Landing: &landing}},
		},
	}
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Resolution: res,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := roundrepo.DecodeResolution(raw)
	if got == nil || len(got.CharacterResults) != 3 {
		t.Fatalf("round trip lost the character results: %+v", got)
	}
	byTarget := map[uuid.UUID]service.CharacterResult{}
	for _, cr := range got.CharacterResults {
		byTarget[cr.TargetID] = cr
	}
	if e := byTarget[noEscape].Escape; e != nil {
		t.Fatalf("a result with no escape read back with one: %+v", *e)
	}
	if e := byTarget[escaped].Escape; e == nil || *e != (service.EscapeResult{MovePassed: true, DodgePassed: true, Escaped: true}) {
		t.Fatalf("the escaped verdict did not survive: %+v", e)
	}
	e := byTarget[landed].Escape
	if e == nil || e.Escaped || e.MovePassed || !e.DodgePassed || e.Landing == nil || *e.Landing != landing {
		t.Fatalf("the failed escape and its landing did not survive: %+v", e)
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

// TestPersistTurnCloseWritesTheTurnsMasterActions: what the master did inside an open turn is
// written in the SAME transaction as the turn (owner decision, 2026-10-01) — both land, or
// neither does. A master action that fails to insert rolls the turn back with it.
func TestPersistTurnCloseWritesTheTurnsMasterActions(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	build := func(t *testing.T, fx resolutionFixture, tn *turnentity.Turn, masterUUID uuid.UUID) masteraction.Record {
		t.Helper()
		turnID := tn.GetID()
		return masteraction.Record{
			UUID: uuid.New(), MatchUUID: fx.matchUUID, SceneUUID: fx.scene.GetID(),
			RoundUUID: fx.round.GetID(), TurnUUID: &turnID, MasterUUID: masterUUID,
			Kind: masteraction.KindMovePiece, Content: []byte(`{"characterId":"c","pieceId":"p"}`),
			Views:      map[uuid.UUID]masteraction.View{uuid.New(): masteraction.ViewFull},
			HappenedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
	}
	countOf := func(t *testing.T, q string, id uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, q, id).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	t.Run("writes the turn and its master actions together", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(time.Now())
		recs := []masteraction.Record{build(t, fx, tn, fx.masterUUID), build(t, fx, tn, fx.masterUUID)}

		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
			MatchUUID: fx.matchUUID, MasterActions: recs,
		}); err != nil {
			t.Fatalf("PersistTurnClose: %v", err)
		}
		if n := countOf(t, `SELECT COUNT(*) FROM master_actions WHERE turn_uuid = $1`, tn.GetID()); n != 2 {
			t.Fatalf("master_actions rows for the turn = %d, want 2", n)
		}
		var views []byte
		if err := pool.QueryRow(ctx, `SELECT views FROM master_actions WHERE uuid = $1`, recs[0].UUID).Scan(&views); err != nil {
			t.Fatalf("read views: %v", err)
		}
		if len(views) <= 2 {
			t.Fatalf("views = %s, want what the record carried", views)
		}
	})

	t.Run("a master action that fails to insert rolls the turn back", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(time.Now())
		// A master who is not a user: master_actions.master_uuid's FK refuses it.
		bad := build(t, fx, tn, uuid.New())

		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
			MatchUUID: fx.matchUUID, MasterActions: []masteraction.Record{build(t, fx, tn, fx.masterUUID), bad},
		})
		if err == nil {
			t.Fatal("PersistTurnClose succeeded with a master action the database refuses")
		}
		if n := countOf(t, `SELECT COUNT(*) FROM turns WHERE uuid = $1`, tn.GetID()); n != 0 {
			t.Fatalf("turns rows = %d, want 0 — the turn must not outlive its master actions' failure", n)
		}
		if n := countOf(t, `SELECT COUNT(*) FROM master_actions WHERE turn_uuid = $1`, tn.GetID()); n != 0 {
			t.Fatalf("master_actions rows = %d, want 0 — the good one must roll back too", n)
		}
	})
}

// TestPersistTurnCloseWritesTheBoardWithTheTurn: the board the close left, and every player's
// fog memory, are written in the turn's own transaction — a failure anywhere in it (here, a
// master action the database refuses) leaves the board row and the memories as they were.
func TestPersistTurnCloseWritesTheBoardWithTheTurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	setup := func(t *testing.T) (resolutionFixture, uuid.UUID, *turnentity.Turn, *action.Action) {
		t.Helper()
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		var campaignUUID string
		if err := pool.QueryRow(ctx, `SELECT campaign_uuid FROM matches WHERE uuid = $1`, fx.matchUUID).Scan(&campaignUUID); err != nil {
			t.Fatalf("read campaign: %v", err)
		}
		mapUUID := uuid.MustParse(pgtest.InsertTestMap(t, pool, campaignUUID, "Board map"))
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(time.Now())
		return fx, mapUUID, tn, act
	}
	boardOf := func(fx resolutionFixture, mapUUID uuid.UUID, col int) *matchboard.Board {
		return &matchboard.Board{
			MatchUUID: fx.matchUUID, MapUUID: mapUUID,
			Grid: mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1},
			Pieces: []mapentity.Piece{{
				ID: "p1", CharacterID: fx.attackerSheet.String(), Visible: true,
				Coord: mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: col, Row: 1}},
			}},
		}
	}
	memoryOf := func(fx resolutionFixture, mapUUID uuid.UUID) fogentity.PlayerMemory {
		return fogentity.PlayerMemory{
			MatchID: fx.matchUUID, MapID: mapUUID, PlayerID: fx.masterUUID,
			Seen: map[fogentity.FeatureRef]struct{}{{Kind: fogentity.FeatureKind("wall"), ID: "w1"}: {}},
		}
	}
	piecesOf := func(t *testing.T, matchUUID uuid.UUID) string {
		t.Helper()
		var pieces string
		if err := pool.QueryRow(ctx, `SELECT pieces::text FROM match_boards WHERE match_uuid = $1`, matchUUID).Scan(&pieces); err != nil {
			t.Fatalf("read board: %v", err)
		}
		return pieces
	}
	memoriesOf := func(t *testing.T, matchUUID uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM player_memories WHERE match_id = $1`, matchUUID).Scan(&n); err != nil {
			t.Fatalf("count memories: %v", err)
		}
		return n
	}

	t.Run("writes the board and the memories with the turn", func(t *testing.T) {
		fx, mapUUID, tn, act := setup(t)
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
			Board: boardOf(fx, mapUUID, 7), Memories: []fogentity.PlayerMemory{memoryOf(fx, mapUUID)},
		}); err != nil {
			t.Fatalf("PersistTurnClose: %v", err)
		}
		if p := piecesOf(t, fx.matchUUID); !strings.Contains(p, `"col": 7`) {
			t.Fatalf("board pieces = %s, want the close's piece at col 7", p)
		}
		if n := memoriesOf(t, fx.matchUUID); n != 1 {
			t.Fatalf("player_memories rows = %d, want 1", n)
		}
	})

	t.Run("a failing master action rolls the board back with the turn", func(t *testing.T) {
		fx, mapUUID, tn, act := setup(t)
		// The board as the LAST close left it.
		if err := pgmatchboard.NewRepository(pool).Save(ctx, boardOf(fx, mapUUID, 2)); err != nil {
			t.Fatalf("seed board: %v", err)
		}
		turnID := tn.GetID()
		bad := masteraction.Record{
			UUID: uuid.New(), MatchUUID: fx.matchUUID, SceneUUID: fx.scene.GetID(), RoundUUID: fx.round.GetID(),
			TurnUUID: &turnID, MasterUUID: uuid.New(), // not a user: the FK refuses it
			Kind: masteraction.KindMovePiece, Content: []byte(`{}`), HappenedAt: time.Now().UTC(),
		}
		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
			MasterActions: []masteraction.Record{bad},
			Board:         boardOf(fx, mapUUID, 7), Memories: []fogentity.PlayerMemory{memoryOf(fx, mapUUID)},
		})
		if err == nil {
			t.Fatal("PersistTurnClose succeeded with a master action the database refuses")
		}
		if p := piecesOf(t, fx.matchUUID); !strings.Contains(p, `"col": 2`) {
			t.Fatalf("board pieces = %s, want the last close's col 2 — the board must roll back with the turn", p)
		}
		if n := memoriesOf(t, fx.matchUUID); n != 0 {
			t.Fatalf("player_memories rows = %d, want 0 — the memories roll back too", n)
		}
	})
}

// TestPersistTurnCloseWritesTheStatusBarsWithTheTurn: the HP a close applied is written in the
// turn's own transaction (owner decision, 2026-10-02) — one master command, one transaction. A
// failure anywhere in it (here, a master action the database refuses, written after the bars)
// leaves the sheet as the last close did.
func TestPersistTurnCloseWritesTheStatusBarsWithTheTurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	barsOf := func(id uuid.UUID, hp int) appmatch.SheetStatusBars {
		return appmatch.SheetStatusBars{
			CharacterID: id,
			Health:      status.ReconstructBar(0, hp, 20),
			Stamina:     status.ReconstructBar(0, 9, 10),
			Aura:        status.ReconstructBar(0, 4, 5),
		}
	}
	healthOf := func(t *testing.T, id uuid.UUID) (curr, max, stamina, aura int) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			// A seeded sheet has no bars yet (NULL): -1 stands for "never written".
			`SELECT COALESCE(health_curr_pts, -1), COALESCE(health_max_pts, -1),
			        COALESCE(stamina_curr_pts, -1), COALESCE(aura_curr_pts, -1)
			 FROM character_sheets WHERE uuid = $1`, id,
		).Scan(&curr, &max, &stamina, &aura); err != nil {
			t.Fatalf("read sheet: %v", err)
		}
		return curr, max, stamina, aura
	}

	t.Run("writes the damaged sheet's bars with the turn", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(time.Now())

		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
			StatusBars: []appmatch.SheetStatusBars{barsOf(fx.victimSheet, 13)},
		}); err != nil {
			t.Fatalf("PersistTurnClose: %v", err)
		}
		curr, max, stamina, aura := healthOf(t, fx.victimSheet)
		if curr != 13 || max != 20 || stamina != 9 || aura != 4 {
			t.Fatalf("victim bars = hp %d/%d, stamina %d, aura %d — want 13/20, 9, 4", curr, max, stamina, aura)
		}
	})

	t.Run("a failing master action rolls the sheet back with the turn", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		before, beforeMax, _, _ := healthOf(t, fx.victimSheet)
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(time.Now())
		turnID := tn.GetID()
		bad := masteraction.Record{
			UUID: uuid.New(), MatchUUID: fx.matchUUID, SceneUUID: fx.scene.GetID(), RoundUUID: fx.round.GetID(),
			TurnUUID: &turnID, MasterUUID: uuid.New(), // not a user: the FK refuses it
			Kind: masteraction.KindMovePiece, Content: []byte(`{}`), HappenedAt: time.Now().UTC(),
		}
		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
			StatusBars:    []appmatch.SheetStatusBars{barsOf(fx.victimSheet, 13)},
			MasterActions: []masteraction.Record{bad},
		})
		if err == nil {
			t.Fatal("PersistTurnClose succeeded with a master action the database refuses")
		}
		if curr, max, _, _ := healthOf(t, fx.victimSheet); curr != before || max != beforeMax {
			t.Fatalf("victim hp = %d/%d, want it untouched at %d/%d — the bars must roll back with the turn", curr, max, before, beforeMax)
		}
	})
}

// TestPersistTurnCloseWritesFinishedAtAsCreatedAt guards AGENTS.md's own known-issues entry:
// Turn has no createdAt field of its own, so persistence is documented as using finishedAt as
// the approximation. Persistence always happens strictly AFTER the turn resolves — a
// time.Now() taken here instead would put every turn's created_at strictly AFTER its
// finished_at, which is invisible until something sorts by created_at (Task 12's
// HistoryTurnResponse.CreatedAt) and then reads as nonsense.
func TestPersistTurnCloseWritesFinishedAtAsCreatedAt(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	finishedAt := time.Now().Truncate(time.Microsecond)
	tn.Close(finishedAt)

	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	// Both columns are read back from the DB and compared to EACH OTHER, not against the
	// in-memory finishedAt: turns.created_at/finished_at is a bare TIMESTAMP (no time zone),
	// so a value round-tripped through the driver and one that never left the process compare
	// unequal on offset alone even when they name the same wall-clock instant. Reading both
	// back cancels that out and asks the only question this test actually cares about: does
	// the row's created_at equal its own finished_at.
	var createdAt, dbFinishedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT created_at, finished_at FROM turns WHERE uuid = $1`, tn.GetID(),
	).Scan(&createdAt, &dbFinishedAt); err != nil {
		t.Fatalf("select created_at, finished_at: %v", err)
	}
	if !createdAt.Equal(dbFinishedAt) {
		t.Fatalf("turns.created_at = %v, want it to equal turns.finished_at %v — "+
			"a time.Now() taken at persist time is always strictly after the real close",
			createdAt, dbFinishedAt)
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
	// The two-sided claim: turn 1 got no reaction attached, and must come back with none —
	// not just "turn 2 has one", which alone would also pass if the two turns' rows had
	// interleaved and both ended up pointing at the same reaction.
	if len(turns[0].Reactions) != 0 {
		t.Fatalf("turn 1 has no reaction, want 0 Reactions, got %d", len(turns[0].Reactions))
	}
	withReaction := turns[1]
	if len(withReaction.Reactions) != 1 {
		t.Fatalf("reactions = %d, want 1 — they are persisted since PR #69",
			len(withReaction.Reactions))
	}
	if withReaction.Reactions[0].ReactionKind == "" {
		t.Fatal("reaction_kind did not come back")
	}
	// Internal consistency, not just a match against the fixture: the reaction's ReactToID is
	// read straight off the DB row, so if the action's own id is fabricated (a fresh
	// uuid.New() instead of the persisted actions.uuid) these two would silently disagree —
	// a tree that lies about its own shape, still passing a check that only looked at the
	// fixture in isolation.
	if withReaction.Action.GetID() != act2.GetID() {
		t.Fatalf("action id did not round-trip: got %s, want %s (the persisted uuid)",
			withReaction.Action.GetID(), act2.GetID())
	}
	if withReaction.Reactions[0].GetID() != reaction.GetID() {
		t.Fatalf("reaction id did not round-trip: got %s, want %s (the persisted uuid)",
			withReaction.Reactions[0].GetID(), reaction.GetID())
	}
	if withReaction.Reactions[0].ReactToID != withReaction.Action.GetID() {
		t.Fatalf("reaction's ReactToID (%s) does not match its own turn's action id (%s) — "+
			"the reaction answers an action that, by id, is not in the tree",
			withReaction.Reactions[0].ReactToID, withReaction.Action.GetID())
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

// TestFindMatchHistoryDoesNotInterleaveTurnsThatTieOnFinishedAt proves the ORDER BY's t.uuid
// tiebreaker earns its place. insertAction writes the SAME timestamp as both created_at and
// finished_at for a turn's action AND every one of its reactions, so two turns closed in the
// same instant tying on t.finished_at is the norm, not a corner case an unlucky clock might
// hit. Without t.uuid keeping each turn's rows contiguous, the assembly loop — which decides
// "this row starts a new turn" purely from "does turnUUID match the turn I'm building" — could
// see turn B's action land between turn A's action and turn A's reaction, and misclassify
// that reaction as turn B's own action.
func TestFindMatchHistoryDoesNotInterleaveTurnsThatTieOnFinishedAt(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	sharedFinish := time.Now() // deliberately identical for both turns — forces the tie

	act1 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn1 := turnentity.NewTurn(*act1)
	tn1.Close(sharedFinish)
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn1, Action: act1, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 1: %v", err)
	}

	act2 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	reaction := action.NewAction(
		fx.victimSheet, nil, act2.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	reaction.ReactionKind = action.ReactRepel
	tn2 := turnentity.NewTurn(*act2)
	tn2.AddReaction(reaction)
	tn2.Close(sharedFinish) // ties tn1's finished_at exactly
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn2, Action: act2, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 2: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 || len(scenes[0].Rounds[0].Turns) != 2 {
		t.Fatalf("the tree is wrong under a finished_at tie: %d scenes", len(scenes))
	}
	turns := scenes[0].Rounds[0].Turns
	for _, tn := range turns {
		// A misclassified reaction masquerading as a turn's own action would carry a non-nil
		// ReactToID — react_to_uuid IS NULL is exactly what discriminates "this is the turn's
		// own action" in the first place.
		if tn.Action.ReactToID != uuid.Nil {
			t.Fatalf("turn %s's Action is actually a reaction (ReactToID=%s) — "+
				"turns interleaved on the finished_at tie", tn.UUID, tn.Action.ReactToID)
		}
	}
	// A tie means arrival order between the two turns is not guaranteed (t.uuid is a random
	// value, not a clock), so find the reaction by summing across both rather than assuming
	// which index it landed on.
	var reactionsSeen int
	for _, tn := range turns {
		reactionsSeen += len(tn.Reactions)
	}
	if reactionsSeen != 1 {
		t.Fatalf("expected exactly 1 reaction across both tied turns, got %d", reactionsSeen)
	}
}

// TestFindMatchHistoryDoesNotInterleaveScenesOrRoundsThatTieOnCreatedAt is
// TestFindMatchHistoryDoesNotInterleaveTurnsThatTieOnFinishedAt's exact same defect one level
// up: the assembly groups a scene by "does the UUID still match the one being built"
// (curScene.UUID != sceneUUID) and a round the same way (curRound.UUID != roundUUID). Two
// scenes tied on created_at could interleave without s.uuid tiebreaking the ORDER BY — and
// scene A carries TWO turns here specifically so an interleave has something to split: if
// scene B's row lands between scene A's two turn-rows, scene A stops being contiguous and the
// assembly appends a THIRD HistoryScene entry (A, B, A again) instead of two.
// ReconstructScene/ReconstructRound let the fixture force the tie explicitly, unlike the
// turn-level test's naturally-tying insertAction timestamps.
func TestFindMatchHistoryDoesNotInterleaveScenesOrRoundsThatTieOnCreatedAt(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	sharedCreated := time.Now() // deliberately identical across both scenes AND both rounds

	sceneA := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Scene A", sharedCreated)
	sceneB := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Scene B", sharedCreated)
	roundA := roundentity.ReconstructRound(uuid.New(), enum.Free, sharedCreated)
	roundB := roundentity.ReconstructRound(uuid.New(), enum.Free, sharedCreated)

	// Scene A's first turn.
	actA1 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	turnA1 := turnentity.NewTurn(*actA1)
	turnA1.Close(sharedCreated)
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: sceneA, Round: roundA, Turn: turnA1, Action: actA1, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose scene A turn 1: %v", err)
	}

	// Scene B's only turn, sandwiched between scene A's two — a naive scan order would put
	// this row physically between turnA1's and turnA2's.
	actB := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	turnB := turnentity.NewTurn(*actB)
	turnB.Close(sharedCreated)
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: sceneB, Round: roundB, Turn: turnB, Action: actB, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose scene B: %v", err)
	}

	// Scene A's second turn, same scene and round as the first.
	actA2 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	turnA2 := turnentity.NewTurn(*actA2)
	turnA2.Close(sharedCreated)
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: sceneA, Round: roundA, Turn: turnA2, Action: actA2, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose scene A turn 2: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes under a created_at tie, got %d — scene A's rows split "+
			"across two entries instead of staying contiguous", len(scenes))
	}
	for _, sc := range scenes {
		if len(sc.Rounds) != 1 {
			t.Fatalf("scene %s has %d rounds, want exactly 1 — rounds interleaved on the tie",
				sc.UUID, len(sc.Rounds))
		}
	}
	var sceneAResult, sceneBResult *appmatch.HistoryScene
	for i := range scenes {
		switch scenes[i].UUID {
		case sceneA.GetID():
			sceneAResult = &scenes[i]
		case sceneB.GetID():
			sceneBResult = &scenes[i]
		}
	}
	if sceneAResult == nil || sceneBResult == nil {
		t.Fatalf("expected both scene A (%s) and scene B (%s) in the result, got %+v",
			sceneA.GetID(), sceneB.GetID(), scenes)
	}
	if len(sceneAResult.Rounds[0].Turns) != 2 {
		t.Fatalf("scene A's round has %d turns, want 2 — its rows split across two scene "+
			"entries", len(sceneAResult.Rounds[0].Turns))
	}
	if len(sceneBResult.Rounds[0].Turns) != 1 {
		t.Fatalf("scene B's round has %d turns, want 1", len(sceneBResult.Rounds[0].Turns))
	}
}

// TestFindMatchHistoryDropsATurnWhoseActionFailsToDecode proves the containment
// DecodeResolution already established for a bad stored resolution, now for a bad stored
// action: one unreadable JSONB blob must not take the whole match's history offline. The
// corruption here is on the turn's OWN action, so that turn can never even start being
// assembled — it must be entirely absent from the result, while every OTHER turn in the same
// match comes back untouched.
func TestFindMatchHistoryDropsATurnWhoseActionFailsToDecode(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	// Turn 1: healthy, closes first.
	act1 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn1 := turnentity.NewTurn(*act1)
	tn1.Close(time.Now())
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn1, Action: act1, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 1: %v", err)
	}

	// Turn 2: needs a non-nil Move so the column exists to corrupt afterwards — the zero-value
	// Move is enough, its content doesn't matter, only that the column is non-NULL going in.
	act2 := action.NewAction(
		fx.attackerSheet, []uuid.UUID{fx.victimSheet}, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{}, nil, nil, nil, nil, nil,
	)
	tn2 := turnentity.NewTurn(*act2)
	tn2.Close(time.Now().Add(time.Second))
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn2, Action: act2, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 2: %v", err)
	}

	// Corrupt turn 2's action: valid JSON — jsonb itself would reject a syntax error outright
	// — but the wrong SHAPE for action.Move (a bare string, not an object). This is what
	// schema drift or a future migration bug actually produces, not a syntax error.
	if _, err := pool.Exec(ctx,
		`UPDATE actions SET move = '"not-a-move-object"'::jsonb WHERE uuid = $1`, act2.GetID(),
	); err != nil {
		t.Fatalf("corrupting turn 2's action: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory must contain the bad row, not fail outright: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 {
		t.Fatalf("the healthy scene/round should still come back: %d scenes", len(scenes))
	}
	turns := scenes[0].Rounds[0].Turns
	if len(turns) != 1 {
		t.Fatalf("expected only the healthy turn 1 to survive, got %d turns", len(turns))
	}
	if turns[0].UUID != tn1.GetID() {
		t.Fatalf("the surviving turn is not turn 1: got %s, want %s", turns[0].UUID, tn1.GetID())
	}
}

// TestFindMatchHistoryDropsATurnWhoseReactionFailsToDecode proves the same containment when
// the corruption is on a REACTION instead of the turn's own action: the action row decodes
// fine and is provisionally appended, but the whole turn — not just the bad reaction — must
// come back out, or the history would silently under-report a reaction that really happened.
func TestFindMatchHistoryDropsATurnWhoseReactionFailsToDecode(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	// Turn 1: healthy, closes first.
	act1 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn1 := turnentity.NewTurn(*act1)
	tn1.Close(time.Now())
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn1, Action: act1, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 1: %v", err)
	}

	// Turn 2: a healthy action, a reaction whose Move column will be corrupted after
	// persisting.
	act2 := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	reaction := action.NewAction(
		fx.victimSheet, nil, act2.GetID(), nil, action.ActionSpeed{},
		nil, &action.Move{}, nil, nil, nil, nil, nil,
	)
	reaction.ReactionKind = action.ReactEscape
	tn2 := turnentity.NewTurn(*act2)
	tn2.AddReaction(reaction)
	tn2.Close(time.Now().Add(time.Second))
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn2, Action: act2, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 2: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE actions SET move = '"not-a-move-object"'::jsonb WHERE uuid = $1`, reaction.GetID(),
	); err != nil {
		t.Fatalf("corrupting turn 2's reaction: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory must contain the bad row, not fail outright: %v", err)
	}
	turns := scenes[0].Rounds[0].Turns
	if len(turns) != 1 {
		t.Fatalf("expected only the healthy turn 1 to survive, got %d turns", len(turns))
	}
	if turns[0].UUID != tn1.GetID() {
		t.Fatalf("the surviving turn is not turn 1: got %s, want %s", turns[0].UUID, tn1.GetID())
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

// TestPersistTurnCloseWritesTheResolutionsEngineFaults keeps the settled record honest about
// what it could NOT compute.
//
// A missing-sheet fault means a target the action aimed at produced no CharacterResult at
// all. Dropping the fault on the way to storage would make that turn read back, years later,
// as a turn that simply never targeted them — which is precisely the silence the fault was
// introduced to break. The record's own doc requires every omission from
// service.TurnResolution to be justified; this field is not omitted.
func TestPersistTurnCloseWritesTheResolutionsEngineFaults(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	ghost := uuid.New()
	res := &service.TurnResolution{
		IsSettled: true,
		Errors: []service.ResolutionError{
			{
				Subject: ghost,
				Kind:    service.ResolutionErrUnknownTarget,
				Detail:  "action target is neither a character nor a wall segment",
			},
			{
				Subject: fx.victimSheet,
				Kind:    service.ResolutionErrMissingSheet,
				Detail:  "the target character's sheet was not handed to the resolver",
			},
		},
	}

	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Resolution: res,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := roundrepo.DecodeResolution(raw)
	if got == nil {
		t.Fatal("the stored resolution did not decode")
	}
	if len(got.Errors) != 2 {
		t.Fatalf("Errors = %+v, want both faults", got.Errors)
	}
	if !reflect.DeepEqual(got.Errors, res.Errors) {
		t.Fatalf("the faults did not survive the round trip: got %+v, want %+v",
			got.Errors, res.Errors)
	}
}

// TestPersistTurnCloseRoundTripsInteractAndSystemBias covers the two pieces of an Action the
// actions table had no column for.
//
// Interact is the worse of the two: opening a door carries NO other payload, so a turn whose
// whole content was an interaction persisted as type "unspecified" with every JSONB column
// NULL — a row that says an action happened and refuses to say which.
//
// SystemBias is the engine-imposed advantage/disadvantage the action was charged under (see
// its doc on action.Action). Losing it costs the history the reason a number was that number,
// on a surface whose entire purpose is to let the table reconstruct the reasoning.
func TestPersistTurnCloseRoundTripsInteractAndSystemBias(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	doorID := uuid.New()
	act := action.NewAction(
		fx.attackerSheet, []uuid.UUID{doorID}, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil,
		&action.Interact{Kind: action.InteractOpen},
	)
	tn := turnentity.NewTurn(*act)

	// A reaction that displaced a queued action: the one thing that sets a non-zero
	// SystemBias today (MatchSession.AttachReaction's "swapping what you were going to do
	// costs Disadvantage").
	reaction := action.NewAction(
		fx.victimSheet, nil, act.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	reaction.ReactionKind = action.ReactRepel
	reaction.SystemBias = -1
	tn.AddReaction(reaction)
	tn.Close(time.Now())

	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var storedType string
	if err := pool.QueryRow(ctx,
		`SELECT type FROM actions WHERE uuid = $1`, act.GetID()).Scan(&storedType); err != nil {
		t.Fatalf("read back type: %v", err)
	}
	if storedType != "interact" {
		t.Errorf("actions.type = %q, want \"interact\" — an opened door is not unspecified",
			storedType)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 || len(scenes[0].Rounds[0].Turns) != 1 {
		t.Fatalf("the tree is wrong: %+v", scenes)
	}
	got := scenes[0].Rounds[0].Turns[0]

	if got.Action.Interact == nil {
		t.Fatal("the interaction did not come back — the history cannot say what was done")
	}
	if got.Action.Interact.Kind != action.InteractOpen {
		t.Errorf("Interact.Kind = %q, want %q", got.Action.Interact.Kind, action.InteractOpen)
	}
	if len(got.Reactions) != 1 {
		t.Fatalf("reactions = %d, want 1", len(got.Reactions))
	}
	if got.Reactions[0].SystemBias != -1 {
		t.Errorf("SystemBias = %d, want -1 — the Disadvantage the reaction was charged under",
			got.Reactions[0].SystemBias)
	}
	// The action itself was charged nothing, and must read back as nothing rather than
	// inheriting its reaction's bias.
	if got.Action.SystemBias != 0 {
		t.Errorf("the action's SystemBias = %d, want 0", got.Action.SystemBias)
	}
}

// TestFindMatchHistoryKeepsEachTiedTurnsReactionWithItsOwnTurn is the test PR #71 owed.
//
// The t.uuid tiebreaker was landed as "correct by the semantics of the SQL, not verified by a
// test": the existing tie test could not reproduce the defect, because only ONE of its two
// tied turns carried a reaction. With one reaction, dropping t.uuid still produces a workable
// order — action(t1), action(t2), reaction(t2) — and the assembly copes.
//
// Both turns carry one here, and that is the whole point. insertAction writes the SAME
// timestamp as created_at for a turn's action AND its reactions, and both turns close on the
// same finished_at, so once t.uuid is gone the only live sort key left is the
// `(a.react_to_uuid IS NOT NULL)` boolean — which groups every ACTION before every REACTION,
// across both turns:
//
//	action(t1), action(t2), reaction(t1), reaction(t2)
//
// reaction(t1) then arrives while curTurn is t2. The UUID does not match, so the assembly
// falls into the branch that opens a NEW turn and reads that row as the turn's own action —
// producing three turns, one of them a reaction wearing an action's clothes. That is exactly
// the discriminator bug the tiebreaker exists to prevent, and this test fails without it.
func TestFindMatchHistoryKeepsEachTiedTurnsReactionWithItsOwnTurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	sharedFinish := time.Now() // deliberately identical for both turns — forces the tie

	closeWithReaction := func(label string) (turnID, reactionID uuid.UUID) {
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		reaction := action.NewAction(
			fx.victimSheet, nil, act.GetID(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil,
		)
		reaction.ReactionKind = action.ReactRepel
		tn := turnentity.NewTurn(*act)
		tn.AddReaction(reaction)
		tn.Close(sharedFinish)
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		}); err != nil {
			t.Fatalf("PersistTurnClose %s: %v", label, err)
		}
		return tn.GetID(), reaction.GetID()
	}

	turn1, reaction1 := closeWithReaction("turn 1")
	turn2, reaction2 := closeWithReaction("turn 2")

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 {
		t.Fatalf("the tree is wrong under a finished_at tie: %+v", scenes)
	}
	turns := scenes[0].Rounds[0].Turns
	if len(turns) != 2 {
		t.Fatalf("turns = %d, want 2 — a reaction row was read as a turn's own action, "+
			"which is what happens when two turns tied on finished_at interleave", len(turns))
	}

	// Each turn must carry ITS OWN reaction, not the other's and not none. Counting across
	// both would pass even on a swap; naming them is what makes the claim two-sided.
	wantReactionOf := map[uuid.UUID]uuid.UUID{turn1: reaction1, turn2: reaction2}
	for _, tn := range turns {
		if tn.Action.ReactToID != uuid.Nil {
			t.Fatalf("turn %s's Action is actually a reaction (ReactToID=%s) — "+
				"turns interleaved on the finished_at tie", tn.UUID, tn.Action.ReactToID)
		}
		want, known := wantReactionOf[tn.UUID]
		if !known {
			t.Fatalf("unexpected turn %s in the result", tn.UUID)
		}
		if len(tn.Reactions) != 1 {
			t.Fatalf("turn %s has %d reactions, want exactly its own", tn.UUID, len(tn.Reactions))
		}
		if got := tn.Reactions[0].GetID(); got != want {
			t.Fatalf("turn %s came back carrying reaction %s, want its own %s — the "+
				"reactions swapped turns across the tie", tn.UUID, got, want)
		}
		delete(wantReactionOf, tn.UUID)
	}
	if len(wantReactionOf) != 0 {
		t.Fatalf("turns missing from the result: %+v", wantReactionOf)
	}
}

// EnsureSceneAndRound is the idempotent half PersistTurnClose used to keep to itself (spec
// §4.5, §4.8): a master action is recorded the instant it happens, and master_actions has an
// FK to the ACTIVE scene and round — which, before the first closed turn, exist only in memory.
// Calling it any number of times writes each row once, and the turn that closes later in that
// same round still goes through.
func TestEnsureSceneAndRound(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("twice is once, and the round's turn still closes afterwards", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Camp1")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match1")
		matchUUIDParsed, _ := uuid.Parse(matchUUID)
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &masterUUID, nil, &campaignUUID, "hero1")
		actorUUIDParsed, _ := uuid.Parse(sheetUUID)

		sc := sceneentity.NewScene(enum.Battle, "Arena")
		r := roundentity.NewRound(enum.Free)

		for i := 0; i < 2; i++ {
			if err := repo.EnsureSceneAndRound(ctx, matchUUIDParsed, sc, r); err != nil {
				t.Fatalf("EnsureSceneAndRound call %d: %v", i+1, err)
			}
		}

		var sceneCount, roundCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM scenes WHERE uuid = $1 AND match_uuid = $2`, sc.GetID(), matchUUIDParsed).Scan(&sceneCount) //nolint:errcheck
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM rounds WHERE uuid = $1 AND scene_uuid = $2`, r.GetID(), sc.GetID()).Scan(&roundCount)       //nolint:errcheck
		if sceneCount != 1 || roundCount != 1 {
			t.Fatalf("scene rows = %d, round rows = %d, want exactly 1 of each", sceneCount, roundCount)
		}

		// The active scene/round are now open rows: a restart rehydrates onto them.
		data, err := repo.FindActiveSession(ctx, matchUUIDParsed)
		if err != nil {
			t.Fatalf("FindActiveSession: %v", err)
		}
		if data == nil || data.SceneID != sc.GetID() || data.RoundID != r.GetID() {
			t.Fatalf("FindActiveSession = %+v, want the ensured scene %s / round %s", data, sc.GetID(), r.GetID())
		}

		act := action.NewAction(actorUUIDParsed, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		tRn := turnentity.NewTurn(*act)
		tRn.Close(time.Now())
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: r, Turn: tRn, Action: act, MatchUUID: matchUUIDParsed,
		}); err != nil {
			t.Fatalf("PersistTurnClose after EnsureSceneAndRound: %v", err)
		}
		var turnCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM turns WHERE round_uuid = $1`, r.GetID()).Scan(&turnCount) //nolint:errcheck
		if turnCount != 1 {
			t.Fatalf("turn rows = %d, want 1", turnCount)
		}
	})
}

// TestFindMatchHistoryKeepsWhatIsNotATurn is B15 (spec §4.5): a scene and a round are rows the
// moment they are born, so the history has to show them even when no turn ever closed inside
// them — the inner JOIN it used to run dropped both. A turn row with no action row (which
// PersistTurnClose's own transaction never writes, so only drift produces one) must not break
// the assembly either: the turn cannot be shown, but its round still is.
func TestFindMatchHistoryKeepsWhatIsNotATurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	// Fixed, increasing timestamps: the tree's order is (created_at, uuid), and uuid is random.
	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	at := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }

	// Scene A — born, and ended, with no turn: one round, no turn.
	sceneA := sceneentity.ReconstructScene(uuid.New(), enum.Roleplay, "Taverna", at(0))
	roundA := roundentity.ReconstructRound(uuid.New(), enum.Free, at(0))
	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sceneA, roundA); err != nil {
		t.Fatalf("EnsureSceneAndRound A: %v", err)
	}
	if err := repo.CloseSceneAndRound(ctx, sceneA.GetID(), roundA.GetID(), at(5)); err != nil {
		t.Fatalf("CloseSceneAndRound A: %v", err)
	}

	// Scene B — a round closed with no turn, a round with a turn, and a round whose only turn
	// lost its action row.
	sceneB := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(10))
	roundB1 := roundentity.ReconstructRound(uuid.New(), enum.Race, at(10))
	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sceneB, roundB1); err != nil {
		t.Fatalf("EnsureSceneAndRound B1: %v", err)
	}
	// B1 ends with no turn: its end and B2's birth go together.
	roundB1.Close(at(11))
	roundB2 := roundentity.ReconstructRound(uuid.New(), enum.Race, at(12))
	if err := repo.PersistRoundClose(ctx, fx.matchUUID, sceneB, roundB1, roundB2); err != nil {
		t.Fatalf("PersistRoundClose B1 -> B2: %v", err)
	}
	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(at(13))
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: sceneB, Round: roundB2, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose B2: %v", err)
	}

	roundB3 := roundentity.ReconstructRound(uuid.New(), enum.Race, at(14))
	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sceneB, roundB3); err != nil {
		t.Fatalf("EnsureSceneAndRound B3: %v", err)
	}
	orphanTurn := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO turns (uuid, round_uuid, created_at, finished_at) VALUES ($1, $2, $3, $3)`,
		orphanTurn, roundB3.GetID(), at(15),
	); err != nil {
		t.Fatalf("insert turn with no action: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("scenes = %d, want 2 — a scene with no turn is still a scene", len(scenes))
	}

	a := scenes[0]
	if a.UUID != sceneA.GetID() || a.Category != string(enum.Roleplay) || a.FinishedAt == nil {
		t.Fatalf("scene A = %+v, want the turnless, finished roleplay scene", a)
	}
	if len(a.Rounds) != 1 || a.Rounds[0].UUID != roundA.GetID() {
		t.Fatalf("scene A rounds = %+v, want only round A", a.Rounds)
	}
	if a.Rounds[0].Turns == nil || len(a.Rounds[0].Turns) != 0 {
		t.Fatalf("round A turns = %#v, want a non-nil empty slice", a.Rounds[0].Turns)
	}

	b := scenes[1]
	if b.UUID != sceneB.GetID() || len(b.Rounds) != 3 {
		t.Fatalf("scene B = %s with %d rounds, want %s with 3", b.UUID, len(b.Rounds), sceneB.GetID())
	}
	wantRounds := []uuid.UUID{roundB1.GetID(), roundB2.GetID(), roundB3.GetID()}
	for i, want := range wantRounds {
		if b.Rounds[i].UUID != want {
			t.Fatalf("scene B round %d = %s, want %s", i, b.Rounds[i].UUID, want)
		}
	}
	if b.Rounds[0].FinishedAt == nil || len(b.Rounds[0].Turns) != 0 || b.Rounds[0].Mode != string(enum.Race) {
		t.Fatalf("round B1 = %+v, want the closed Race round with no turn", b.Rounds[0])
	}
	if len(b.Rounds[1].Turns) != 1 || b.Rounds[1].Turns[0].UUID != tn.GetID() ||
		b.Rounds[1].Turns[0].Action.GetID() != act.GetID() {
		t.Fatalf("round B2 turns = %+v, want exactly the persisted turn", b.Rounds[1].Turns)
	}
	if b.Rounds[2].Turns == nil || len(b.Rounds[2].Turns) != 0 {
		t.Fatalf("round B3 turns = %#v, want a non-nil empty slice — a turn with no action cannot be shown", b.Rounds[2].Turns)
	}
}

// TestEnsureSceneAndRoundRefreshesTheRoundsMode: a round is a row from birth now, so a regime
// change after that has to reach the row, or the history would read every round as the regime
// it was born in. EnsureSceneAndRound is what the change_round_mode arm calls.
func TestEnsureSceneAndRoundRefreshesTheRoundsMode(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, fx.scene, fx.round); err != nil {
		t.Fatalf("EnsureSceneAndRound (born Free): %v", err)
	}
	fx.round.SetMode(enum.Race)
	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, fx.scene, fx.round); err != nil {
		t.Fatalf("EnsureSceneAndRound (now Race): %v", err)
	}

	var mode string
	var n int
	if err := pool.QueryRow(ctx, `SELECT mode, (SELECT COUNT(*) FROM rounds WHERE uuid = $1) FROM rounds WHERE uuid = $1`,
		fx.round.GetID()).Scan(&mode, &n); err != nil {
		t.Fatalf("read round: %v", err)
	}
	if n != 1 || mode != string(enum.Race) {
		t.Fatalf("rounds row = %d with mode %q, want exactly 1 with %q", n, mode, enum.Race)
	}
}

// TestEnsureSceneAndRoundCarriesFinishedAt (F3): a round whose birth write failed, and that
// nothing else marked as a row, is closed in memory by exhaustion without CloseRound ever
// running (it was not a row to close). Its last turn's PersistTurnClose then writes it for
// the first time — and must write it CLOSED, or the scene ends up with two open rounds (the
// stray one and the next one, born right after). A finish already on the row is never moved.
func TestEnsureSceneAndRoundCarriesFinishedAt(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	at := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }
	finishedAt := func(t *testing.T, table string, id uuid.UUID) *time.Time {
		t.Helper()
		var f *time.Time
		if err := pool.QueryRow(ctx, `SELECT finished_at FROM `+table+` WHERE uuid = $1`, id).Scan(&f); err != nil {
			t.Fatalf("read %s.finished_at: %v", table, err)
		}
		return f
	}

	t.Run("a turn closing on a round closed in memory writes it closed, and only the next round is open", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		stray := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		// The birth write of `stray` failed: no row. Exhaustion closes it in memory.
		stray.Close(at(3))

		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(at(2))
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: stray, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		}); err != nil {
			t.Fatalf("PersistTurnClose: %v", err)
		}
		if f := finishedAt(t, "rounds", stray.GetID()); f == nil || !f.Equal(at(3)) {
			t.Fatalf("stray round finished_at = %v, want %v — it was written open", f, at(3))
		}

		next := roundentity.ReconstructRound(uuid.New(), enum.Race, at(3))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, next); err != nil {
			t.Fatalf("EnsureSceneAndRound next: %v", err)
		}
		var open int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM rounds WHERE scene_uuid = $1 AND finished_at IS NULL`, sc.GetID(),
		).Scan(&open); err != nil {
			t.Fatalf("count open rounds: %v", err)
		}
		if open != 1 {
			t.Fatalf("open rounds in the scene = %d, want 1 (only the next one)", open)
		}
	})

	t.Run("an open row is closed by a later ensure of the closed round; a finish is never moved", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Free, at(0))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound (open): %v", err)
		}
		if f := finishedAt(t, "rounds", rd.GetID()); f != nil {
			t.Fatalf("round finished_at = %v right after its birth, want NULL", f)
		}

		rd.Close(at(5))
		sc.Close(at(5))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound (closed): %v", err)
		}
		if f := finishedAt(t, "rounds", rd.GetID()); f == nil || !f.Equal(at(5)) {
			t.Fatalf("round finished_at = %v, want %v", f, at(5))
		}
		if f := finishedAt(t, "scenes", sc.GetID()); f == nil || !f.Equal(at(5)) {
			t.Fatalf("scene finished_at = %v, want %v", f, at(5))
		}

		// A later ensure carrying a DIFFERENT finish (or none) never moves the recorded one.
		later := roundentity.ReconstructRound(rd.GetID(), enum.Free, at(0))
		later.Close(at(9))
		laterScene := sceneentity.ReconstructScene(sc.GetID(), enum.Battle, "Arena", at(0))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, laterScene, later); err != nil {
			t.Fatalf("EnsureSceneAndRound (later finish): %v", err)
		}
		if f := finishedAt(t, "rounds", rd.GetID()); f == nil || !f.Equal(at(5)) {
			t.Fatalf("round finished_at = %v after a later ensure, want it kept at %v", f, at(5))
		}
		if f := finishedAt(t, "scenes", sc.GetID()); f == nil || !f.Equal(at(5)) {
			t.Fatalf("scene finished_at = %v after a later ensure with no finish, want it kept at %v", f, at(5))
		}
	})
}

// roundRowsOf reads what a round-end test asserts: a round's finished_at (exists=false when the
// round is not a row) and how many rounds of the scene are open.
func roundRowsOf(t *testing.T, pool *pgxpool.Pool, rd uuid.UUID, sc uuid.UUID) (finished *time.Time, exists bool, open int) {
	t.Helper()
	ctx := context.Background()
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rounds WHERE uuid = $1`, rd).Scan(&n); err != nil {
		t.Fatalf("count round: %v", err)
	}
	if n == 1 {
		if err := pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, rd).Scan(&finished); err != nil {
			t.Fatalf("read round finished_at: %v", err)
		}
	}
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rounds WHERE scene_uuid = $1 AND finished_at IS NULL`, sc,
	).Scan(&open); err != nil {
		t.Fatalf("count open rounds: %v", err)
	}
	return finished, n == 1, open
}

// TestPersistTurnCloseWritesTheRoundEndWithTheTurn: an open_next_action that closes the last
// turn of a round and finds nothing that can still pay its price ends the round too — one master
// command, one transaction (owner decision, 2026-10-02). The round's finish and its successor's
// birth go in the turn's transaction (TurnCloseData.NextRound), and roll back with it.
func TestPersistTurnCloseWritesTheRoundEndWithTheTurn(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	base := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	at := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }
	setup := func(t *testing.T) (resolutionFixture, *sceneentity.Scene, *roundentity.Round, *roundentity.Round, *turnentity.Turn, *action.Action) {
		t.Helper()
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		// Born a row (B15), open.
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound: %v", err)
		}
		act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
		tn := turnentity.NewTurn(*act)
		tn.Close(at(2))
		rd.Close(at(2))
		next := roundentity.ReconstructRound(uuid.New(), enum.Race, at(2))
		return fx, sc, rd, next, tn, act
	}

	t.Run("the round's finish and the successor's birth go with the turn", func(t *testing.T) {
		fx, sc, rd, next, tn, act := setup(t)
		if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: rd, NextRound: next, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		}); err != nil {
			t.Fatalf("PersistTurnClose: %v", err)
		}
		if f, ok, _ := roundRowsOf(t, pool, rd.GetID(), sc.GetID()); !ok || f == nil || !f.Equal(at(2)) {
			t.Fatalf("closed round finished_at = %v (row %v), want %v", f, ok, at(2))
		}
		f, ok, open := roundRowsOf(t, pool, next.GetID(), sc.GetID())
		if !ok || f != nil {
			t.Fatalf("successor round row %v with finished_at %v, want an open row", ok, f)
		}
		if open != 1 {
			t.Fatalf("open rounds in the scene = %d, want 1 — the successor alone", open)
		}
	})

	t.Run("a failing master action rolls the round's end back with the turn", func(t *testing.T) {
		fx, sc, rd, next, tn, act := setup(t)
		turnID := tn.GetID()
		bad := masteraction.Record{
			UUID: uuid.New(), MatchUUID: fx.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
			TurnUUID: &turnID, MasterUUID: uuid.New(), // not a user: the FK refuses it
			Kind: masteraction.KindMovePiece, Content: []byte(`{}`), HappenedAt: at(1),
		}
		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: rd, NextRound: next, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
			MasterActions: []masteraction.Record{bad},
		})
		if err == nil {
			t.Fatal("PersistTurnClose succeeded with a master action the database refuses")
		}
		if f, ok, open := roundRowsOf(t, pool, rd.GetID(), sc.GetID()); !ok || f != nil || open != 1 {
			t.Fatalf("closed round row %v finished_at %v, %d open — want it still the one open round", ok, f, open)
		}
		if _, ok, _ := roundRowsOf(t, pool, next.GetID(), sc.GetID()); ok {
			t.Fatal("the successor became a row although the turn's transaction failed")
		}
	})

	t.Run("a successor with an open round refuses to write two open rounds", func(t *testing.T) {
		fx, sc, rd, next, tn, act := setup(t)
		stillOpen := roundentity.ReconstructRound(rd.GetID(), enum.Race, at(0))
		err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
			Scene: sc, Round: stillOpen, NextRound: next, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		})
		if err == nil {
			t.Fatal("PersistTurnClose wrote a successor next to a round that is not finished")
		}
		if _, ok, open := roundRowsOf(t, pool, next.GetID(), sc.GetID()); ok || open != 1 {
			t.Fatalf("successor row %v, %d open rounds — want no successor and the one open round", ok, open)
		}
	})
}

// TestPersistRoundClose: a round that ends with no turn closing in the same command writes its
// end and its successor's birth in one transaction of their own — never one without the other,
// so the scene never holds two open rounds, nor none.
func TestPersistRoundClose(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)

	base := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	at := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }

	t.Run("writes the round's finish and the successor's birth", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound: %v", err)
		}
		rd.Close(at(3))
		next := roundentity.ReconstructRound(uuid.New(), enum.Race, at(3))

		if err := repo.PersistRoundClose(ctx, fx.matchUUID, sc, rd, next); err != nil {
			t.Fatalf("PersistRoundClose: %v", err)
		}
		if f, ok, _ := roundRowsOf(t, pool, rd.GetID(), sc.GetID()); !ok || f == nil || !f.Equal(at(3)) {
			t.Fatalf("closed round finished_at = %v (row %v), want %v", f, ok, at(3))
		}
		if f, ok, open := roundRowsOf(t, pool, next.GetID(), sc.GetID()); !ok || f != nil || open != 1 {
			t.Fatalf("successor row %v finished_at %v, %d open — want the successor as the one open round", ok, f, open)
		}
		data, err := repo.FindActiveSession(ctx, fx.matchUUID)
		if err != nil || data == nil || data.RoundID != next.GetID() {
			t.Fatalf("FindActiveSession = %+v (%v), want the successor %s", data, err, next.GetID())
		}
	})

	t.Run("a round that was never a row lands closed", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		rd.Close(at(3))
		next := roundentity.ReconstructRound(uuid.New(), enum.Race, at(3))

		if err := repo.PersistRoundClose(ctx, fx.matchUUID, sc, rd, next); err != nil {
			t.Fatalf("PersistRoundClose: %v", err)
		}
		if f, ok, open := roundRowsOf(t, pool, rd.GetID(), sc.GetID()); !ok || f == nil || open != 1 {
			t.Fatalf("closed round row %v finished_at %v, %d open — want it written closed", ok, f, open)
		}
	})

	t.Run("a successor the database refuses rolls the round's finish back", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound: %v", err)
		}
		rd.Close(at(3))
		// rounds.mode is VARCHAR(16): a longer regime is refused by the database.
		next := roundentity.ReconstructRound(uuid.New(), enum.RoundMode("a-regime-too-long-for-the-column"), at(3))

		if err := repo.PersistRoundClose(ctx, fx.matchUUID, sc, rd, next); err == nil {
			t.Fatal("PersistRoundClose succeeded with a successor the database refuses")
		}
		if f, ok, open := roundRowsOf(t, pool, rd.GetID(), sc.GetID()); !ok || f != nil || open != 1 {
			t.Fatalf("closed round row %v finished_at %v, %d open — want it still the one open round", ok, f, open)
		}
	})

	t.Run("a round that is not finished is refused", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := seedMatchAndSheets(t, pool)
		sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", at(0))
		rd := roundentity.ReconstructRound(uuid.New(), enum.Race, at(0))
		next := roundentity.ReconstructRound(uuid.New(), enum.Race, at(3))
		if err := repo.PersistRoundClose(ctx, fx.matchUUID, sc, rd, next); err == nil {
			t.Fatal("PersistRoundClose accepted a round with no finished_at")
		}
	})
}

// TestFindActiveSessionIsDeterministic (F3): LIMIT 1 with no ORDER BY picked whichever open
// round the plan happened to hand back first. If drift ever leaves two open rounds in an open
// scene, the active one is the NEWEST — the round born last is the one the table is in.
func TestFindActiveSessionIsDeterministic(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	sc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "Arena", base)
	older := roundentity.ReconstructRound(uuid.New(), enum.Race, base)
	newer := roundentity.ReconstructRound(uuid.New(), enum.Race, base.Add(time.Minute))
	// The older one is written first, so it is also first on disk.
	for _, rd := range []*roundentity.Round{older, newer} {
		if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
			t.Fatalf("EnsureSceneAndRound: %v", err)
		}
	}

	data, err := repo.FindActiveSession(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindActiveSession: %v", err)
	}
	if data == nil || data.RoundID != newer.GetID() {
		t.Fatalf("FindActiveSession round = %+v, want the newest open round %s", data, newer.GetID())
	}
}

// TestPersistTurnCloseRoundTripsTheViewsRecordedLive is the persistence half of "the history
// shows each reader a move as they saw it live" (owner decision, 2026-10-01): the opened move's
// per-player verdicts go to actions.move_views, each landed escape's to its escape entry
// (landingViews), and FindMatchHistory reads both back. A turn closed without them — what every
// row written before this column existed looks like — reads back with neither, and decodes.
func TestPersistTurnCloseRoundTripsTheViewsRecordedLive(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	sawAll, sawLeave, sawLanding := uuid.New(), uuid.New(), uuid.New()
	landing := [3]int{7, 6, 0}
	escapeRes := func() *service.TurnResolution {
		return &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{{
			TargetID: fx.victimSheet, ReactionKind: string(action.ReactEscapeGuard),
			Escape: &service.EscapeResult{DodgePassed: true, Landing: &landing},
		}}}
	}
	dash := func() *action.Action {
		return action.NewAction(
			fx.attackerSheet, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Dash, From: &[3]int{1, 1, 0}, Position: [3]int{4, 4, 0}},
			nil, nil, nil, nil, nil,
		)
	}

	// Turn 1 carries the views; turn 2 is shaped like a row from before them.
	act1 := dash()
	tn1 := turnentity.NewTurn(*act1)
	tn1.Close(time.Now())
	moveViews := map[uuid.UUID]masteraction.View{sawAll: masteraction.ViewFull, sawLeave: masteraction.ViewLeft}
	landingViews := map[uuid.UUID]map[uuid.UUID]masteraction.View{fx.victimSheet: {sawLanding: masteraction.ViewFull}}
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn1, Action: act1, MatchUUID: fx.matchUUID,
		Resolution: escapeRes(), MoveViews: moveViews, LandingViews: landingViews,
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 1: %v", err)
	}
	act2 := dash()
	tn2 := turnentity.NewTurn(*act2)
	tn2.Close(time.Now().Add(time.Second))
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn2, Action: act2, MatchUUID: fx.matchUUID,
		Resolution: escapeRes(),
	}); err != nil {
		t.Fatalf("PersistTurnClose turn 2: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT move_views FROM actions WHERE uuid = $1`, act2.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read move_views of turn 2: %v", err)
	}
	if raw != nil {
		t.Fatalf("a turn closed with no views wrote move_views = %s, want NULL", raw)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	turns := scenes[0].Rounds[0].Turns
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2", len(turns))
	}
	if !reflect.DeepEqual(turns[0].MoveViews, moveViews) {
		t.Errorf("move views read back = %v, want %v", turns[0].MoveViews, moveViews)
	}
	if !reflect.DeepEqual(turns[0].LandingViews, landingViews) {
		t.Errorf("landing views read back = %v, want %v", turns[0].LandingViews, landingViews)
	}
	if turns[1].MoveViews != nil || turns[1].LandingViews != nil {
		t.Errorf("a row with no views read back with some: move %v, landing %v", turns[1].MoveViews, turns[1].LandingViews)
	}
	// The resolution itself is untouched by the views, old row or new.
	for i, tu := range turns {
		if tu.Resolution == nil || len(tu.Resolution.CharacterResults) != 1 {
			t.Fatalf("turn %d: the resolution did not decode: %+v", i+1, tu.Resolution)
		}
		if e := tu.Resolution.CharacterResults[0].Escape; e == nil || e.Landing == nil || *e.Landing != landing {
			t.Errorf("turn %d: the escape's landing did not survive: %+v", i+1, e)
		}
	}
}

// An escape that ESCAPED has no landing — its piece went to its own destination — and who saw
// it arrive is recorded all the same: it is what the history shows the reaction's move.position
// by. An escape whose piece did not move records nothing.
func TestPersistTurnCloseKeepsWhoSawAnEscapedEscapeArrive(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())
	stayed := uuid.New()
	saw := uuid.New()
	landingViews := map[uuid.UUID]map[uuid.UUID]masteraction.View{
		fx.victimSheet: {saw: masteraction.ViewFull},
		stayed:         {saw: masteraction.ViewFull}, // not a move: must not be written
	}
	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		Resolution: &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{
			{TargetID: fx.victimSheet, ReactionKind: string(action.ReactEscape),
				Escape: &service.EscapeResult{MovePassed: true, DodgePassed: true, Escaped: true}},
			{TargetID: stayed, ReactionKind: string(action.ReactEscape),
				Escape: &service.EscapeResult{MovePassed: true}},
		}},
		LandingViews: landingViews,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	got := scenes[0].Rounds[0].Turns[0].LandingViews
	want := map[uuid.UUID]map[uuid.UUID]masteraction.View{fx.victimSheet: {saw: masteraction.ViewFull}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("landing views read back = %v, want %v", got, want)
	}
}
