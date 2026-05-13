package match_test

import (
	"context"
	"errors"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestEnqueueActionUC(t *testing.T) {
	t.Run("enqueues action for enrolled player", func(t *testing.T) {
		playerUUID := uuid.New()
		matchUUID := uuid.New()
		p := &matchDomain.Participant{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &playerUUID},
		}
		session := matchsession.NewMatchSession(matchUUID, nil, []*matchDomain.Participant{p})
		a := action.NewAction(playerUUID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
		uc := match.NewEnqueueActionUC()
		if err := uc.Execute(context.Background(), session, playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for unknown player", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		unknownUUID := uuid.New()
		a := action.NewAction(unknownUUID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
		uc := match.NewEnqueueActionUC()
		err := uc.Execute(context.Background(), session, unknownUUID, a)
		if !errors.Is(err, matchsession.ErrParticipantNotFound) {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})
}
