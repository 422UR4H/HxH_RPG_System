// internal/gateway/pg/map/mapper.go
package pgmap

import (
	"encoding/json"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type pgModel struct {
	ID          uuid.UUID  // scanned from column: uuid
	CampaignID  uuid.UUID  // scanned from column: campaign_uuid
	Name        string
	Description string
	Grid        []byte
	Bg          []byte
	Pieces      []byte
	Walls       []byte
	Decorations []byte
	Items       []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toEntity(m *pgModel) (*entity.TacticalMap, error) {
	var grid entity.GridShape
	if err := json.Unmarshal(m.Grid, &grid); err != nil {
		return nil, fmt.Errorf("unmarshal grid: %w", err)
	}

	var bg *entity.BgImage
	if m.Bg != nil && string(m.Bg) != "null" {
		bg = &entity.BgImage{}
		if err := json.Unmarshal(m.Bg, bg); err != nil {
			return nil, fmt.Errorf("unmarshal bg: %w", err)
		}
	}

	pieces := []entity.Piece{}
	if err := json.Unmarshal(m.Pieces, &pieces); err != nil {
		return nil, fmt.Errorf("unmarshal pieces: %w", err)
	}

	walls := []entity.WallSegment{}
	if err := json.Unmarshal(m.Walls, &walls); err != nil {
		return nil, fmt.Errorf("unmarshal walls: %w", err)
	}

	decorations := []entity.Decoration{}
	if err := json.Unmarshal(m.Decorations, &decorations); err != nil {
		return nil, fmt.Errorf("unmarshal decorations: %w", err)
	}

	items := []entity.MapItem{}
	if err := json.Unmarshal(m.Items, &items); err != nil {
		return nil, fmt.Errorf("unmarshal items: %w", err)
	}

	return &entity.TacticalMap{
		ID:          m.ID,
		CampaignID:  m.CampaignID,
		Name:        m.Name,
		Description: m.Description,
		Grid:        grid,
		Bg:          bg,
		Pieces:      pieces,
		Walls:       walls,
		Decorations: decorations,
		Items:       items,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}, nil
}

func marshalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return b, nil
}
