//go:build integration

package pgcancelledaction_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/actioncodec"
	pgcancelledaction "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestCancelledAction(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgcancelledaction.NewRepository(pool)
	ctx := context.Background()

	t.Run("insert with and without a turn — the row reads back by match, action whole", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "gm-ca", "gm-ca@test.com", "pass")
		playerStr := pgtest.InsertTestUser(t, pool, "pl-ca", "pl-ca@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "CA Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "CA Match")
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Race")
		sheetStr := pgtest.InsertTestCharacterSheet(t, pool, &playerStr, nil, nil, "Gon")
		matchUUID, sheetUUID := uuid.MustParse(matchStr), uuid.MustParse(sheetStr)

		build := func(turn *uuid.UUID, at time.Time) cancelledaction.Record {
			act := action.NewAction(sheetUUID, []uuid.UUID{uuid.New()}, uuid.Nil, nil,
				action.ActionSpeed{Bar: 1, RollCheck: action.RollCheck{SkillName: "Legerity", Result: 8}},
				nil, nil, &action.Attack{Hit: action.RollCheck{SkillName: "Accuracy", Result: 12}},
				nil, nil, nil, nil)
			act.SystemBias = -1
			return cancelledaction.Record{
				UUID: act.GetID(), MatchUUID: matchUUID,
				SceneUUID: uuid.MustParse(sceneStr), RoundUUID: uuid.MustParse(roundStr),
				TurnUUID: turn, ActorUUID: sheetUUID, CancelledBy: uuid.MustParse(playerStr),
				Action: *act, CancelledAt: at,
			}
		}
		turnID := uuid.New()
		at := time.Now().UTC().Truncate(time.Microsecond)
		withTurn := build(&turnID, at)
		withoutTurn := build(nil, at.Add(time.Second))

		for _, rec := range []cancelledaction.Record{withTurn, withoutTurn} {
			if err := repo.Insert(ctx, rec); err != nil {
				t.Fatalf("Insert: %v", err)
			}
		}

		rows, err := pool.Query(ctx,
			`SELECT uuid, scene_uuid, round_uuid, turn_uuid, actor_uuid, cancelled_by, action, cancelled_at
			 FROM cancelled_actions WHERE match_uuid = $1 ORDER BY cancelled_at`, matchUUID)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()
		var got []cancelledaction.Record
		for rows.Next() {
			rec := cancelledaction.Record{MatchUUID: matchUUID}
			var raw []byte
			if err := rows.Scan(&rec.UUID, &rec.SceneUUID, &rec.RoundUUID, &rec.TurnUUID, &rec.ActorUUID,
				&rec.CancelledBy, &raw, &rec.CancelledAt); err != nil {
				t.Fatalf("scan: %v", err)
			}
			act, err := actioncodec.Unmarshal(rec.UUID, rec.ActorUUID, raw)
			if err != nil {
				t.Fatalf("decode action: %v", err)
			}
			rec.Action = *act
			rec.CancelledAt = rec.CancelledAt.UTC()
			got = append(got, rec)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("rows = %d, want 2", len(got))
		}
		for i, want := range []cancelledaction.Record{withTurn, withoutTurn} {
			if !reflect.DeepEqual(got[i], want) {
				t.Errorf("row %d:\n got  %+v\n want %+v", i, got[i], want)
			}
		}
	})

	t.Run("ListByMatch reads this match's records in time order, action decoded", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "gm-cl", "gm-cl@test.com", "pass")
		playerStr := pgtest.InsertTestUser(t, pool, "pl-cl", "pl-cl@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "CL Campaign")
		sheetStr := pgtest.InsertTestCharacterSheet(t, pool, &playerStr, nil, nil, "Gon")
		sheetUUID := uuid.MustParse(sheetStr)

		build := func(matchStr string, turn *uuid.UUID, at time.Time) cancelledaction.Record {
			sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
			roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Race")
			act := action.NewAction(sheetUUID, []uuid.UUID{uuid.New()}, uuid.Nil, nil,
				action.ActionSpeed{Bar: 1, RollCheck: action.RollCheck{SkillName: "Legerity", Result: 8}},
				nil, nil, &action.Attack{Hit: action.RollCheck{SkillName: "Accuracy", Result: 12}},
				nil, nil, nil, nil)
			return cancelledaction.Record{
				UUID: act.GetID(), MatchUUID: uuid.MustParse(matchStr),
				SceneUUID: uuid.MustParse(sceneStr), RoundUUID: uuid.MustParse(roundStr),
				TurnUUID: turn, ActorUUID: sheetUUID, CancelledBy: uuid.MustParse(playerStr),
				Action: *act, CancelledAt: at,
			}
		}
		matchStr := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "CL Match")
		otherMatch := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "CL Other")
		turnID := uuid.New()
		at := time.Now().UTC().Truncate(time.Microsecond)
		first := build(matchStr, &turnID, at)
		second := build(matchStr, nil, at.Add(time.Second))
		elsewhere := build(otherMatch, nil, at.Add(500*time.Millisecond))

		// Inserted out of time order: the read orders.
		for _, rec := range []cancelledaction.Record{second, elsewhere, first} {
			if err := repo.Insert(ctx, rec); err != nil {
				t.Fatalf("Insert: %v", err)
			}
		}

		got, err := repo.ListByMatch(ctx, uuid.MustParse(matchStr))
		if err != nil {
			t.Fatalf("ListByMatch: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("records = %d, want 2 (only this match's)", len(got))
		}
		for i, want := range []cancelledaction.Record{first, second} {
			got[i].CancelledAt = got[i].CancelledAt.UTC()
			if !reflect.DeepEqual(got[i], want) {
				t.Errorf("record %d:\n got  %+v\n want %+v", i, got[i], want)
			}
		}

		empty, err := repo.ListByMatch(ctx, uuid.New())
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("ListByMatch of a match with none = %#v, %v; want a non-nil empty slice", empty, err)
		}
	})

	// cancelled_actions.hidden_from (revisão final do PR 2, achado 2): o registro do ator
	// Escondido volta como foi gravado; nil = SQL NULL; ilegível falha fechado (a sentinela).
	t.Run("hidden_from round-trips, NULL when none, fails closed when unreadable", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "gm-ch", "gm-ch@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "CH Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "CH Match")
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Race")
		sheetUUID := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, nil, &masterStr, nil, "NPC"))
		at := time.Now().UTC().Truncate(time.Microsecond)
		build := func(sec int, hf statuseffect.HiddenFrom) cancelledaction.Record {
			act := action.NewAction(sheetUUID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
			return cancelledaction.Record{
				UUID: act.GetID(), MatchUUID: uuid.MustParse(matchStr),
				SceneUUID: uuid.MustParse(sceneStr), RoundUUID: uuid.MustParse(roundStr),
				ActorUUID: sheetUUID, CancelledBy: uuid.MustParse(masterStr),
				Action: *act, CancelledAt: at.Add(time.Duration(sec) * time.Second), HiddenFrom: hf,
			}
		}
		hidden := statuseffect.HiddenFrom{sheetUUID: {Effect: uuid.New(), PerceivedBy: []uuid.UUID{uuid.New()}}}
		recs := []cancelledaction.Record{build(0, hidden), build(1, nil), build(2, hidden)}
		for _, rec := range recs {
			if err := repo.Insert(ctx, rec); err != nil {
				t.Fatalf("Insert: %v", err)
			}
		}
		var raw []byte
		if err := pool.QueryRow(ctx, `SELECT hidden_from FROM cancelled_actions WHERE uuid = $1`, recs[1].UUID).Scan(&raw); err != nil {
			t.Fatalf("read hidden_from: %v", err)
		}
		if raw != nil {
			t.Fatalf("hidden_from with no record = %s, want SQL NULL", raw)
		}
		if _, err := pool.Exec(ctx, `UPDATE cancelled_actions SET hidden_from = '"not-a-map"'::jsonb WHERE uuid = $1`, recs[2].UUID); err != nil {
			t.Fatalf("corrupt hidden_from: %v", err)
		}
		got, err := repo.ListByMatch(ctx, uuid.MustParse(matchStr))
		if err != nil || len(got) != 3 {
			t.Fatalf("ListByMatch = %d, %v", len(got), err)
		}
		if !reflect.DeepEqual(got[0].HiddenFrom, hidden) {
			t.Fatalf("hidden_from = %v, want %v", got[0].HiddenFrom, hidden)
		}
		if got[1].HiddenFrom != nil {
			t.Fatalf("hidden_from = %v, want nil", got[1].HiddenFrom)
		}
		if want := (statuseffect.HiddenFrom{sheetUUID: {}}); !reflect.DeepEqual(got[2].HiddenFrom, want) {
			t.Fatalf("unreadable hidden_from = %v, want the sentinel %v", got[2].HiddenFrom, want)
		}
	})

	t.Run("an actor that is not a sheet is refused", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterStr := pgtest.InsertTestUser(t, pool, "gm-ca2", "gm-ca2@test.com", "pass")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterStr, "CA Campaign")
		matchStr := pgtest.InsertTestMatch(t, pool, masterStr, campaignUUID, "CA Match")
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Race")
		act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		err := repo.Insert(ctx, cancelledaction.Record{
			UUID: act.GetID(), MatchUUID: uuid.MustParse(matchStr),
			SceneUUID: uuid.MustParse(sceneStr), RoundUUID: uuid.MustParse(roundStr),
			ActorUUID: act.GetActorID(), CancelledBy: uuid.MustParse(masterStr),
			Action: *act, CancelledAt: time.Now().UTC(),
		})
		if err == nil {
			t.Fatal("Insert accepted an actor_uuid with no character sheet")
		}
	})
}
