package match

import (
	"context"
	"errors"
	"log"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

// IBoardReader reads the match board a session is born on top of (spec §4.3, B11): Init
// scans its pieces for an NPC that has not been rostered yet. Satisfied by
// matchboarduc.LoadMatchBoardUC. nil in NewInitMatchSessionUC restores today's behaviour
// untouched — every test written before B11 leaves it unset.
type IBoardReader interface {
	// Load returns the match's board, or (nil, nil) when the match has no map attached.
	Load(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error)
}

// INPCEnroller enrolls a map piece's character onto the match roster as an NPC — the same
// AddMatchNPCUC.Add the REST POST /npcs and the WS add_npc verb already call. Narrowed
// locally, rather than reusing IAddMatchNPC, because Init only ever calls Add.
type INPCEnroller interface {
	Add(ctx context.Context, input *AddMatchNPCInput) (*matchEntity.Participant, error)
}

type InitMatchSessionUC struct {
	matchRepo   IRepository
	sheetLoader ICharSheetLoader
	roundRepo   IRoundRepository
	board       IBoardReader
	npcs        INPCEnroller
}

// NewInitMatchSessionUC wires the session loader. board and npcs are optional — nil for
// either (or both) keeps the pre-B11 behaviour: no board is read and no piece is enrolled.
func NewInitMatchSessionUC(
	matchRepo IRepository, sheetLoader ICharSheetLoader, roundRepo IRoundRepository,
	board IBoardReader, npcs INPCEnroller,
) *InitMatchSessionUC {
	return &InitMatchSessionUC{
		matchRepo: matchRepo, sheetLoader: sheetLoader, roundRepo: roundRepo, board: board, npcs: npcs,
	}
}

func (uc *InitMatchSessionUC) Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error) {
	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	if uc.board != nil && uc.npcs != nil {
		// A board error (reading the master, loading the board) is logged and swallowed: the
		// match's own session must be born regardless — B11 is a convenience enrollment, not
		// a precondition of the match existing (spec §4.3, "B11").
		if enrolled := uc.enrollBoardNPCs(ctx, matchUUID, participants); enrolled != nil {
			participants = enrolled
		}
	}

	charSheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(participants))
	for _, p := range participants {
		// No PlayerUUID guard: an NPC has PlayerUUID nil and MasterUUID set, and the
		// master plays it. The sheet loader is keyed by sheet UUID either way.
		//
		// ⚠️ The loader's second return is wasCorrected — whether hydrating the sheet had to
		// repair it — NOT whether it was found. This used to read `if found { ... }`, which
		// silently dropped every intact sheet and kept only the repaired ones, leaving the
		// session with an empty charSheets map. Nothing read that map until the character
		// collision arrived, which is why it went unnoticed. A missing sheet comes back as
		// an error, not as a false.
		sheet, _, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID.String())
		if err != nil {
			if errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
				// One roster entry without a sheet must not take the whole match down.
				log.Printf("init match session %s: no sheet for participant %s, skipping", matchUUID, p.Sheet.UUID)
				continue
			}
			return nil, err
		}
		charSheets[p.Sheet.UUID] = sheet
	}

	data, err := uc.roundRepo.FindActiveSession(ctx, matchUUID)
	if err != nil {
		return nil, err
	}
	if data != nil {
		sc := sceneentity.ReconstructScene(data.SceneID, enum.SceneCategory(data.Category), data.BriefInitDesc, data.SceneCreatedAt)
		r := roundentity.ReconstructRound(data.RoundID, enum.RoundMode(data.Mode), data.RoundCreatedAt)
		return matchsession.NewMatchSessionWithState(matchUUID, charSheets, participants, sc, r), nil
	}
	return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
}

// enrollBoardNPCs is B11 (spec §4.3): every piece on the match's board whose character is
// not yet a participant is enrolled as an NPC through the same AddMatchNPCUC.Add the REST
// POST /npcs and the WS add_npc verb already use. It runs before the session is built — on
// start_match (T3 already persists the lobby's board first, so this reads what is actually
// on screen) and on rehydration alike — and fixes up matches that already exist without
// touching the database by hand.
//
// Returns the refreshed participant list (with any newly enrolled NPC in it) once at least
// one board existed to scan, or nil when there was nothing to refresh (no board attached, or
// every board/master read failed) — the caller keeps its own list in that case.
//
// Never returns an error: every failure along this path is logged and swallowed, because the
// match's own session must be born whether or not its board could be scanned.
func (uc *InitMatchSessionUC) enrollBoardNPCs(
	ctx context.Context, matchUUID uuid.UUID, participants []*matchEntity.Participant,
) []*matchEntity.Participant {
	mt, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		log.Printf("init match session %s: enroll board NPCs: get match: %v", matchUUID, err)
		return nil
	}

	board, err := uc.board.Load(ctx, matchUUID)
	if err != nil {
		log.Printf("init match session %s: enroll board NPCs: load board: %v", matchUUID, err)
		return nil
	}
	if board == nil {
		// No map attached at all — nothing to enroll, exactly like before B11.
		return nil
	}

	alreadyIn := make(map[uuid.UUID]bool, len(participants))
	for _, p := range participants {
		alreadyIn[p.Sheet.UUID] = true
	}

	for _, piece := range board.Pieces {
		charUUID, err := uuid.Parse(piece.CharacterID)
		if err != nil {
			// Not every piece necessarily carries a character (or one that parses) —
			// nothing to enroll here.
			continue
		}
		if alreadyIn[charUUID] {
			continue
		}

		_, err = uc.npcs.Add(ctx, &AddMatchNPCInput{
			RequesterUUID: mt.MasterUUID,
			MatchUUID:     matchUUID,
			SheetUUID:     charUUID,
		})
		if err != nil && !errors.Is(err, ErrNPCAlreadyInMatch) {
			// ErrSheetNotNPC: a player's character with a piece that was never enrolled —
			// not this Init's job to fix (review focus: it must stay OUT of the session).
			// ErrCharacterSheetNotFound / ErrSheetNotOwnedByMaster: the piece points at a
			// sheet that no longer qualifies. None of these abort the match being born.
			log.Printf("init match session %s: enroll board NPC %s: %v", matchUUID, charUUID, err)
		}
		// ErrNPCAlreadyInMatch means every guard already passed before the INSERT that
		// reported the duplicate — the roster already has it, and the re-fetch below
		// picks it up.
	}

	refreshed, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		log.Printf("init match session %s: enroll board NPCs: re-list participants: %v", matchUUID, err)
		return nil
	}
	return refreshed
}

var _ IInitMatchSession = (*InitMatchSessionUC)(nil)
