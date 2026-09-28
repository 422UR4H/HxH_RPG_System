package game

import (
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
)

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
}
