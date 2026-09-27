package game

import appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"

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
}
