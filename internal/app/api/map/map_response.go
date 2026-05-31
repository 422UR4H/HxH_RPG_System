// internal/app/api/map/map_response.go
package mapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type GridShapeResponse struct {
	Kind      string  `json:"kind"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cell_size"`
	SkewRatio float64 `json:"skew_ratio"`
	Rotation  float64 `json:"rotation"`
	Color     string  `json:"color"`
	Opacity   float64 `json:"opacity"`
	LineStyle string  `json:"line_style"`
}

type MapResponse struct {
	ID          uuid.UUID         `json:"id"`
	CampaignID  uuid.UUID         `json:"campaign_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Grid        GridShapeResponse `json:"grid"`
	Bg          any               `json:"bg"`
	Pieces      any               `json:"pieces"`
	Walls       any               `json:"walls"`
	Decorations any               `json:"decorations"`
	Items       any               `json:"items"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

func toMapResponse(m *entity.TacticalMap) MapResponse {
	pieces := m.Pieces
	if pieces == nil {
		pieces = []entity.Piece{}
	}
	walls := m.Walls
	if walls == nil {
		walls = []entity.Wall{}
	}
	decorations := m.Decorations
	if decorations == nil {
		decorations = []entity.Decoration{}
	}
	items := m.Items
	if items == nil {
		items = []entity.MapItem{}
	}

	return MapResponse{
		ID:          m.ID,
		CampaignID:  m.CampaignID,
		Name:        m.Name,
		Description: m.Description,
		Grid: GridShapeResponse{
			Kind:      string(m.Grid.Kind),
			Cols:      m.Grid.Cols,
			Rows:      m.Grid.Rows,
			CellSize:  m.Grid.CellSize,
			SkewRatio: m.Grid.SkewRatio,
			Rotation:  m.Grid.Rotation,
			Color:     m.Grid.Color,
			Opacity:   m.Grid.Opacity,
			LineStyle: string(m.Grid.LineStyle),
		},
		Bg:          m.Bg,
		Pieces:      pieces,
		Walls:       walls,
		Decorations: decorations,
		Items:       items,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
