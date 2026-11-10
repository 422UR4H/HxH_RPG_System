package round

import (
	"context"
	"errors"
	"fmt"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	pgclock "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/clock"
	pgmatchevent "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchevent"
	pgsheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrMatchEndRefused: a partida não existe ou já foi encerrada.
var ErrMatchEndRefused = errors.New("match not found or already ended")

// naiveStory entrega o horário da história como relógio de parede UTC (colunas TIMESTAMP/DATE).
func naiveStory(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), u.Hour(), u.Minute(), u.Second(), u.Nanosecond(), time.UTC)
}

// PersistMatchEnd grava o fim da partida numa transação (spec 2026-11-08 §5): fins de round ainda
// não gravados e o par ativo, fechados; o relógio parado no instante final; o clockRealTime da
// pausa; as barras das fichas que um fechamento não gravou; matches.game_end_at/story_end_at/brief_final_description (recusa partida já encerrada);
// campaigns.story_current_at e o story_current_at de toda ficha participante avançam ao
// instante final (a data, nas fichas), sem nunca regredir.
//
// Vive no pacote round, não no match: precisa do upsert do par (ensureSceneAndRound) e recebe
// tipos de application/match, que já importa gateway/pg/match (ciclo).
func (r *Repository) PersistMatchEnd(ctx context.Context, d appmatch.MatchEndData) error {
	if err := validateRoundEnds(append(append([]appmatch.RoundEnd(nil), d.UnwrittenRoundEnds...), d.End)); err != nil {
		return fmt.Errorf("PersistMatchEnd: %w", err)
	}
	if d.End.Scene.GetFinishedAt() == nil {
		return fmt.Errorf("PersistMatchEnd: the scene must be closed")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PersistMatchEnd begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	storyAt := naiveStory(d.StoryAt)
	storyDate := time.Date(storyAt.Year(), storyAt.Month(), storyAt.Day(), 0, 0, 0, 0, time.UTC)

	// A partida primeiro: a recusa (já encerrada) não pode deixar nada para trás.
	tag, err := tx.Exec(ctx,
		`UPDATE matches SET game_end_at = $2, story_end_at = $3, brief_final_description = $4, updated_at = $5
		 WHERE uuid = $1 AND game_end_at IS NULL`,
		d.MatchUUID, d.GameEndAt.UTC(), storyDate, d.BriefFinalDescription, d.GameEndAt,
	)
	if err != nil {
		return fmt.Errorf("PersistMatchEnd match: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("PersistMatchEnd %s: %w", d.MatchUUID, ErrMatchEndRefused)
	}

	for _, end := range d.UnwrittenRoundEnds {
		if err := ensureSceneAndRound(ctx, tx, d.MatchUUID, end.Scene, end.Round); err != nil {
			return fmt.Errorf("PersistMatchEnd unwritten round end: %w", err)
		}
	}
	if err := ensureSceneAndRound(ctx, tx, d.MatchUUID, d.End.Scene, d.End.Round); err != nil {
		return fmt.Errorf("PersistMatchEnd: %w", err)
	}
	if err := pgclock.UpsertTx(ctx, tx, d.MatchUUID, d.Clock, time.Now()); err != nil {
		return fmt.Errorf("PersistMatchEnd clock: %w", err)
	}
	if d.RealTimeEvent != nil {
		if err := pgmatchevent.NewRepository(tx).Insert(ctx, *d.RealTimeEvent); err != nil {
			return fmt.Errorf("PersistMatchEnd clock event: %w", err)
		}
	}

	// O HP que um fechamento de turno não gravou: o fim é a última escrita da partida.
	if len(d.StatusBars) > 0 {
		sheets := pgsheet.NewRepository(tx)
		for _, sb := range d.StatusBars {
			if err := sheets.UpdateStatusBars(ctx, sb.CharacterID.String(), sb.Health, sb.Stamina, sb.Aura); err != nil {
				return fmt.Errorf("PersistMatchEnd status bars of %s: %w", sb.CharacterID, err)
			}
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE campaigns SET story_current_at = GREATEST(COALESCE(story_current_at, $2), $2)
		 WHERE uuid = (SELECT campaign_uuid FROM matches WHERE uuid = $1)`,
		d.MatchUUID, storyAt,
	); err != nil {
		return fmt.Errorf("PersistMatchEnd campaign story: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE character_sheets cs SET story_current_at = GREATEST(COALESCE(cs.story_current_at, $2::date), $2::date)
		 FROM match_participants mp
		 WHERE mp.match_uuid = $1 AND mp.character_sheet_uuid = cs.uuid`,
		d.MatchUUID, storyDate,
	); err != nil {
		return fmt.Errorf("PersistMatchEnd participant sheets: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("PersistMatchEnd commit: %w", err)
	}
	return nil
}

// ReadMatchEnd devolve o fim gravado (game_end_at e o story_at do relógio), ou (nil, nil) quando a
// partida não foi encerrada.
func (r *Repository) ReadMatchEnd(ctx context.Context, matchUUID uuid.UUID) (*appmatch.MatchEndState, error) {
	var gameEnd, storyAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT m.game_end_at, c.story_at
		 FROM matches m LEFT JOIN match_clocks c ON c.match_uuid = m.uuid
		 WHERE m.uuid = $1`,
		matchUUID,
	).Scan(&gameEnd, &storyAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("ReadMatchEnd: %w", err)
	}
	if gameEnd == nil {
		return nil, nil
	}
	out := &appmatch.MatchEndState{GameEndAt: gameEnd.UTC()}
	if storyAt != nil {
		s := naiveStory(*storyAt)
		out.StoryAt = &s
	}
	return out, nil
}

var _ appmatch.IMatchEndRepository = (*Repository)(nil)
