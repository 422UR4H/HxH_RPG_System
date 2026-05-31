// internal/app/api/map/create_map_test.go
package mapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func newTestMap(campaignID, userID uuid.UUID, name string) *entity.TacticalMap {
	now := time.Now().UTC()
	return &entity.TacticalMap{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Name:        name,
		Description: "test description",
		Grid:        entity.DefaultGrid(),
		Pieces:      []entity.Piece{},
		Walls:       []entity.Wall{},
		Decorations: []entity.Decoration{},
		Items:       []entity.MapItem{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestCreateMapHandler_Success(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()
	mapName := "Test Map"

	_, api := humatest.New(t)

	mock := &mockCreateMap{
		result: newTestMap(campaignID, userID, mapName),
		err:    nil,
	}
	handler := mapapi.CreateMapHandler(mock)

	huma.Register(api, huma.Operation{
		Method:        http.MethodPost,
		Path:          "/campaigns/{campaign_id}/maps",
		DefaultStatus: http.StatusCreated,
	}, handler)

	body := map[string]any{
		"name":        mapName,
		"description": "test description",
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/campaigns/"+campaignID.String()+"/maps", body)

	if resp.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if result["name"] != mapName {
		t.Errorf("got name %v, want %q", result["name"], mapName)
	}
	if result["campaign_id"] != campaignID.String() {
		t.Errorf("got campaign_id %v, want %v", result["campaign_id"], campaignID.String())
	}
}

func TestCreateMapHandler_NotMaster_Returns403(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()

	_, api := humatest.New(t)

	mock := &mockCreateMap{
		result: nil,
		err:    mapuc.ErrNotMapMaster,
	}
	handler := mapapi.CreateMapHandler(mock)

	huma.Register(api, huma.Operation{
		Method:        http.MethodPost,
		Path:          "/campaigns/{campaign_id}/maps",
		DefaultStatus: http.StatusCreated,
	}, handler)

	body := map[string]any{
		"name":        "Forbidden Map",
		"description": "",
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/campaigns/"+campaignID.String()+"/maps", body)

	if resp.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
}
