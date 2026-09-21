package match

import (
	"context"
	"errors"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// IMatchReader reads the match being rostered. Narrow on purpose: widening IRepository
// would drag every mock that implements it into this slice.
type IMatchReader interface {
	GetMatch(ctx context.Context, uuid uuid.UUID) (*matchEntity.Match, error)
}

// ISheetOwnershipReader answers who owns a sheet — the only thing the roster needs to know
// about it.
type ISheetOwnershipReader interface {
	GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
}

// INPCRoster writes the NPC side of match_participants.
type INPCRoster interface {
	AddNPCParticipant(
		ctx context.Context, matchUUID, sheetUUID uuid.UUID, joinedAt time.Time,
	) (*matchEntity.Participant, error)
	RemoveNPCParticipant(ctx context.Context, matchUUID, sheetUUID uuid.UUID) error
}

type IAddMatchNPC interface {
	Add(ctx context.Context, input *AddMatchNPCInput) (*matchEntity.Participant, error)
}

type AddMatchNPCInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	SheetUUID     uuid.UUID
}

type AddMatchNPCUC struct {
	matchReader IMatchReader
	sheets      ISheetOwnershipReader
	roster      INPCRoster
}

func NewAddMatchNPCUC(
	matchReader IMatchReader, sheets ISheetOwnershipReader, roster INPCRoster,
) *AddMatchNPCUC {
	return &AddMatchNPCUC{
		matchReader: matchReader,
		sheets:      sheets,
		roster:      roster,
	}
}

// Add puts a master-controlled NPC's character sheet onto a match's roster.
//
// The requester's authorization is checked before any sheet is read, so the endpoint
// never leaks whether someone else's sheet exists (D1's backdoor concern applies here too).
func (uc *AddMatchNPCUC) Add(
	ctx context.Context, input *AddMatchNPCInput,
) (*matchEntity.Participant, error) {
	mt, err := uc.matchReader.GetMatch(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if mt.MasterUUID != input.RequesterUUID {
		return nil, ErrNotMatchMaster
	}
	if mt.StoryEndAt != nil {
		return nil, ErrMatchAlreadyFinished
	}

	rel, err := uc.sheets.GetCharacterSheetRelationshipUUIDs(ctx, input.SheetUUID)
	if err != nil {
		if errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			return nil, ErrCharacterSheetNotFound
		}
		return nil, err
	}

	if rel.PlayerUUID != nil {
		return nil, ErrSheetNotNPC
	}

	isOwnedByMaster := rel.MasterUUID != nil && *rel.MasterUUID == input.RequesterUUID
	isInMatchCampaign := rel.CampaignUUID != nil && *rel.CampaignUUID == mt.CampaignUUID
	if !isOwnedByMaster && !isInMatchCampaign {
		return nil, ErrSheetNotOwnedByMaster
	}

	participant, err := uc.roster.AddNPCParticipant(ctx, input.MatchUUID, input.SheetUUID, time.Now())
	if err != nil {
		if errors.Is(err, matchPg.ErrNPCAlreadyInMatch) {
			return nil, ErrNPCAlreadyInMatch
		}
		if errors.Is(err, matchPg.ErrSheetNotEligibleNPC) {
			return nil, ErrSheetNotNPC
		}
		return nil, err
	}
	return participant, nil
}
