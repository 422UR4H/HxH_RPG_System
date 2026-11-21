//go:build integration

package pgstatuseffect_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	pgstatuseffect "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/statuseffect"
)

func now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

func prone(sheet uuid.UUID, at time.Time) statuseffect.Effect {
	return statuseffect.Effect{
		ID: uuid.New(), CharacterID: sheet, Kind: statuseffect.KindProne,
		AppliedAt: at, UpdatedAt: at,
	}
}

func TestWrite(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	setup := func(t *testing.T) uuid.UUID {
		pgtest.TruncateAll(t, pool)
		player := pgtest.InsertTestUser(t, pool, "pl-se", "pl-se@test.com", "pass")
		return uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Gon"))
	}
	total := func(t *testing.T) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM character_status_effects`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}

	t.Run("upsert inserts then updates", func(t *testing.T) {
		sheet := setup(t)
		at := now()
		e := statuseffect.Effect{
			ID: uuid.New(), CharacterID: sheet, Kind: statuseffect.KindBleeding,
			Params:    statuseffect.Params{HPPerRound: 1},
			Alias:     "corte",
			Audience:  &statuseffect.Audience{Everyone: true},
			AppliedAt: at, UpdatedAt: at,
		}
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e}}); err != nil {
			t.Fatalf("insert: %v", err)
		}
		edited := e
		edited.Params.HPPerRound = 3
		edited.Alias = ""
		edited.Audience = nil
		edited.AppliedAt = at.Add(time.Hour) // imutável: deve ser ignorado
		edited.UpdatedAt = at.Add(2 * time.Hour)
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: edited}}); err != nil {
			t.Fatalf("update: %v", err)
		}
		got, err := repo.ListBySheets(ctx, []uuid.UUID{sheet})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		want := edited
		want.AppliedAt = at
		if len(got[sheet]) != 1 || !reflect.DeepEqual(got[sheet][0], want) {
			t.Fatalf("got %+v, want %+v", got[sheet], want)
		}
	})

	t.Run("delete removes and is idempotent", func(t *testing.T) {
		sheet := setup(t)
		e := prone(sheet, now())
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e}}); err != nil {
			t.Fatalf("insert: %v", err)
		}
		del := []statuseffect.Write{{Delete: true, Effect: statuseffect.Effect{ID: e.ID}}}
		if err := repo.Write(ctx, del); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if err := repo.Write(ctx, del); err != nil {
			t.Fatalf("second delete: %v", err)
		}
		if n := total(t); n != 0 {
			t.Fatalf("rows = %d, want 0", n)
		}
	})

	t.Run("ops apply in order", func(t *testing.T) {
		sheet := setup(t)
		a, b := prone(sheet, now()), prone(sheet, now())
		// A e B têm a mesma identidade: só cabe B se A for apagado antes.
		err := repo.Write(ctx, []statuseffect.Write{
			{Effect: a},
			{Delete: true, Effect: statuseffect.Effect{ID: a.ID}},
			{Effect: b},
		})
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		got, _ := repo.ListBySheets(ctx, []uuid.UUID{sheet})
		if len(got[sheet]) != 1 || got[sheet][0].ID != b.ID {
			t.Fatalf("got %+v, want only B", got[sheet])
		}
	})

	t.Run("same identity with another uuid replaces the row", func(t *testing.T) {
		sheet := setup(t)
		old := prone(sheet, now())
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: old}}); err != nil {
			t.Fatalf("first write: %v", err)
		}
		fresh := prone(sheet, now().Add(time.Minute))
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: fresh}}); err != nil {
			t.Fatalf("second write: %v", err)
		}
		got, _ := repo.ListBySheets(ctx, []uuid.UUID{sheet})
		if len(got[sheet]) != 1 || got[sheet][0].ID != fresh.ID {
			t.Fatalf("got %+v, want only the last write", got[sheet])
		}
	})

	t.Run("a deleted sheet is skipped, not an error", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		player := pgtest.InsertTestUser(t, pool, "pl-se-del", "pl-se-del@test.com", "pass")
		alive := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Gon"))
		err := repo.Write(ctx, []statuseffect.Write{
			{Effect: prone(uuid.New(), now())}, // ficha apagada
			{Effect: prone(alive, now())},
		})
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		if n := total(t); n != 1 {
			t.Fatalf("rows = %d, want 1 (only the live sheet)", n)
		}
	})

	t.Run("atomic", func(t *testing.T) {
		sheet := setup(t)
		err := repo.Write(ctx, []statuseffect.Write{
			{Effect: prone(sheet, now())},
			{Effect: statuseffect.Effect{ID: uuid.New(), CharacterID: sheet,
				Kind: statuseffect.Kind(strings.Repeat("x", 40)), AppliedAt: now(), UpdatedAt: now()}}, // kind > VARCHAR(32)
		})
		if err == nil {
			t.Fatal("expected an error")
		}
		if n := total(t); n != 0 {
			t.Fatalf("rows = %d, want 0", n)
		}
	})

	t.Run("runs inside an outer transaction", func(t *testing.T) {
		sheet := setup(t)
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := pgstatuseffect.NewRepository(tx).Write(ctx,
			[]statuseffect.Write{{Effect: prone(sheet, now())}}); err != nil {
			t.Fatalf("write in tx: %v", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("rollback: %v", err)
		}
		if n := total(t); n != 0 {
			t.Fatalf("rows = %d, want 0", n)
		}
	})
}

func TestListBySheets(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	t.Run("groups by sheet and orders", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		player := pgtest.InsertTestUser(t, pool, "pl-se2", "pl-se2@test.com", "pass")
		s1 := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Gon"))
		s2 := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Killua"))
		s3 := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Kurapika"))
		at := now()
		late := prone(s1, at.Add(time.Minute))
		early := statuseffect.Effect{ID: uuid.New(), CharacterID: s1, Kind: statuseffect.KindAirborne, AppliedAt: at, UpdatedAt: at}
		other := prone(s2, at)
		unrelated := prone(s3, at)
		err := repo.Write(ctx, []statuseffect.Write{{Effect: late}, {Effect: early}, {Effect: other}, {Effect: unrelated}})
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListBySheets(ctx, []uuid.UUID{s1, s2})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(got) != 2 || len(got[s1]) != 2 || len(got[s2]) != 1 {
			t.Fatalf("grouping wrong: %+v", got)
		}
		if got[s1][0].ID != early.ID || got[s1][1].ID != late.ID {
			t.Fatalf("order wrong: %+v", got[s1])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := repo.ListBySheets(ctx, nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v; want empty non-nil map", got, err)
		}
	})

	t.Run("round-trips every params field", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		player := pgtest.InsertTestUser(t, pool, "pl-se3", "pl-se3@test.com", "pass")
		sheet := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Gon"))
		ctrl := uuid.New()
		at := now()
		full := Full(sheet, at, ctrl)
		if err := repo.Write(ctx, full); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListBySheets(ctx, []uuid.UUID{sheet})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		byID := map[uuid.UUID]statuseffect.Effect{}
		for _, e := range got[sheet] {
			byID[e.ID] = e
		}
		for _, w := range full {
			if !reflect.DeepEqual(byID[w.Effect.ID], w.Effect) {
				t.Errorf("kind %s: got %+v, want %+v", w.Effect.Kind, byID[w.Effect.ID], w.Effect)
			}
		}
	})
}

// Full devolve efeitos que, juntos, usam todos os campos de Params, as duas variantes e as
// três formas de audiência.
func Full(sheet uuid.UUID, at time.Time, ctrl uuid.UUID) []statuseffect.Write {
	p1, p2 := uuid.New(), uuid.New()
	mk := func(k statuseffect.Kind, f func(*statuseffect.Effect)) statuseffect.Write {
		e := statuseffect.Effect{ID: uuid.New(), CharacterID: sheet, Kind: k, AppliedAt: at, UpdatedAt: at}
		f(&e)
		return statuseffect.Write{Effect: e}
	}
	return []statuseffect.Write{
		mk(statuseffect.KindParalyzed, func(e *statuseffect.Effect) {
			e.Params = statuseffect.Params{Domains: statuseffect.AllDomains(), Points: 4, Irresistible: true}
			e.Audience = &statuseffect.Audience{Players: []uuid.UUID{p1, p2}}
		}),
		mk(statuseffect.KindInjured, func(e *statuseffect.Effect) {
			e.Params = statuseffect.Params{Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{
				statuseffect.DomainAttack: statuseffect.InjuryHurt,
				statuseffect.DomainDodge:  statuseffect.InjuryDisabled,
			}}
			e.Audience = &statuseffect.Audience{Players: []uuid.UUID{}}
		}),
		mk(statuseffect.KindSlowed, func(e *statuseffect.Effect) { e.Params = statuseffect.Params{Penalty: 2} }),
		mk(statuseffect.KindSense, func(e *statuseffect.Effect) {
			e.Sense = statuseffect.SenseTact
			e.Params = statuseffect.Params{
				Level: statuseffect.SenseImpaired, Disadvantage: true,
				Faces: []statuseffect.TouchFace{statuseffect.FaceBalance, statuseffect.FaceTouch},
			}
		}),
		mk(statuseffect.KindWeakened, func(e *statuseffect.Effect) {
			e.Aptitude = statuseffect.AptitudeMental
			e.Params = statuseffect.Params{Disadvantage: true}
			e.Alias = "fadiga"
		}),
		mk(statuseffect.KindCompelled, func(e *statuseffect.Effect) {
			e.Params = statuseffect.Params{Conduct: "obedecer o mestre"}
		}),
		mk(statuseffect.KindControlled, func(e *statuseffect.Effect) {
			e.Params = statuseffect.Params{ControllerID: &ctrl}
			e.Audience = &statuseffect.Audience{Everyone: true}
		}),
	}
}
