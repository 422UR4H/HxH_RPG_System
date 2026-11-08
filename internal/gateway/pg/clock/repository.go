package pgclock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persiste o relógio da história (match_clocks, uma linha por partida).
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
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
// cena...): o relógio e o que o moveu ficam duráveis juntos ou não ficam.
func UpsertTx(ctx context.Context, tx pgx.Tx, matchUUID uuid.UUID, snap clock.Snapshot) error {
	var runningSince *time.Time
	if snap.Running && snap.RunningSince != nil {
		r := snap.RunningSince.UTC()
		runningSince = &r
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO match_clocks (match_uuid, story_at, limit_at, limit_reason, running_since, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (match_uuid) DO UPDATE SET
		   story_at = EXCLUDED.story_at,
		   limit_at = EXCLUDED.limit_at,
		   limit_reason = EXCLUDED.limit_reason,
		   running_since = EXCLUDED.running_since,
		   updated_at = now()`,
		matchUUID, naive(snap.StoryAt), naive(snap.LimitAt), string(snap.LimitReason), runningSince,
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
