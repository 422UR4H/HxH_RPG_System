package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type EnrollCharacterRequestBody struct {
	SheetUUID uuid.UUID `json:"sheet_uuid" required:"true" doc:"UUID of the character sheet"`
	MatchUUID uuid.UUID `json:"match_uuid" required:"true" doc:"UUID of the match"`
}

type EnrollCharacterRequest struct {
	Body EnrollCharacterRequestBody `json:"body"`
}

type EnrollCharacterResponse struct {
	Status int `json:"status"`
}

func EnrollCharacterHandler(
	uc domainEnrollment.IEnrollCharacterInMatch,
) func(context.Context, *EnrollCharacterRequest) (*EnrollCharacterResponse, error) {

	return func(ctx context.Context, req *EnrollCharacterRequest) (*EnrollCharacterResponse, error) {
		playerUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		err := uc.Enroll(ctx, req.Body.MatchUUID, req.Body.SheetUUID, playerUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, charactersheet.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, charactersheet.ErrNotCharacterSheetOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, domainEnrollment.ErrCharacterNotInCampaign):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, domainEnrollment.ErrCharacterAlreadyEnrolled):
				return nil, huma.Error409Conflict(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &EnrollCharacterResponse{
			Status: http.StatusCreated,
		}, nil
	}
}
