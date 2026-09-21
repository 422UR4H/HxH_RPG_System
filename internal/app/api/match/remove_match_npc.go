package match

import (
	"context"
	"errors"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type RemoveMatchNPCRequest struct {
	UUID      uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
	SheetUUID uuid.UUID `path:"sheet_uuid" required:"true" doc:"NPC character sheet UUID"`
}

type RemoveMatchNPCResponse struct{}

func RemoveMatchNPCHandler(
	uc matchUC.IRemoveMatchNPC,
) func(context.Context, *RemoveMatchNPCRequest) (*RemoveMatchNPCResponse, error) {
	return func(ctx context.Context, req *RemoveMatchNPCRequest) (*RemoveMatchNPCResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		err := uc.Remove(ctx, &matchUC.RemoveMatchNPCInput{
			RequesterUUID: userUUID,
			MatchUUID:     req.UUID,
			SheetUUID:     req.SheetUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound),
				errors.Is(err, matchUC.ErrCharacterSheetNotFound),
				errors.Is(err, matchUC.ErrNPCNotInMatch):
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

		return &RemoveMatchNPCResponse{}, nil
	}
}
