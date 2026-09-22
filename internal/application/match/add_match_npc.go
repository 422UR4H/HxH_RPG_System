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
	// The campaign arm reads looser than the master arm, but it cannot admit a stranger:
	// an NPC only gets a campaign_uuid through create_character_sheet, which forces its
	// master_uuid to be that campaign's master, and create_match forces the match's master
	// to be the same person (accept_submission, the only other writer, needs a submission,
	// which only a player-owned sheet can have). So both arms land on the requester — which
	// is the identity matchsession later trusts in charToPlayer.
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
