package match

import (
	"context"
	"errors"

	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IRemoveMatchNPC interface {
	Remove(ctx context.Context, input *RemoveMatchNPCInput) error
}

type RemoveMatchNPCInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	SheetUUID     uuid.UUID
}

type RemoveMatchNPCUC struct {
	matchReader IMatchReader
	roster      INPCRoster
}

func NewRemoveMatchNPCUC(matchReader IMatchReader, roster INPCRoster) *RemoveMatchNPCUC {
	return &RemoveMatchNPCUC{
		matchReader: matchReader,
		roster:      roster,
	}
}

// Remove takes an NPC's character sheet off a match's roster.
//
// Unlike Add, there is no StoryEndAt guard: taking an NPC out of a finished match is
// harmless, and refusing it would only get in the way of cleanup. There is also no sheet
// re-read — the NPC eligibility guard already lives in the SQL (Task 1).
func (uc *RemoveMatchNPCUC) Remove(ctx context.Context, input *RemoveMatchNPCInput) error {
	mt, err := uc.matchReader.GetMatch(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return ErrMatchNotFound
		}
		return err
	}

	if mt.MasterUUID != input.RequesterUUID {
		return ErrNotMatchMaster
	}

	err = uc.roster.RemoveNPCParticipant(ctx, input.MatchUUID, input.SheetUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrNPCNotInMatch) {
			return ErrNPCNotInMatch
		}
		return err
	}
	return nil
}
