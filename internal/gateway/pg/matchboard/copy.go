// internal/gateway/pg/matchboard/copy.go
package pgmatchboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/fog"
)

// Copy gives a match's board the same board another match ended with (spec §4.3, B16 —
// "uma partida pode começar de onde outra terminou"). It is a straight row copy, not a
// diff: the destination's board row and player_memories are entirely replaced by the
// source's, with only match_uuid/match_id (and, for player_memories, the surrogate id)
// swapped — same "portrait" semantics as Save/Get.
//
// Both tables are copied in ONE transaction: a crash between them would otherwise leave a
// board without its fog, or vice versa, and a reload is supposed to bring them back together
// (spec §4.3, "O fog do jogador vai junto"). r.q.Begin is available because pgfs.IQuerier
// includes Begin, and the returned pgx.Tx satisfies pgfs.IQuerier itself (same Exec/Query/
// QueryRow/Begin signatures) — so fog.NewPlayerMemoryRepository(tx) runs its CopyMatch
// inside the very same transaction as this method's own board-row copy, without either
// package needing a transaction type of its own.
func (r *Repository) Copy(ctx context.Context, src, dst uuid.UUID) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("copy match board begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	mapUUID, err := copyBoardRow(ctx, tx, src, dst)
	if err != nil {
		return err
	}

	// tx satisfies pgfs.IQuerier, so the fog copy runs inside this same transaction.
	// mapUUID (the source board's own map, just read back via RETURNING) restricts the fog
	// copy to that map, so a stray player_memories row of src's on some OTHER map never
	// rides along (fix round 1, review finding).
	fogRepo := fog.NewPlayerMemoryRepository(tx)
	if err := fogRepo.CopyMatch(ctx, src, dst, mapUUID); err != nil {
		return fmt.Errorf("copy match board: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("copy match board commit: %w", err)
	}
	return nil
}

// copyBoardRow deletes dst's board row (if any — "replace, not merge", same as a fresh
// Save) and replaces it with a copy of src's row, keeping everything but match_uuid. It
// returns the copied row's map_uuid so the caller can restrict the matching fog copy to
// that same map.
func copyBoardRow(ctx context.Context, tx pgx.Tx, src, dst uuid.UUID) (uuid.UUID, error) {
	const del = `DELETE FROM match_boards WHERE match_uuid = $1`
	if _, err := tx.Exec(ctx, del, dst); err != nil {
		return uuid.Nil, fmt.Errorf("copy match board: delete dst: %w", err)
	}

	const ins = `
		INSERT INTO match_boards (match_uuid, map_uuid, grid, bg, pieces, walls, updated_at)
		SELECT $1, map_uuid, grid, bg, pieces, walls, updated_at
		FROM match_boards WHERE match_uuid = $2
		RETURNING map_uuid
	`
	var mapUUID uuid.UUID
	err := tx.QueryRow(ctx, ins, dst, src).Scan(&mapUUID)
	if err != nil {
		// Defense in depth: the use case already checks the source has a board
		// (ErrSourceMatchHasNoBoard) before calling Copy, but a caller that skips that
		// check should get a clear error instead of a silent no-op. INSERT…SELECT inserts
		// (and so RETURNs) zero rows when the SELECT side is empty, which QueryRow/Scan
		// reports as pgx.ErrNoRows, same as an ordinary empty SELECT would.
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("copy match board: source match %s has no board row", src)
		}
		return uuid.Nil, fmt.Errorf("copy match board: insert: %w", err)
	}
	return mapUUID, nil
}
