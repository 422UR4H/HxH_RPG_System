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
