package match

import (
	"context"
	"errors"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

// IAddLiveNPC puts an NPC's character sheet into a LIVE match: it does the same roster
// I/O and rules as AddMatchNPCUC, then hands back the sheet the caller needs to insert
// into the in-memory session. It never touches the session itself — see Execute's doc.
type IAddLiveNPC interface {
	Execute(ctx context.Context, input *AddMatchNPCInput) (*csSheet.CharacterSheet, error)
}

type AddLiveNPCUC struct {
	roster      IAddMatchNPC     // the real AddMatchNPCUC in production
	sheetLoader ICharSheetLoader // the same loader InitMatchSessionUC uses
}

func NewAddLiveNPCUC(roster IAddMatchNPC, sheetLoader ICharSheetLoader) *AddLiveNPCUC {
	return &AddLiveNPCUC{roster: roster, sheetLoader: sheetLoader}
}

// Execute rosters the NPC (DB) and loads its sheet (DB), but never mutates
// MatchSession: that I/O runs outside the room's lock, and the game-server caller
// inserts the returned sheet into the session under r.mu afterwards (Decision:
// I/O outside the lock).
func (uc *AddLiveNPCUC) Execute(
	ctx context.Context, input *AddMatchNPCInput,
) (*csSheet.CharacterSheet, error) {
	_, err := uc.roster.Add(ctx, input)
	if err != nil {
		// ErrNPCAlreadyInMatch means every guard in AddMatchNPCUC.Add already ran and
		// passed before the INSERT that raced us — the NPC is legitimately on the
		// roster (D2), so we still load and hand back its sheet instead of failing
		// a request that's really just late to a roster it belongs on.
		if !errors.Is(err, ErrNPCAlreadyInMatch) {
			return nil, err
		}
	}

	// ⚠️ Second return is wasCorrected, NOT found — see init_match_session.go's warning.
	// A missing sheet comes back as an error, not as a false.
	sheet, _, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, input.SheetUUID.String())
	if err != nil {
		if errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			return nil, ErrCharacterSheetNotFound
		}
		return nil, err
	}
	return sheet, nil
}

var _ IAddLiveNPC = (*AddLiveNPCUC)(nil)
