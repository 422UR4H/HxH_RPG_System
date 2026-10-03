// internal/gateway/pg/matchboard/board.go
package pgmatchboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
)

// Get returns the match's board, or (nil, nil) when the match has no row yet — the
// caller falls back to a fresh snapshot of the attached map without writing anything
// (spec §4.3, "Ciclo de vida").
func (r *Repository) Get(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error) {
	const q = `
		SELECT map_uuid, grid, bg, pieces, walls, updated_at
		FROM match_boards
		WHERE match_uuid = $1
	`
	var mapUUID uuid.UUID
	var gridRaw, bgRaw, piecesRaw, wallsRaw []byte
	var updatedAt time.Time

	err := r.q.QueryRow(ctx, q, matchUUID).Scan(&mapUUID, &gridRaw, &bgRaw, &piecesRaw, &wallsRaw, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get match board: %w", err)
	}

	var grid mapentity.GridShape
	if err := json.Unmarshal(gridRaw, &grid); err != nil {
		return nil, fmt.Errorf("unmarshal match board grid: %w", err)
	}

	var bg *mapentity.BgImage
	if bgRaw != nil && string(bgRaw) != "null" {
		bg = &mapentity.BgImage{}
		if err := json.Unmarshal(bgRaw, bg); err != nil {
			return nil, fmt.Errorf("unmarshal match board bg: %w", err)
		}
	}

	pieces := []mapentity.Piece{}
	if err := json.Unmarshal(piecesRaw, &pieces); err != nil {
		return nil, fmt.Errorf("unmarshal match board pieces: %w", err)
	}

	walls := []mapentity.WallSegment{}
	if err := json.Unmarshal(wallsRaw, &walls); err != nil {
		return nil, fmt.Errorf("unmarshal match board walls: %w", err)
	}

	return &matchboard.Board{
		MatchUUID: matchUUID,
		MapUUID:   mapUUID,
		Grid:      grid,
		Bg:        bg,
		Pieces:    pieces,
		Walls:     walls,
		UpdatedAt: updatedAt,
	}, nil
}

// Save upserts the board's line (spec §4.3, "Quando persiste"). Timestamp generated in
// Go, not SQL now() (gateway-conventions.instructions.md); b.UpdatedAt is set as a side
// effect, matching pgmap.UpdateMap's convention.
func (r *Repository) Save(ctx context.Context, b *matchboard.Board) error {
	grid, err := json.Marshal(b.Grid)
	if err != nil {
		return fmt.Errorf("marshal match board grid: %w", err)
	}

	var bg []byte
	if b.Bg != nil {
		bg, err = json.Marshal(b.Bg)
		if err != nil {
			return fmt.Errorf("marshal match board bg: %w", err)
		}
	}

	pieces, err := marshalJSONSlice(b.Pieces)
	if err != nil {
		return fmt.Errorf("marshal match board pieces: %w", err)
	}

	walls, err := marshalJSONSlice(b.Walls)
	if err != nil {
		return fmt.Errorf("marshal match board walls: %w", err)
	}

	b.UpdatedAt = time.Now().UTC()

	const q = `
		INSERT INTO match_boards (match_uuid, map_uuid, grid, bg, pieces, walls, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (match_uuid) DO UPDATE SET
			map_uuid = EXCLUDED.map_uuid, grid = EXCLUDED.grid, bg = EXCLUDED.bg,
			pieces = EXCLUDED.pieces, walls = EXCLUDED.walls, updated_at = EXCLUDED.updated_at
	`
	_, err = r.q.Exec(ctx, q, b.MatchUUID, b.MapUUID, grid, bg, pieces, walls, b.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save match board: %w", err)
	}
	return nil
}

// Delete removes the match's board line.
func (r *Repository) Delete(ctx context.Context, matchUUID uuid.UUID) error {
	const q = `DELETE FROM match_boards WHERE match_uuid = $1`
	if _, err := r.q.Exec(ctx, q, matchUUID); err != nil {
		return fmt.Errorf("delete match board: %w", err)
	}
	return nil
}

// marshalJSONSlice marshals a slice to JSON, defaulting nil to "[]" rather than the
// literal JSON null — matching pieces/walls' NOT NULL DEFAULT '[]' column and the same
// shape maps.pieces/maps.walls already use (see pgmap.mapper.go).
func marshalJSONSlice[T any](s []T) ([]byte, error) {
	if s == nil {
		s = []T{}
	}
	return json.Marshal(s)
}
