package mapuc_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type fakeMapRepo struct {
	m       *entity.TacticalMap
	updated *entity.TacticalMap
}

func (f *fakeMapRepo) CreateMap(context.Context, *entity.TacticalMap) error { return nil }
func (f *fakeMapRepo) GetMap(context.Context, uuid.UUID) (*entity.TacticalMap, error) {
	return f.m, nil
}
func (f *fakeMapRepo) ListMapsByCampaign(context.Context, uuid.UUID) ([]*entity.TacticalMap, error) {
	return nil, nil
}
func (f *fakeMapRepo) UpdateMap(_ context.Context, m *entity.TacticalMap) error {
	f.updated = m
	return nil
}
func (f *fakeMapRepo) DeleteMap(context.Context, uuid.UUID) error { return nil }

// fakeCampaignRepo embeds the interface: only GetCampaignMasterUUID is exercised here.
type fakeCampaignRepo struct {
	campaignApp.IRepository
	master uuid.UUID
}

func (f *fakeCampaignRepo) GetCampaignMasterUUID(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.master, nil
}

func TestUpdateMapUC_Bg(t *testing.T) {
	master := uuid.New()
	oldBg := &entity.BgImage{URL: "old"}
	newBg := &entity.BgImage{URL: "new"}
	tests := []struct {
		name  string
		input mapuc.UpdateMapInput
		want  *entity.BgImage
	}{
		{"neither keeps", mapuc.UpdateMapInput{}, oldBg},
		{"ClearBg removes", mapuc.UpdateMapInput{ClearBg: true}, nil},
		{"Bg replaces", mapuc.UpdateMapInput{Bg: newBg}, newBg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMapRepo{m: &entity.TacticalMap{
				Name: "mapa",
				Grid: entity.GridShape{Kind: entity.GridKindSquare, Cols: 5, Rows: 5, CellSize: 32},
				Bg:   oldBg,
			}}
			uc := mapuc.NewUpdateMapUC(repo, &fakeCampaignRepo{master: master})
			in := tt.input
			in.RequesterID = master
			if err := uc.UpdateMap(context.Background(), &in); err != nil {
				t.Fatalf("UpdateMap: %v", err)
			}
			if repo.updated == nil {
				t.Fatal("repo.UpdateMap not called")
			}
			if repo.updated.Bg != tt.want {
				t.Errorf("Bg = %+v, want %+v", repo.updated.Bg, tt.want)
			}
		})
	}
}
