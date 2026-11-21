package sheet

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteCharacterSheetRequest struct {
	UUID string `path:"uuid" required:"true"`
}

type DeleteCharacterSheetResponse struct {
	Status int
}

func DeleteCharacterSheetHandler(
	uc cs.IDeleteCharacterSheet,
) func(context.Context, *DeleteCharacterSheetRequest) (*DeleteCharacterSheetResponse, error) {
	return func(ctx context.Context, req *DeleteCharacterSheetRequest) (*DeleteCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, apierr.NoUser()
		}

		sheetUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("Ficha inválida.")
		}

		err = uc.DeleteCharacterSheet(ctx, sheetUUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, auth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, cs.ErrCharacterSheetNotFreeToManage):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, apierr.Internal(err)
			}
		}

		return &DeleteCharacterSheetResponse{Status: http.StatusNoContent}, nil
	}
}
