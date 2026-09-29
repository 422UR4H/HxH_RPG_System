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
// (spec §4.5, §4.8). Idempotent: ON CONFLICT DO NOTHING on both, so calling it any number of
// times for the same pair writes each once, and a later PersistTurnClose of the same round —
// which runs these same two inserts — does not fail over them either.
//
// It exists because the scene and round used to become rows only when their first turn
// closed, and a master action is recorded the instant it happens, with or without a turn:
// master_actions references both.
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
		`INSERT INTO scenes (uuid, match_uuid, category, brief_initial_description, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uuid) DO NOTHING`,
		sc.GetID(), matchUUID, string(sc.GetCategory()), sc.BriefInitialDescription, sc.GetCreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("insert scene: %w", err)
	}

	_, err = q.Exec(ctx,
		`INSERT INTO rounds (uuid, scene_uuid, mode, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (uuid) DO NOTHING`,
		rd.GetID(), sc.GetID(), string(rd.GetMode()), rd.GetCreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("insert round: %w", err)
	}
	return nil
}
