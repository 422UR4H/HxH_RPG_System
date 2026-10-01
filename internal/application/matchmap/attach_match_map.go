package matchmapuc

import (
	"context"
	"errors"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	"github.com/google/uuid"
)

type IAttachMatchMap interface {
	Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error)
}

type AttachMatchMapInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	MapUUID       uuid.UUID
	// InheritBoardFromMatchUUID is B16 (spec §4.3): optional. When set, the match starts
	// with the board (pieces, wall state, fog) that the named match ended with, instead of
	// a fresh snapshot of MapUUID. The source match must be in the same campaign, its own
	// board must already be on MapUUID, and it must actually have a board to copy.
	InheritBoardFromMatchUUID *uuid.UUID
}

type AttachMatchMapUC struct {
	repo      IRepository
	matchRepo IMatchRepository
	boardRepo IMatchBoardRepository
}

func NewAttachMatchMapUC(repo IRepository, matchRepo IMatchRepository, boardRepo IMatchBoardRepository) *AttachMatchMapUC {
	return &AttachMatchMapUC{repo: repo, matchRepo: matchRepo, boardRepo: boardRepo}
}

func (uc *AttachMatchMapUC) Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error) {
	info, err := uc.matchRepo.GetMatchInfo(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	if info.MasterUUID != input.RequesterUUID {
		return nil, ErrNotMatchMaster
	}
	if info.GameStartAt != nil {
		return nil, ErrMatchAlreadyStarted
	}

	// The match's CURRENT map (before this attach), read now because AttachMap below
	// overwrites it. nil when the match never had a map attached — nothing to compare, and
	// nothing to delete if inheritance is not requested.
	var prevMapUUID *uuid.UUID
	prevMM, err := uc.repo.GetMatchMap(ctx, input.MatchUUID)
	if err != nil {
		if !errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return nil, fmt.Errorf("attach match map: get current match map: %w", err)
		}
	} else {
		parsed, err := uuid.Parse(prevMM.MapUUID)
		if err != nil {
			return nil, fmt.Errorf("attach match map: parse current map uuid: %w", err)
		}
		prevMapUUID = &parsed
	}

	if input.InheritBoardFromMatchUUID != nil {
		// Checked before any repo call: inheriting from itself would have Copy(dst, dst)
		// delete the very row its own INSERT…SELECT reads from, failing with a plain error
		// (500) instead of a clean, user-facing one.
		if *input.InheritBoardFromMatchUUID == input.MatchUUID {
			return nil, ErrSourceMatchIsTheSameMatch
		}

		srcInfo, err := uc.matchRepo.GetMatchInfo(ctx, *input.InheritBoardFromMatchUUID)
		if err != nil {
			if errors.Is(err, ErrMatchNotFound) {
				return nil, ErrMatchNotFound
			}
			return nil, err
		}
		if srcInfo.CampaignUUID != info.CampaignUUID {
			return nil, ErrSourceMatchNotInCampaign
		}

		srcBoard, err := uc.boardRepo.Get(ctx, *input.InheritBoardFromMatchUUID)
		if err != nil {
			return nil, fmt.Errorf("attach match map: get source board: %w", err)
		}
		if srcBoard == nil {
			return nil, ErrSourceMatchHasNoBoard
		}
		if srcBoard.MapUUID != input.MapUUID {
			return nil, ErrSourceMatchOnAnotherMap
		}
	}

	mm, err := uc.repo.AttachMap(ctx, input.MatchUUID, input.MapUUID)
	if err != nil {
		return nil, err
	}

	if input.InheritBoardFromMatchUUID != nil {
		if err := uc.boardRepo.Copy(ctx, *input.InheritBoardFromMatchUUID, input.MatchUUID); err != nil {
			return nil, fmt.Errorf("attach match map: copy inherited board: %w", err)
		}
	} else if prevMapUUID != nil && *prevMapUUID != input.MapUUID {
		// Attaching a DIFFERENT map without inheritance: the old board was a portrait of
		// the old map, and stays meaningless on the new one (spec §4.3, "anexar outro mapa
		// apaga o tabuleiro velho"). Attaching the SAME map leaves the board untouched.
		if err := uc.boardRepo.Delete(ctx, input.MatchUUID); err != nil {
			return nil, fmt.Errorf("attach match map: delete old board: %w", err)
		}
	}

	return mm, nil
}
