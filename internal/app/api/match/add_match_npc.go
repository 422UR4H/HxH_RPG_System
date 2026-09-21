package match

import (
	"context"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AddMatchNPCRequestBody struct {
	CharacterSheetUUID uuid.UUID `json:"characterSheetUuid" required:"true" doc:"UUID of the NPC character sheet to put in the match"`
}

type AddMatchNPCRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
	Body AddMatchNPCRequestBody
}

type MatchNPCResponse struct {
	UUID               uuid.UUID `json:"uuid"`
	MatchUUID          uuid.UUID `json:"matchUuid"`
	CharacterSheetUUID uuid.UUID `json:"characterSheetUuid"`
	JoinedAt           string    `json:"joinedAt"`
}

type AddMatchNPCResponseBody struct {
	Participant MatchNPCResponse `json:"participant"`
}

type AddMatchNPCResponse struct {
	Body AddMatchNPCResponseBody
}

func AddMatchNPCHandler(
	uc matchUC.IAddMatchNPC,
) func(context.Context, *AddMatchNPCRequest) (*AddMatchNPCResponse, error) {
	return func(ctx context.Context, req *AddMatchNPCRequest) (*AddMatchNPCResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		participant, err := uc.Add(ctx, &matchUC.AddMatchNPCInput{
			RequesterUUID: userUUID,
			MatchUUID:     req.UUID,
			SheetUUID:     req.Body.CharacterSheetUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound),
				errors.Is(err, matchUC.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchUC.ErrSheetNotNPC),
				errors.Is(err, matchUC.ErrSheetNotOwnedByMaster),
				errors.Is(err, matchUC.ErrMatchAlreadyFinished),
				errors.Is(err, matchUC.ErrNPCAlreadyInMatch):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		return &AddMatchNPCResponse{
			Body: AddMatchNPCResponseBody{
				Participant: toMatchNPCResponse(participant),
			},
		}, nil
	}
}

func toMatchNPCResponse(p *matchEntity.Participant) MatchNPCResponse {
	return MatchNPCResponse{
		UUID:               p.UUID,
		MatchUUID:          p.MatchUUID,
		CharacterSheetUUID: p.Sheet.UUID,
		JoinedAt:           p.JoinedAt.Format(time.RFC3339),
	}
}
