package round

import (
	"context"
	"fmt"

	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// execer is the one method ensureSceneAndRound needs, satisfied by the pool AND by a pgx.Tx —
// which is what lets PersistTurnClose run the very same inserts inside its own transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// EnsureSceneAndRound writes the scene and the round as rows, if they are not rows yet
// (spec §4.5, §4.8). Idempotent: calling it any number of times for the same pair writes each
// once, and a later PersistTurnClose of the same round — which runs these same two inserts —
// does not fail over them either.
//
// It exists because the scene and round used to become rows only when their first turn
// closed, and both are now rows from the moment they are born (start_match, rehydration,
// change_scene, a round closing and the next one opening) — master_actions and match_events
// reference both.
//
// The scene is ON CONFLICT DO NOTHING; the round refreshes its mode. Being a row from birth
// means a round is written in the regime it was born in, and the master may switch it after
// that: the change_round_mode arm calls this again, and a turn closing does too, so the row
// always carries the regime the round was last seen in — what the history's round.mode says.
// Which regime it passed through, and when, is match_events' job (roundModeChanged).
//
// Both carry finished_at when the entity already has one, and a finish already on the row is
// never moved (COALESCE keeps the first). A round whose birth write failed and that nothing
// else marked as a row is closed in memory by exhaustion with no CloseRound (there was no row
// to close); its last turn's PersistTurnClose is then its first write, and it must land
// closed — or the scene would carry two open rounds, it and the one born right after.
func (r *Repository) EnsureSceneAndRound(
	ctx context.Context, matchUUID uuid.UUID, sc *sceneentity.Scene, rd *roundentity.Round,
) error {
	return ensureSceneAndRound(ctx, r.pool, matchUUID, sc, rd)
}

func ensureSceneAndRound(
	ctx context.Context, q execer, matchUUID uuid.UUID, sc *sceneentity.Scene, rd *roundentity.Round,
) error {
	if sc == nil || rd == nil {
		return fmt.Errorf("EnsureSceneAndRound: scene and round are required")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO scenes (uuid, match_uuid, category, brief_initial_description, created_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (uuid) DO UPDATE SET finished_at = COALESCE(scenes.finished_at, EXCLUDED.finished_at)`,
		sc.GetID(), matchUUID, string(sc.GetCategory()), sc.BriefInitialDescription, sc.GetCreatedAt(),
		sc.GetFinishedAt(),
	)
	if err != nil {
		return fmt.Errorf("insert scene: %w", err)
	}

	_, err = q.Exec(ctx,
		`INSERT INTO rounds (uuid, scene_uuid, mode, created_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uuid) DO UPDATE SET
		   mode = EXCLUDED.mode,
		   finished_at = COALESCE(rounds.finished_at, EXCLUDED.finished_at)`,
		rd.GetID(), sc.GetID(), string(rd.GetMode()), rd.GetCreatedAt(), rd.GetFinishedAt(),
	)
	if err != nil {
		return fmt.Errorf("insert round: %w", err)
	}
	return nil
}
