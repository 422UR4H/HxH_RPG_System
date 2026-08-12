package entity

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/google/uuid"
)

type TacticalMap struct {
	ID          uuid.UUID    `json:"id"`
	CampaignID  uuid.UUID    `json:"campaign_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Grid        GridShape    `json:"grid"`
	Bg          *BgImage     `json:"bg"`
	Pieces      []Piece      `json:"pieces"`
	Walls       []WallSegment `json:"walls"`
	Decorations []Decoration `json:"decorations"`
	Items       []MapItem    `json:"items"`
	// Persisted in maps.fog_mode and honoured by FilterMapState, but not yet exposed in
	// any REST request/response: fog_mode is slated to become a per-match setting chosen
	// by the master, and the campaign/match settings mechanism does not exist yet. The
	// game server hardcodes explored until then (see room.go). Do not remove — dropping
	// FogMode would remove the `live` fog mode from the product.
	// See: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
	FogMode     fog.FogMode  `json:"fog_mode"` // default FogModeExplored when zero
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func NewTacticalMap(campaignID uuid.UUID, name, description string) *TacticalMap {
	now := time.Now().UTC()
	return &TacticalMap{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Name:        name,
		Description: description,
		Grid:        DefaultGrid(),
		Bg:          nil,
		Pieces:      []Piece{},
		Walls:       []WallSegment{},
		Decorations: []Decoration{},
		Items:       []MapItem{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
