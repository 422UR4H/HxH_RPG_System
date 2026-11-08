//go:build integration

package pgclock_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	pgclock "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/clock"
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
