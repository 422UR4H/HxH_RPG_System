package matchmapuc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	"github.com/google/uuid"
)

func TestAttachMatchMapUC_Attach(t *testing.T) {
	requesterUUID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()
	campaignUUID := uuid.New()
	srcMatchUUID := uuid.New()

	// baseRepo returns a match that never had a map attached (ErrMatchMapNotFound), and a
	// successful AttachMap — every test case tweaks what it needs from there.
	newBaseRepo := func() *mockRepository {
		return &mockRepository{
			getMatchMapFn: func(_ context.Context, _ uuid.UUID) (*entity.MatchMap, error) {
				return nil, pgmatchmap.ErrMatchMapNotFound
			},
			attachMapFn: func(_ context.Context, mUUID, mapUUID uuid.UUID) (*entity.MatchMap, error) {
				return &entity.MatchMap{
					MatchUUID:  mUUID.String(),
					MapUUID:    mapUUID.String(),
					AttachedAt: time.Now(),
				}, nil
			},
		}
	}

	dstInfo := &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: campaignUUID}

	t.Run("valid inheritance: same campaign, same map, source has a board -> Copy is called", func(t *testing.T) {
		repo := newBaseRepo()
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, mUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
				if mUUID == matchUUID {
					return dstInfo, nil
				}
				return &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{
			getFn: func(_ context.Context, mUUID uuid.UUID) (*matchboard.Board, error) {
				return &matchboard.Board{MatchUUID: mUUID, MapUUID: mapUUID}, nil
			},
		}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID:             requesterUUID,
			MatchUUID:                 matchUUID,
			MapUUID:                   mapUUID,
			InheritBoardFromMatchUUID: &srcMatchUUID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !boardRepo.copyCalled {
			t.Fatalf("expected Copy to be called")
		}
		if boardRepo.copySrc != srcMatchUUID || boardRepo.copyDst != matchUUID {
			t.Fatalf("Copy called with wrong uuids: src=%s dst=%s", boardRepo.copySrc, boardRepo.copyDst)
		}
		if boardRepo.deleteCalled {
			t.Fatalf("expected Delete NOT to be called on inheritance")
		}
	})

	t.Run("source match in a different campaign -> ErrSourceMatchNotInCampaign", func(t *testing.T) {
		repo := newBaseRepo()
		otherCampaignUUID := uuid.New()
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, mUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
				if mUUID == matchUUID {
					return dstInfo, nil
				}
				return &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: otherCampaignUUID}, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{
			getFn: func(_ context.Context, mUUID uuid.UUID) (*matchboard.Board, error) {
				t.Fatalf("Get should not be reached when campaigns differ")
				return nil, nil
			},
		}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID:             requesterUUID,
			MatchUUID:                 matchUUID,
			MapUUID:                   mapUUID,
			InheritBoardFromMatchUUID: &srcMatchUUID,
		})
		if !errors.Is(err, matchmapuc.ErrSourceMatchNotInCampaign) {
			t.Fatalf("expected ErrSourceMatchNotInCampaign, got %v", err)
		}
	})

	t.Run("source match's board is on a different map -> ErrSourceMatchOnAnotherMap", func(t *testing.T) {
		repo := newBaseRepo()
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, mUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
				if mUUID == matchUUID {
					return dstInfo, nil
				}
				return &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{
			getFn: func(_ context.Context, mUUID uuid.UUID) (*matchboard.Board, error) {
				return &matchboard.Board{MatchUUID: mUUID, MapUUID: uuid.New()}, nil // different map
			},
		}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID:             requesterUUID,
			MatchUUID:                 matchUUID,
			MapUUID:                   mapUUID,
			InheritBoardFromMatchUUID: &srcMatchUUID,
		})
		if !errors.Is(err, matchmapuc.ErrSourceMatchOnAnotherMap) {
			t.Fatalf("expected ErrSourceMatchOnAnotherMap, got %v", err)
		}
	})

	t.Run("source match has no board -> ErrSourceMatchHasNoBoard", func(t *testing.T) {
		repo := newBaseRepo()
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, mUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
				if mUUID == matchUUID {
					return dstInfo, nil
				}
				return &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: campaignUUID}, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{
			getFn: func(_ context.Context, _ uuid.UUID) (*matchboard.Board, error) {
				return nil, nil // no board row
			},
		}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID:             requesterUUID,
			MatchUUID:                 matchUUID,
			MapUUID:                   mapUUID,
			InheritBoardFromMatchUUID: &srcMatchUUID,
		})
		if !errors.Is(err, matchmapuc.ErrSourceMatchHasNoBoard) {
			t.Fatalf("expected ErrSourceMatchHasNoBoard, got %v", err)
		}
	})

	t.Run("match already started -> ErrMatchAlreadyStarted, even with inheritance requested", func(t *testing.T) {
		repo := newBaseRepo()
		startedAt := time.Now()
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, _ uuid.UUID) (*matchmapuc.MatchInfo, error) {
				return &matchmapuc.MatchInfo{MasterUUID: requesterUUID, CampaignUUID: campaignUUID, GameStartAt: &startedAt}, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{
			getFn: func(_ context.Context, _ uuid.UUID) (*matchboard.Board, error) {
				t.Fatalf("board repo should not be reached once the match already started")
				return nil, nil
			},
		}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID:             requesterUUID,
			MatchUUID:                 matchUUID,
			MapUUID:                   mapUUID,
			InheritBoardFromMatchUUID: &srcMatchUUID,
		})
		if !errors.Is(err, matchmapuc.ErrMatchAlreadyStarted) {
			t.Fatalf("expected ErrMatchAlreadyStarted, got %v", err)
		}
	})

	t.Run("attaching a DIFFERENT map without inheritance deletes the old board", func(t *testing.T) {
		oldMapUUID := uuid.New()
		repo := newBaseRepo()
		repo.getMatchMapFn = func(_ context.Context, mUUID uuid.UUID) (*entity.MatchMap, error) {
			return &entity.MatchMap{MatchUUID: mUUID.String(), MapUUID: oldMapUUID.String(), AttachedAt: time.Now()}, nil
		}
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, _ uuid.UUID) (*matchmapuc.MatchInfo, error) {
				return dstInfo, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID: requesterUUID,
			MatchUUID:     matchUUID,
			MapUUID:       mapUUID, // different from oldMapUUID
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !boardRepo.deleteCalled {
			t.Fatalf("expected Delete to be called when attaching a different map")
		}
		if boardRepo.deleteUUID != matchUUID {
			t.Fatalf("Delete called with wrong match uuid: %s", boardRepo.deleteUUID)
		}
		if boardRepo.copyCalled {
			t.Fatalf("expected Copy NOT to be called without inheritance")
		}
	})

	t.Run("attaching the SAME map without inheritance leaves the board alone", func(t *testing.T) {
		repo := newBaseRepo()
		repo.getMatchMapFn = func(_ context.Context, mUUID uuid.UUID) (*entity.MatchMap, error) {
			return &entity.MatchMap{MatchUUID: mUUID.String(), MapUUID: mapUUID.String(), AttachedAt: time.Now()}, nil
		}
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, _ uuid.UUID) (*matchmapuc.MatchInfo, error) {
				return dstInfo, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID: requesterUUID,
			MatchUUID:     matchUUID,
			MapUUID:       mapUUID, // same as current
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if boardRepo.deleteCalled {
			t.Fatalf("expected Delete NOT to be called when the map is unchanged")
		}
		if boardRepo.copyCalled {
			t.Fatalf("expected Copy NOT to be called without inheritance")
		}
	})

	t.Run("first-ever attach (no previous map) without inheritance does not touch the board repo", func(t *testing.T) {
		repo := newBaseRepo() // getMatchMapFn returns ErrMatchMapNotFound
		matchRepo := &mockMatchRepository{
			getMatchInfoFn: func(_ context.Context, _ uuid.UUID) (*matchmapuc.MatchInfo, error) {
				return dstInfo, nil
			},
		}
		boardRepo := &mockMatchBoardRepository{}
		uc := matchmapuc.NewAttachMatchMapUC(repo, matchRepo, boardRepo)

		_, err := uc.Attach(context.Background(), &matchmapuc.AttachMatchMapInput{
			RequesterUUID: requesterUUID,
			MatchUUID:     matchUUID,
			MapUUID:       mapUUID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if boardRepo.deleteCalled || boardRepo.copyCalled {
			t.Fatalf("expected neither Delete nor Copy on a first-ever attach")
		}
	})
}
