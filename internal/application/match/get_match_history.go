package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// GetMatchHistoryResult is the match's closed turns, ALREADY projected per viewer — Scene ->
// Round -> Turn -> Action, the same tree FindMatchHistory reads, with service.ProjectAction /
// service.ProjectResolution run over every action, reaction and resolution in it. The REST
// handler serializes this straight to the wire; it does not filter anything itself.
type GetMatchHistoryResult struct {
	Scenes []HistoryScene
}

type IGetMatchHistory interface {
	Get(ctx context.Context, matchUUID, userUUID uuid.UUID) (*GetMatchHistoryResult, error)
}

// GetMatchHistoryUC is the read side of the Action History — a game surface with per-field
// visibility, not a log. Authorization mirrors GetMatchParticipantsUC exactly (public match, or
// master, or a campaign participant); the only thing this use case adds on top is running the
// tree through the SAME projection functions the WebSocket path runs, so a field can never end
// up public on one surface and hidden on the other.
type GetMatchHistoryUC struct {
	matchRepo            IRepository
	roundRepo            IRoundRepository
	participationChecker CampaignParticipationChecker
}

func NewGetMatchHistoryUC(
	matchRepo IRepository,
	roundRepo IRoundRepository,
	participationChecker CampaignParticipationChecker,
) *GetMatchHistoryUC {
	return &GetMatchHistoryUC{
		matchRepo:            matchRepo,
		roundRepo:            roundRepo,
		participationChecker: participationChecker,
	}
}

func (uc *GetMatchHistoryUC) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*GetMatchHistoryResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	viewerIsMaster := match.MasterUUID == userUUID
	if !match.IsPublic && !viewerIsMaster {
		ok, err := uc.participationChecker.ExistsSheetInCampaign(
			ctx, userUUID, match.CampaignUUID,
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, auth.ErrInsufficientPermissions
		}
	}

	// Owns is built from the match's PARTICIPANTS, not a live MatchSession: the history is
	// read for a match that may have no session running right now, so there is no
	// MatchSession.GetCharToPlayer() to borrow the way room.go's publishResolution does. A
	// participant's Sheet.PlayerUUID is the caller's own sheets — nil for an NPC, which by
	// construction nobody but the master can own.
	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}
	owns := make(map[uuid.UUID]bool, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil && *p.Sheet.PlayerUUID == userUUID {
			owns[p.Sheet.UUID] = true
		}
	}
	viewer := service.Viewer{IsMaster: viewerIsMaster, Owns: owns}

	scenes, err := uc.roundRepo.FindMatchHistory(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	// Never nil, even when scenes itself already is not (FindMatchHistory's own doc): an
	// explicit make keeps this true regardless of what the gateway returns, so an empty
	// history marshals as [] on the wire, not null.
	projected := make([]HistoryScene, len(scenes))
	for i, sc := range scenes {
		projected[i] = sc
		projected[i].Rounds = make([]HistoryRound, len(sc.Rounds))
		for j, ro := range sc.Rounds {
			projected[i].Rounds[j] = ro
			projected[i].Rounds[j].Turns = make([]HistoryTurn, len(ro.Turns))
			for k, tu := range ro.Turns {
				pt := tu
				pt.Action = service.ProjectAction(tu.Action, viewer)
				pt.Reactions = make([]action.Action, len(tu.Reactions))
				for l, react := range tu.Reactions {
					pt.Reactions[l] = service.ProjectAction(react, viewer)
				}
				pt.Resolution = service.ProjectResolution(tu.Resolution, viewer)
				projected[i].Rounds[j].Turns[k] = pt
			}
		}
	}

	return &GetMatchHistoryResult{Scenes: projected}, nil
}
