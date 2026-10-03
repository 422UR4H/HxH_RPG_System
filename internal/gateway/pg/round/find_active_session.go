package round

import (
	"context"
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FindActiveSession returns the current unfinished scene+round for the given match,
// or nil if none exists. Ordered newest first: should drift ever leave more than one open
// round (or scene), the one born last is the one the table is in — and LIMIT 1 alone would
// pick whichever the plan happened to return first.
func (r *Repository) FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*matchsession.ActiveSessionData, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT s.uuid, s.category, s.brief_initial_description, s.created_at,
		        ro.uuid, ro.mode, ro.created_at
		 FROM scenes s
		 JOIN rounds ro ON ro.scene_uuid = s.uuid
		 WHERE s.match_uuid = $1
		   AND s.finished_at IS NULL
		   AND ro.finished_at IS NULL
		 ORDER BY s.created_at DESC, ro.created_at DESC, ro.uuid DESC
		 LIMIT 1`,
		matchUUID,
	)

	data := &matchsession.ActiveSessionData{}
	err := row.Scan(
		&data.SceneID, &data.Category, &data.BriefInitDesc, &data.SceneCreatedAt,
		&data.RoundID, &data.Mode, &data.RoundCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindActiveSession: %w", err)
	}
	return data, nil
}
