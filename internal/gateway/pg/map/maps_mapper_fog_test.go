// internal/gateway/pg/map/maps_mapper_fog_test.go
package pgmap

import (
	"encoding/json"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/google/uuid"
)

// mustMarshal is a test helper that panics on json.Marshal failure.
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// newTestPgModel returns a minimal pgModel with all required JSONB columns
// populated so toEntity can unmarshal them without error.
func newTestPgModel(fogMode string) *pgModel {
	return &pgModel{
		ID:          uuid.New(),
		CampaignID:  uuid.New(),
		Name:        "Test Map",
		Description: "desc",
		FogMode:     fogMode,
		Grid:        mustMarshal(entity.DefaultGrid()),
		Bg:          nil,
		Pieces:      mustMarshal([]entity.Piece{}),
		Walls:       mustMarshal([]entity.WallSegment{}),
		Decorations: mustMarshal([]entity.Decoration{}),
		Items:       mustMarshal([]entity.MapItem{}),
	}
}

func TestMapper_FogModeRoundTrip_Explored(t *testing.T) {
	m, err := toEntity(newTestPgModel(string(fog.FogModeExplored)))
	if err != nil {
		t.Fatalf("toEntity: %v", err)
	}
	if m.FogMode != fog.FogModeExplored {
		t.Fatalf("want FogModeExplored, got %q", m.FogMode)
	}
}

func TestMapper_FogModeRoundTrip_Live(t *testing.T) {
	m, err := toEntity(newTestPgModel(string(fog.FogModeLive)))
	if err != nil {
		t.Fatalf("toEntity: %v", err)
	}
	if m.FogMode != fog.FogModeLive {
		t.Fatalf("want FogModeLive, got %q", m.FogMode)
	}
}

func TestMapper_FogModeRoundTrip_EmptyDefaultsToExplored(t *testing.T) {
	m, err := toEntity(newTestPgModel(""))
	if err != nil {
		t.Fatalf("toEntity: %v", err)
	}
	if m.FogMode != fog.FogModeExplored {
		t.Fatalf("empty fog_mode: want FogModeExplored default, got %q", m.FogMode)
	}
}

func TestMapper_FogModeRoundTrip_InvalidDefaultsToExplored(t *testing.T) {
	m, err := toEntity(newTestPgModel("bogus"))
	if err != nil {
		t.Fatalf("toEntity: %v", err)
	}
	if m.FogMode != fog.FogModeExplored {
		t.Fatalf("invalid fog_mode: want FogModeExplored default, got %q", m.FogMode)
	}
}
