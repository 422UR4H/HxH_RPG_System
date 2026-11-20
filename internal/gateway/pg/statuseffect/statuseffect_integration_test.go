//go:build integration

package pgstatuseffect_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
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

// eventFixture monta mestre, jogador, ficha e partida para os testes de evento.
type eventFixture struct {
	player, sheet, match uuid.UUID
}

func newEventFixture(t *testing.T, pool *pgxpool.Pool, tag string) eventFixture {
	t.Helper()
	pgtest.TruncateAll(t, pool)
	master := pgtest.InsertTestUser(t, pool, "gm-"+tag, "gm-"+tag+"@test.com", "pass")
	player := pgtest.InsertTestUser(t, pool, "pl-"+tag, "pl-"+tag+"@test.com", "pass")
	campaign := pgtest.InsertTestCampaign(t, pool, master, "Camp "+tag)
	match := pgtest.InsertTestMatch(t, pool, master, campaign, "Match "+tag)
	sheet := pgtest.InsertTestCharacterSheet(t, pool, &player, nil, nil, "Gon")
	return eventFixture{player: uuid.MustParse(player), sheet: uuid.MustParse(sheet), match: uuid.MustParse(match)}
}

func (f eventFixture) event(change statuseffect.Change, at time.Time) *statuseffect.Event {
	return &statuseffect.Event{
		ID: uuid.New(), Change: change, MatchID: f.match, SceneID: uuid.New(), RoundID: uuid.New(),
		Listing:    statuseffect.Audience{Everyone: true},
		Views:      map[uuid.UUID]masteraction.View{f.player: masteraction.ViewFull},
		HappenedAt: at,
	}
}

