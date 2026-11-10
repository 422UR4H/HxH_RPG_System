package pgclock

import (
	"context"
	"errors"
	"fmt"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	pgmatchevent "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchevent"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persiste o relógio da história (match_clocks, uma linha por partida).
type Repository struct {
	pool *pgxpool.Pool
	// now é a fonte do updated_at (tempo do Go, não now() do SQL — gateway-conventions).
	now func() time.Time
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, now: time.Now}
}

// naive entrega o horário como relógio de parede UTC. As colunas story_at e limit_at são
// TIMESTAMP (sem fuso): o valor gravado nunca pode depender do TimeZone da sessão.
func naive(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	n := time.Date(u.Year(), u.Month(), u.Day(), u.Hour(), u.Minute(), u.Second(), u.Nanosecond(), time.UTC)
	return &n
}

// UpsertTx grava o checkpoint do relógio dentro da transação de quem chama (turno, round,
// cena...): o relógio e o que o moveu ficam duráveis juntos ou não ficam. updatedAt vem do Go
// (gateway-conventions: nada de now() no SQL).
func UpsertTx(ctx context.Context, tx pgx.Tx, matchUUID uuid.UUID, snap clock.Snapshot, updatedAt time.Time) error {
	var runningSince *time.Time
	if snap.Running && snap.RunningSince != nil {
		r := snap.RunningSince.UTC()
		runningSince = &r
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO match_clocks (match_uuid, story_at, limit_at, limit_reason, running_since, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (match_uuid) DO UPDATE SET
		   story_at = EXCLUDED.story_at,
		   limit_at = EXCLUDED.limit_at,
		   limit_reason = EXCLUDED.limit_reason,
		   running_since = EXCLUDED.running_since,
		   updated_at = EXCLUDED.updated_at`,
		matchUUID, naive(snap.StoryAt), naive(snap.LimitAt), string(snap.LimitReason), runningSince, updatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert match clock: %w", err)
	}
	return nil
}

// rowQuerier é o que a leitura precisa, satisfeito pelo pool e por pgx.Tx.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// LoadFrom lê o checkpoint em q; nil quando a partida não tem linha de relógio.
func LoadFrom(ctx context.Context, q rowQuerier, matchUUID uuid.UUID) (*clock.Snapshot, error) {
	var storyAt, limitAt, runningSince *time.Time
	var reason string
	err := q.QueryRow(ctx,
		`SELECT story_at, limit_at, limit_reason, running_since FROM match_clocks WHERE match_uuid = $1`,
		matchUUID,
	).Scan(&storyAt, &limitAt, &reason, &runningSince)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match clock: %w", err)
	}
	snap := &clock.Snapshot{
		StoryAt: naive(storyAt), LimitAt: naive(limitAt), LimitReason: clock.LimitReason(reason),
	}
	if runningSince != nil {
		r := runningSince.UTC()
		snap.RunningSince = &r
		snap.Running = true
	}
	return snap, nil
}

func (r *Repository) Load(ctx context.Context, matchUUID uuid.UUID) (*clock.Snapshot, error) {
	return LoadFrom(ctx, r.pool, matchUUID)
}

// Save grava o checkpoint e, se ev != nil, o evento, numa transação só: o relógio e o que o
// moveu ficam duráveis juntos ou não ficam. Com ev.MergeInto, atualiza o payload dessa linha
// de match_events (só se for da mesma partida) em vez de inserir ev.Event.
func (r *Repository) Save(
	ctx context.Context, matchUUID uuid.UUID, snap clock.Snapshot, ev *appmatch.ClockEvent,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin clock save: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	if err := UpsertTx(ctx, tx, matchUUID, snap, r.now()); err != nil {
		return err
	}
	if ev != nil {
		if ev.MergeInto != uuid.Nil {
			payload := ev.Event.Payload
			if payload == nil {
				payload = []byte("{}")
			}
			tag, err := tx.Exec(ctx,
				`UPDATE match_events SET payload = $1 WHERE uuid = $2 AND match_uuid = $3`,
				[]byte(payload), ev.MergeInto, matchUUID,
			)
			if err != nil {
				return fmt.Errorf("merge match event: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("merge match event %s: not found in match %s", ev.MergeInto, matchUUID)
			}
		} else if err := pgmatchevent.NewRepository(tx).Insert(ctx, ev.Event); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit clock save: %w", err)
	}
	return nil
}

var _ appmatch.IClockRepository = (*Repository)(nil)
