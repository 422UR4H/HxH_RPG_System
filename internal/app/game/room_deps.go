package game

import (
	"context"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
)

// IMasterActionRepo writes one master action row (spec §4.8). recordMasterAction is the one
// caller; the one real implementation is pgmasteraction.Repository.
type IMasterActionRepo interface {
	Insert(ctx context.Context, r masteraction.Record) error
}

// RoomDeps is everything a Room needs from the outside, in one place.
//
// It exists because the list grew past what a positional signature can carry honestly: the
// handler, the hub and the room each took the same seventeen use cases in the same order, and
// every new dependency meant editing all three plus every test that builds a handler. A field
// left nil is a capability the room does not have — the arms that need it check.
type RoomDeps struct {
	StartMatchUC          IStartMatch
	KickPlayerUC          IKickPlayer
	InitSessionUC         IInitMatchSession
	OpenNextActionUC      IOpenNextAction
	PullActionUC          IPullAction
	EnqueueActionUC       IEnqueueAction
	AttachReactionUC      IAttachReaction
	OpenReactionUC        IOpenReaction
	CloseTurnUC           ICloseTurn
	ChangeSceneUC         IChangeScene
	RoundRepo             appmatch.IRoundRepository
	EnqueueMasterActionUC IEnqueueMasterAction
	ChangeRoundModeUC     appmatch.IChangeRoundMode
	EditActionUC          IEditAction
	AddLiveNPCUC          IAddLiveNPC
	// LoadBoardUC loads the match's board from the database (spec §4.3, "Quem carrega",
	// B14): the Room calls it when it is born and, while still a lobby, on every master
	// reconnect. nil means the room has no board capability — the same "field left nil is a
	// capability the room does not have" rule this struct's own doc comment states, and what
	// every test built before B14 still gets by leaving it unset.
	LoadBoardUC matchboarduc.ILoadMatchBoard
	// SaveBoardUC persists the board and every player's fog memory (spec §4.3, "Quando
	// persiste", B3). persistBoard is the one caller. nil means the room has no persistence
	// capability — every test built before T3 leaves it unset.
	SaveBoardUC matchboarduc.ISaveMatchBoard
	// MemoryLoader seeds per-player fog memory when a session is born (StartMatch and
	// RehydrateSession call it before the session goes live). nil means no persisted memory
	// is loaded — the session simply starts with none, same as before T3.
	MemoryLoader matchboarduc.IMemoryLoader
	// SheetOwnership answers who owns a sheet, for the lobby's server-side piece ownership
	// check (spec §4.3, "Quem move o quê", B14): the lobby has no charToPlayer, so
	// handlePieceMoved reads the sheet's own PlayerUUID instead. Unlike every other field in
	// this struct, nil here does NOT mean "no capability, skip the check" — a player's
	// handlePieceMoved treats it as "cannot verify ownership" and refuses, the same
	// fail-closed default GetCharacterSheetRelationshipUUIDs' own AddMatchNPCUC already applies
	// to sheet reads. The one real implementation is the same sheetRepository cmd/game/main.go
	// already wires into AddMatchNPCUC.
	SheetOwnership appmatch.ISheetOwnershipReader
	// MasterActionRepo records every accepted enqueue_master_action the instant it is applied,
	// with what each player saw of it live (spec §4.8). recordMasterAction is the one caller.
	// nil means the room records nothing — the usual "no capability" rule, and what every test
	// built before T5 still gets by leaving it unset.
	MasterActionRepo IMasterActionRepo
	// EventRepo records what happens inside a round that is not a turn and is nobody's action —
	// today the round's regime change, in the change_round_mode arm (spec §4.5, B15). The one
	// real implementation is pgmatchevent.Repository. nil means the room records no event — the
	// usual "no capability" rule.
	EventRepo interface {
		Insert(ctx context.Context, e matchevent.Event) error
	}
}
