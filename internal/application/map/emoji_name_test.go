package mapuc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

// writeSpyRepo conta as gravações, para provar que um nome inválido nunca chega ao repositório.
type writeSpyRepo struct {
	fakeMapRepo
	created int
}

func (r *writeSpyRepo) CreateMap(context.Context, *entity.TacticalMap) error {
	r.created++
	return nil
}

func TestCreateMapUC_NameEmoji(t *testing.T) {
	master := uuid.New()
	run := func(name string) (*writeSpyRepo, error) {
		repo := &writeSpyRepo{}
		uc := mapuc.NewCreateMapUC(repo, &fakeCampaignRepo{master: master})
		_, err := uc.CreateMap(context.Background(), &mapuc.CreateMapInput{
			RequesterID: master, CampaignID: uuid.New(), Name: name,
		})
		return repo, err
	}
	for _, name := range []string{"😀Floresta", "Flo😀resta", "Floresta😀"} {
		repo, err := run(name)
		if !errors.Is(err, service.ErrNameHasEmoji) {
			t.Errorf("name %q: got %v, want ErrNameHasEmoji", name, err)
		}
		if repo.created != 0 {
			t.Errorf("name %q: repo.CreateMap must not be called", name)
		}
	}
	repo, err := run("José Ção")
	if err != nil {
		t.Fatalf("accented name must pass: %v", err)
	}
	if repo.created != 1 {
		t.Errorf("repo.CreateMap calls = %d, want 1", repo.created)
	}
}

func TestUpdateMapUC_NameEmoji(t *testing.T) {
	master := uuid.New()
	run := func(name string) (*fakeMapRepo, error) {
		repo := &fakeMapRepo{m: &entity.TacticalMap{
			Name: "mapa",
			Grid: entity.GridShape{Kind: entity.GridKindSquare, Cols: 5, Rows: 5, CellSize: 32},
		}}
		uc := mapuc.NewUpdateMapUC(repo, &fakeCampaignRepo{master: master})
		err := uc.UpdateMap(context.Background(), &mapuc.UpdateMapInput{RequesterID: master, Name: &name})
		return repo, err
	}
	for _, name := range []string{"😀Floresta", "Flo😀resta", "Floresta😀"} {
		repo, err := run(name)
		if !errors.Is(err, service.ErrNameHasEmoji) {
			t.Errorf("name %q: got %v, want ErrNameHasEmoji", name, err)
		}
		if repo.updated != nil {
			t.Errorf("name %q: repo.UpdateMap must not be called", name)
		}
	}
	repo, err := run("José Ção")
	if err != nil {
		t.Fatalf("accented name must pass: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "José Ção" {
		t.Errorf("repo.UpdateMap not called with the new name: %+v", repo.updated)
	}
}
