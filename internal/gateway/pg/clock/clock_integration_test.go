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
	if err := pgclock.UpsertTx(ctx, tx, matchUUID, snap, time.Now()); err != nil {
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

	// updated_at vem do Go (o instante do Save), não do now() do banco
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT updated_at FROM match_clocks WHERE match_uuid = $1`, matchUUID).Scan(&updatedAt); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(updatedAt); d < -time.Minute || d > time.Minute {
		t.Errorf("updated_at = %v, want about now", updatedAt)
	}
}

// Uma partida de verdade, com relógio e evento próprios, tenta fundir no evento da primeira:
// erro, o payload do evento da primeira não muda e o relógio da segunda não avança (mesma
// transação).
func TestClockMergeIsScopedToTheMatch(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	master := pgtest.InsertTestUser(t, pool, "gm3", "gm3@test.com", "pass")
	camp := pgtest.InsertTestCampaign(t, pool, master, "Camp")
	repo := pgclock.NewRepository(pool)
	events := pgmatchevent.NewRepository(pool)
	at := time.Date(2000, 1, 1, 8, 0, 0, 0, time.UTC)

	seed := func(name, payload string) (uuid.UUID, matchevent.Event) {
		t.Helper()
		matchStr := pgtest.InsertTestMatch(t, pool, master, camp, name)
		matchUUID := uuid.MustParse(matchStr)
		sceneStr := pgtest.InsertTestScene(t, pool, matchStr, "Battle")
		roundStr := pgtest.InsertTestRound(t, pool, sceneStr, "Free")
		ev := matchevent.Event{
			UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: uuid.MustParse(sceneStr),
			RoundUUID: uuid.MustParse(roundStr), Kind: matchevent.KindClockAdjusted,
			Payload: json.RawMessage(payload), CreatedAt: time.Now().UTC(),
		}
		snap := clock.Snapshot{StoryAt: &at, LimitAt: &at, LimitReason: clock.LimitStart}
		if err := repo.Save(ctx, matchUUID, snap, &appmatch.ClockEvent{Event: ev}); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
		return matchUUID, ev
	}
	first, firstEv := seed("First", `{"deltaSeconds":60}`)
	second, secondEv := seed("Second", `{"deltaSeconds":5}`)

	later := at.Add(time.Hour)
	hijack := secondEv
	hijack.Payload = json.RawMessage(`{"deltaSeconds":999}`)
	err := repo.Save(ctx, second, clock.Snapshot{StoryAt: &later, LimitAt: &at, LimitReason: clock.LimitStart},
		&appmatch.ClockEvent{Event: hijack, MergeInto: firstEv.UUID})
	if err == nil {
		t.Fatal("merging into another match's event must fail")
	}

	list, err := events.ListByMatch(ctx, first)
	if err != nil || len(list) != 1 {
		t.Fatalf("first match events = %d, %v; want 1", len(list), err)
	}
	var p map[string]int
	if err := json.Unmarshal(list[0].Payload, &p); err != nil || len(p) != 1 || p["deltaSeconds"] != 60 {
		t.Errorf("first match's event payload = %s, want unchanged", list[0].Payload)
	}
	got, err := repo.Load(ctx, second)
	if err != nil || got == nil || !got.StoryAt.Equal(at) {
		t.Errorf("second match's clock = %+v, %v; want still at %v (rolled back)", got, err, at)
	}
}
