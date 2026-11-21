package match

import (
	"context"
	"errors"
	"log"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type RevealStatusEffectRequest struct {
	MatchUUID  uuid.UUID `path:"matchUuid" required:"true" doc:"Match UUID"`
	EffectUUID uuid.UUID `path:"effectUuid" required:"true" doc:"Status effect UUID (the id the history events carry)"`
}

type RevealStatusEffectResponse struct{}

func RevealStatusEffectHandler(
	uc matchUC.IRevealStatusEffect,
) func(context.Context, *RevealStatusEffectRequest) (*RevealStatusEffectResponse, error) {
	return func(ctx context.Context, req *RevealStatusEffectRequest) (*RevealStatusEffectResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		if err := uc.Reveal(ctx, req.MatchUUID, req.EffectUUID, userUUID); err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound):
				return nil, huma.Error404NotFound("Partida não encontrada.")
			case errors.Is(err, statuseffect.ErrNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden("Só o mestre da partida pode revelar.")
			default:
				log.Printf("reveal status effect: %v", err)
				return nil, huma.Error500InternalServerError("Não foi possível revelar agora.")
			}
		}
		return &RevealStatusEffectResponse{}, nil
	}
}
