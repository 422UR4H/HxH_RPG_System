package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AddNPCParticipant inserts a master-controlled NPC (character sheet with no
// player_uuid) directly into a match's roster, bypassing enrollments. The
// NPC eligibility guard lives in the SQL (player_uuid IS NULL) so an upper
// layer bug cannot delete or add a player's roster row through this path.
func (r *Repository) AddNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time,
) (*matchEntity.Participant, error) {
	const query = `
		INSERT INTO match_participants
			(uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
		SELECT gen_random_uuid(), $1, cs.uuid, $3, $4, $4
		FROM character_sheets cs
		WHERE cs.uuid = $2 AND cs.player_uuid IS NULL
		ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
		RETURNING uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at
	`
	now := time.Now()

	var p matchEntity.Participant
	err := r.q.QueryRow(ctx, query, matchUUID, sheetUUID, joinedAt, now).Scan(
		&p.UUID, &p.MatchUUID, &p.Sheet.UUID, &p.JoinedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, r.disambiguateAddNPCParticipant(ctx, matchUUID, sheetUUID)
		}
		return nil, fmt.Errorf("failed to add npc participant: %w", err)
	}
	return &p, nil
}

// disambiguateAddNPCParticipant runs when the INSERT above returns no row.
// That happens for two distinct causes that the SELECT ... WHERE / ON
// CONFLICT collapse into the same pgx.ErrNoRows: the sheet is already a
// participant (conflict), or the sheet is not an eligible NPC (WHERE
// filtered it out). Each needs a different error.
func (r *Repository) disambiguateAddNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID,
) error {
	const existsQuery = `
		SELECT EXISTS (
			SELECT 1 FROM match_participants
			WHERE match_uuid = $1 AND character_sheet_uuid = $2
		)
	`
	var exists bool
	if err := r.q.QueryRow(ctx, existsQuery, matchUUID, sheetUUID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to disambiguate npc participant insert: %w", err)
	}
	if exists {
		return ErrNPCAlreadyInMatch
	}
	return fmt.Errorf("%w: %w", ErrSheetNotEligibleNPC, pgx.ErrNoRows)
}
