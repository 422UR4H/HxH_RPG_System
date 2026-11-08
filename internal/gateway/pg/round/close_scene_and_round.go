package round

import (
	"context"
	"fmt"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	pgclock "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/clock"
	"github.com/google/uuid"
)

// CloseSceneAndRound fecha a cena e o round que o change_scene encerrou, numa transação: o
// finished_at dos dois, o resumo final e o fim na história da cena (brief_final_description,
// story_finished_at), o counted/drained do round e — se clk != nil — o relógio (spec §4: "um
// comando, uma transação"). end leva os dois já fechados (snapshots).
//
// Vai pelo mesmo upsert idempotente de toda escrita de cena/round (ensureSceneAndRound): um
// finished_at já gravado não se move, e um par que nunca virou linha é escrito já fechado.
func (r *Repository) CloseSceneAndRound(
	ctx context.Context, matchUUID uuid.UUID, end appmatch.RoundEnd, clk *clock.Snapshot,
) error {
	if err := validateRoundEnds([]appmatch.RoundEnd{end}); err != nil {
		return fmt.Errorf("CloseSceneAndRound: %w", err)
	}
	if end.Scene.GetFinishedAt() == nil {
		return fmt.Errorf("CloseSceneAndRound: the scene must be closed")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("CloseSceneAndRound begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	if err := ensureSceneAndRound(ctx, tx, matchUUID, end.Scene, end.Round); err != nil {
		return fmt.Errorf("CloseSceneAndRound: %w", err)
	}
	if clk != nil {
		if err := pgclock.UpsertTx(ctx, tx, matchUUID, *clk); err != nil {
			return fmt.Errorf("CloseSceneAndRound clock: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("CloseSceneAndRound commit: %w", err)
	}
	return nil
}