func countEvents(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM status_effect_events`).Scan(&n); err != nil {
		t.Fatalf("count events: %v", err)
	}
	return n
}

func TestWriteEvents(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	t.Run("event rides its op", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev1")
		at := now()
		ctrl := uuid.New()
		e := statuseffect.Effect{
			ID: uuid.New(), CharacterID: f.sheet, Kind: statuseffect.KindControlled,
			Params: statuseffect.Params{ControllerID: &ctrl}, Alias: "marionete",
			Audience:  &statuseffect.Audience{Players: []uuid.UUID{f.player}},
			AppliedAt: at, UpdatedAt: at,
		}
		ev := f.event(statuseffect.ChangeApplied, at)
		ev.Listing = statuseffect.Audience{Players: []uuid.UUID{}}
		action, hid := uuid.New(), uuid.New()
		other := uuid.New()
		ev.ActionID = &action
		ev.HiddenFrom = map[uuid.UUID]uuid.UUID{other: hid}
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e, Event: ev}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("events = %d, want 1", len(got))
		}
		if !reflect.DeepEqual(got[0].Effect, e) {
			t.Errorf("snapshot = %+v, want %+v", got[0].Effect, e)
		}
		want := *ev
		if !reflect.DeepEqual(got[0].Event, want) {
			t.Errorf("event = %+v, want %+v", got[0].Event, want)
		}
		if got[0].Event.TurnID != nil {
			t.Errorf("turn = %v, want nil", got[0].Event.TurnID)
		}
		if live, _ := repo.ListBySheets(ctx, []uuid.UUID{f.sheet}); len(live[f.sheet]) != 1 {
			t.Errorf("effect row missing: %+v", live)
		}
	})

	t.Run("no hidden_from and no action are stored as NULL and read as nil", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev1b")
		ev := f.event(statuseffect.ChangeApplied, now())
		ev.HiddenFrom = map[uuid.UUID]uuid.UUID{} // vazio grava NULL, como ausente
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: prone(f.sheet, now()), Event: ev}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		var hiddenNull, actionNull bool
		if err := pool.QueryRow(ctx,
			`SELECT hidden_from IS NULL, action_uuid IS NULL FROM status_effect_events`).Scan(&hiddenNull, &actionNull); err != nil {
			t.Fatalf("read: %v", err)
		}
		if !hiddenNull || !actionNull {
			t.Fatalf("hidden_from NULL=%v action_uuid NULL=%v, want both SQL NULL", hiddenNull, actionNull)
		}
		got, _ := repo.ListEventsByMatch(ctx, f.match)
		if len(got) != 1 || got[0].Event.HiddenFrom != nil || got[0].Event.ActionID != nil {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a failing op writes no event", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev2")
		bad := statuseffect.Effect{ID: uuid.New(), CharacterID: f.sheet,
			Kind: statuseffect.Kind(strings.Repeat("x", 40)), AppliedAt: now(), UpdatedAt: now()}
		err := repo.Write(ctx, []statuseffect.Write{
			{Effect: prone(f.sheet, now()), Event: f.event(statuseffect.ChangeApplied, now())},
			{Effect: bad, Event: f.event(statuseffect.ChangeApplied, now())},
		})
		if err == nil {
			t.Fatal("expected an error")
		}
		if n := countEvents(t, pool); n != 0 {
			t.Fatalf("events = %d, want 0", n)
		}
	})

	t.Run("a skipped sheet skips its event", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev3")
		gone := prone(uuid.New(), now())
		err := repo.Write(ctx, []statuseffect.Write{
			{Effect: gone, Event: f.event(statuseffect.ChangeApplied, now())},
			{Delete: true, Effect: gone, Event: f.event(statuseffect.ChangeRemoved, now())},
			{Effect: prone(f.sheet, now()), Event: f.event(statuseffect.ChangeApplied, now())},
		})
		if err != nil {
			t.Fatalf("write: %v", err)
		}
		if n := countEvents(t, pool); n != 1 {
			t.Fatalf("events = %d, want 1 (only the live sheet)", n)
		}
	})

	t.Run("a deleted match skips its event", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev3b")
		ev := f.event(statuseffect.ChangeApplied, now())
		ev.MatchID = uuid.New()
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: prone(f.sheet, now()), Event: ev}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		if n := countEvents(t, pool); n != 0 {
			t.Fatalf("events = %d, want 0", n)
		}
	})

	t.Run("a repeated event id is ignored", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev4")
		w := statuseffect.Write{Effect: prone(f.sheet, now()), Event: f.event(statuseffect.ChangeApplied, now())}
		if err := repo.Write(ctx, []statuseffect.Write{w}); err != nil {
			t.Fatalf("first: %v", err)
		}
		if err := repo.Write(ctx, []statuseffect.Write{w}); err != nil {
			t.Fatalf("second: %v", err)
		}
		if n := countEvents(t, pool); n != 1 {
			t.Fatalf("events = %d, want 1", n)
		}
	})

	t.Run("runs inside an outer transaction", func(t *testing.T) {
		f := newEventFixture(t, pool, "ev5")
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := pgstatuseffect.NewRepository(tx).Write(ctx, []statuseffect.Write{
			{Effect: prone(f.sheet, now()), Event: f.event(statuseffect.ChangeApplied, now())}}); err != nil {
			t.Fatalf("write in tx: %v", err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("rollback: %v", err)
		}
		if n := countEvents(t, pool); n != 0 {
			t.Fatalf("events = %d, want 0", n)
		}
	})
}

func TestListEventsByMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	t.Run("orders by happened_at then seq", func(t *testing.T) {
		f := newEventFixture(t, pool, "le1")
		at := now()
		e := prone(f.sheet, at)
		late := f.event(statuseffect.ChangeEdited, at.Add(time.Minute))
		first := f.event(statuseffect.ChangeApplied, at)
		second := f.event(statuseffect.ChangeEdited, at) // mesmo instante: vale a ordem de gravação
		if err := repo.Write(ctx, []statuseffect.Write{
			{Effect: e, Event: late}, {Effect: e, Event: first}, {Effect: e, Event: second},
		}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(got) != 3 || got[0].Event.ID != first.ID || got[1].Event.ID != second.ID || got[2].Event.ID != late.ID {
			t.Fatalf("order wrong: %+v", got)
		}
	})

	t.Run("only that match", func(t *testing.T) {
		f := newEventFixture(t, pool, "le2")
		master := pgtest.InsertTestUser(t, pool, "gm-le2b", "gm-le2b@test.com", "pass")
		campaign := pgtest.InsertTestCampaign(t, pool, master, "Camp le2b")
		otherMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, master, campaign, "Other"))
		mine := f.event(statuseffect.ChangeApplied, now())
		theirs := f.event(statuseffect.ChangeApplied, now())
		theirs.MatchID = otherMatch
		if err := repo.Write(ctx, []statuseffect.Write{
			{Effect: prone(f.sheet, now()), Event: mine},
			{Effect: statuseffect.Effect{ID: uuid.New(), CharacterID: f.sheet, Kind: statuseffect.KindAirborne,
				AppliedAt: now(), UpdatedAt: now()}, Event: theirs},
		}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil || len(got) != 1 || got[0].Event.ID != mine.ID {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("removed keeps its snapshot", func(t *testing.T) {
		f := newEventFixture(t, pool, "le3")
		e := statuseffect.Effect{ID: uuid.New(), CharacterID: f.sheet, Kind: statuseffect.KindBleeding,
			Params: statuseffect.Params{HPPerRound: 2}, Alias: "corte", AppliedAt: now(), UpdatedAt: now()}
		if err := repo.Write(ctx, []statuseffect.Write{
			{Effect: e, Event: f.event(statuseffect.ChangeApplied, now())},
			{Delete: true, Effect: e, Event: f.event(statuseffect.ChangeRemoved, now().Add(time.Second))},
		}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil || len(got) != 2 {
			t.Fatalf("got %+v, %v", got, err)
		}
		if got[1].Event.Change != statuseffect.ChangeRemoved || !reflect.DeepEqual(got[1].Effect, e) {
			t.Fatalf("removed = %+v, want snapshot %+v", got[1], e)
		}
		if live, _ := repo.ListBySheets(ctx, []uuid.UUID{f.sheet}); len(live[f.sheet]) != 0 {
			t.Fatalf("effect row still there: %+v", live)
		}
	})

	t.Run("an undecodable row is skipped", func(t *testing.T) {
		f := newEventFixture(t, pool, "le4")
		good := f.event(statuseffect.ChangeApplied, now())
		bad := f.event(statuseffect.ChangeApplied, now())
		if err := repo.Write(ctx, []statuseffect.Write{
			{Effect: prone(f.sheet, now()), Event: good}, {Effect: prone(f.sheet, now()), Event: bad},
		}); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE status_effect_events SET views = '[1]' WHERE uuid = $1`, bad.ID); err != nil {
			t.Fatalf("corrupt: %v", err)
		}
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil || len(got) != 1 || got[0].Event.ID != good.ID {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("no events is an empty list", func(t *testing.T) {
		f := newEventFixture(t, pool, "le5")
		got, err := repo.ListEventsByMatch(ctx, f.match)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

func TestLatestListings(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	t.Run("latest across matches", func(t *testing.T) {
		f := newEventFixture(t, pool, "ll1")
		master := pgtest.InsertTestUser(t, pool, "gm-ll1b", "gm-ll1b@test.com", "pass")
		campaign := pgtest.InsertTestCampaign(t, pool, master, "Camp ll1b")
		matchB := uuid.MustParse(pgtest.InsertTestMatch(t, pool, master, campaign, "B"))
		e := prone(f.sheet, now())
		inA := f.event(statuseffect.ChangeApplied, now())
		inA.Listing = statuseffect.Audience{Players: []uuid.UUID{}}
		inB := f.event(statuseffect.ChangeEdited, now().Add(time.Minute))
		inB.MatchID = matchB
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e, Event: inA}, {Effect: e, Event: inB}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.LatestListings(ctx, []uuid.UUID{e.ID, uuid.New()})
		if err != nil {
			t.Fatalf("latest: %v", err)
		}
		if len(got) != 1 || !got[e.ID].Everyone {
			t.Fatalf("got %+v, want everyone for the effect only", got)
		}
	})

	t.Run("same happened_at resolved by seq", func(t *testing.T) {
		f := newEventFixture(t, pool, "ll2")
		at := now()
		e := prone(f.sheet, at)
		a := f.event(statuseffect.ChangeApplied, at)
		b := f.event(statuseffect.ChangeEdited, at)
		b.Listing = statuseffect.Audience{Players: []uuid.UUID{f.player}}
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e, Event: a}, {Effect: e, Event: b}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, err := repo.LatestListings(ctx, []uuid.UUID{e.ID})
		if err != nil {
			t.Fatalf("latest: %v", err)
		}
		if l := got[e.ID]; l.Everyone || len(l.Players) != 1 || l.Players[0] != f.player {
			t.Fatalf("got %+v, want [player]", l)
		}
	})

	t.Run("an empty list round-trips as master only", func(t *testing.T) {
		f := newEventFixture(t, pool, "ll3")
		e := prone(f.sheet, now())
		ev := f.event(statuseffect.ChangeApplied, now())
		ev.Listing = statuseffect.Audience{Players: []uuid.UUID{}}
		if err := repo.Write(ctx, []statuseffect.Write{{Effect: e, Event: ev}}); err != nil {
			t.Fatalf("write: %v", err)
		}
		got, _ := repo.LatestListings(ctx, []uuid.UUID{e.ID})
		l, ok := got[e.ID]
		if !ok || l.Everyone || l.Players == nil || len(l.Players) != 0 {
			t.Fatalf("got %+v (%v), want players []", l, ok)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := repo.LatestListings(ctx, nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

func TestReveals(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	t.Run("revealed at is the max across matches and rows", func(t *testing.T) {
		f := newEventFixture(t, pool, "rv1")
		master := pgtest.InsertTestUser(t, pool, "gm-rv1b", "gm-rv1b@test.com", "pass")
		campaign := pgtest.InsertTestCampaign(t, pool, master, "Camp rv1b")
		matchB := uuid.MustParse(pgtest.InsertTestMatch(t, pool, master, campaign, "B"))
		eff, other := uuid.New(), uuid.New()
		t0 := now()
		for _, r := range []struct {
			m  uuid.UUID
			id uuid.UUID
			at time.Time
		}{
			{f.match, eff, t0}, {matchB, eff, t0.Add(time.Hour)}, {f.match, eff, t0.Add(time.Minute)},
			{f.match, other, t0.Add(2 * time.Hour)},
		} {
			if err := repo.InsertReveal(ctx, r.m, r.id, r.at); err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
		got, err := repo.RevealedAt(ctx, []uuid.UUID{eff, uuid.New()})
		if err != nil {
			t.Fatalf("revealed at: %v", err)
		}
		if len(got) != 1 || !got[eff].Equal(t0.Add(time.Hour)) {
			t.Fatalf("got %v, want only the effect at t0+1h", got)
		}
	})

	t.Run("empty input is a non-nil empty map", func(t *testing.T) {
		got, err := repo.RevealedAt(ctx, nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})

	t.Run("cascade with the match", func(t *testing.T) {
		f := newEventFixture(t, pool, "rv2")
		eff := uuid.New()
		if err := repo.InsertReveal(ctx, f.match, eff, now()); err != nil {
			t.Fatalf("insert: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM matches WHERE uuid = $1`, f.match); err != nil {
			t.Fatalf("delete match: %v", err)
		}
		got, err := repo.RevealedAt(ctx, []uuid.UUID{eff})
		if err != nil || len(got) != 0 {
			t.Fatalf("got %v, %v, want the reveal gone with its match", got, err)
		}
	})
}

// TestRevealableInMatch (revisão final do PR 2, achado 3): um efeito é revelável numa partida
// quando tem evento em QUALQUER partida e a ficha dele participa desta — um Escondido aplicado
// na partida A e ainda ativo na B é revelável pelo mestre da B; o de uma ficha fora da B, não.
func TestRevealableInMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgstatuseffect.NewRepository(pool)
	ctx := context.Background()

	f := newEventFixture(t, pool, "he1")
	master := pgtest.InsertTestUser(t, pool, "gm-he1b", "gm-he1b@test.com", "pass")
	campaign := pgtest.InsertTestCampaign(t, pool, master, "Camp he1b")
	matchB := uuid.MustParse(pgtest.InsertTestMatch(t, pool, master, campaign, "B"))
	matchC := uuid.MustParse(pgtest.InsertTestMatch(t, pool, master, campaign, "C"))
	// A ficha participa da partida do evento (A) e da B; não da C.
	pgtest.InsertTestMatchParticipant(t, pool, f.match.String(), f.sheet.String(), now())
	pgtest.InsertTestMatchParticipant(t, pool, matchB.String(), f.sheet.String(), now())
	e := prone(f.sheet, now())
	if err := repo.Write(ctx, []statuseffect.Write{{Effect: e, Event: f.event(statuseffect.ChangeApplied, now())}}); err != nil {
		t.Fatalf("write: %v", err)
	}
	for _, c := range []struct {
		name  string
		match uuid.UUID
		eff   uuid.UUID
		want  bool
	}{
		{"event in the match, sheet a participant", f.match, e.ID, true},
		{"event in another match, sheet a participant here", matchB, e.ID, true},
		{"sheet not a participant here", matchC, e.ID, false},
		{"unknown effect", f.match, uuid.New(), false},
		{"nil effect", f.match, uuid.Nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := repo.RevealableInMatch(ctx, c.match, c.eff)
			if err != nil || got != c.want {
				t.Fatalf("got %v, %v, want %v", got, err, c.want)
			}
		})
	}
}
