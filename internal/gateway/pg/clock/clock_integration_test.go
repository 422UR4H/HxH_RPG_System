//go:build integration

package pgclock_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	pgclock "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/clock"
	pgmatchevent "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchevent"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	"github.com/google/uuid"
)

// 23:59:59 tem que sobreviver à ida e volta mesmo com a sessão do banco num fuso diferente:
// a coluna é TIMESTAMP, e o relógio da história é relógio de parede, não instante.
func TestClockSnapshotNaiveRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	master := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")
	camp := pgtest.InsertTestCampaign(t, pool, master, "Camp")
	matchUUID, _ := uuid.Parse(pgtest.InsertTestMatch(t, pool, master, camp, "M"))

	storyAt := time.Date(2000, 1, 1, 23, 59, 59, 0, time.UTC)
	limit := time.Date(2000, 1, 1, 23, 59, 58, 0, time.UTC)
	snap := clock.Snapshot{StoryAt: &storyAt, LimitAt: &limit, LimitReason: clock.LimitTurn}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SET LOCAL TIME ZONE 'America/Belem'`); err != nil {
		t.Fatal(err)
	}
	if err := pgclock.UpsertTx(ctx, tx, matchUUID, snap); err != nil {
		t.Fatalf("UpsertTx: %v", err)
	}
	got, err := pgclock.LoadFrom(ctx, tx, matchUUID)
	if err != nil || got == nil {
		t.Fatalf("LoadFrom = %v, %v", got, err)
	}
	if !got.StoryAt.Equal(storyAt) || !got.LimitAt.Equal(limit) || got.LimitReason != clock.LimitTurn || got.Running {
		t.Errorf("round trip in Belem = %+v story %v limit %v", got, got.StoryAt, got.LimitAt)
	}
	var wall string
	if err := tx.QueryRow(ctx, `SELECT story_at::text FROM match_clocks WHERE match_uuid=$1`, matchUUID).Scan(&wall); err != nil {
		t.Fatal(err)
	}
	if wall != "2000-01-01 23:59:59" {
		t.Errorf("stored wall clock = %q", wall)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// e, já commitado, o Load do repositório (outra conexão) lê o mesmo
	loaded, err := pgclock.NewRepository(pool).Load(ctx, matchUUID)
	if err != nil || loaded == nil || !loaded.StoryAt.Equal(storyAt) {
		t.Errorf("Load = %+v, %v", loaded, err)
	}
	if none, err := pgclock.NewRepository(pool).Load(ctx, uuid.New()); err != nil || none != nil {
		t.Errorf("Load of unknown match = %v, %v, want nil", none, err)
	}
}

func TestClockRepositorySaveLoadAndMerge(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	master := pgtest.InsertTestUser(t, pool, "gm2", "gm2@test.com", "pass")
	camp := pgtest.InsertTestCampaign(t, pool, master, "Camp")
	matchStr := pgtest.InsertTestMatch(t, pool, master, camp, "M")
	matchUUID := uuid.MustParse(matchStr)
	sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
	roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Free")
	repo := pgclock.NewRepository(pool)
	events := pgmatchevent.NewRepository(pool)

	at := time.Date(2000, 1, 1, 8, 0, 0, 0, time.UTC)
	ev := matchevent.Event{
		UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: uuid.MustParse(sceneStr),
		RoundUUID: uuid.MustParse(roundStr), Kind: matchevent.KindRoundModeChanged,
		Payload: json.RawMessage(`{"a":1}`), CreatedAt: time.Now().UTC(),
	}
	if err := repo.Save(ctx, matchUUID, clock.Snapshot{StoryAt: &at, LimitAt: &at, LimitReason: clock.LimitStart}, &appmatch.ClockEvent{Event: ev}); err != nil {
		t.Fatalf("Save with event: %v", err)
	}
	got, err := repo.Load(ctx, matchUUID)
	if err != nil || got == nil || !got.StoryAt.Equal(at) {
		t.Fatalf("Load = %+v, %v", got, err)
	}

	later := at.Add(time.Hour)
	merged := ev
	merged.Payload = json.RawMessage(`{"a":2}`)
	if err := repo.Save(ctx, matchUUID, clock.Snapshot{StoryAt: &later, LimitAt: &at, LimitReason: clock.LimitStart}, &appmatch.ClockEvent{Event: merged, MergeInto: ev.UUID}); err != nil {
		t.Fatalf("Save with merge: %v", err)
	}
	list, err := events.ListByMatch(ctx, matchUUID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListByMatch = %d events, %v; want 1", len(list), err)
	}
	var p map[string]int
	if err := json.Unmarshal(list[0].Payload, &p); err != nil || p["a"] != 2 {
		t.Errorf("payload after merge = %s", list[0].Payload)
	}

	// merge em evento de outra partida: erro, e o checkpoint não avança (mesma transação)
	other := uuid.New()
	evErr := repo.Save(ctx, other, clock.Snapshot{StoryAt: &at}, &appmatch.ClockEvent{Event: merged, MergeInto: ev.UUID})
	if evErr == nil {
		t.Error("merge into another match's event should fail")
	}
}
