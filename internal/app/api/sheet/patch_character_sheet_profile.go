package sheet

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type IProfileImageUpdater interface {
	UpdateCharacterSheetProfile(ctx context.Context, sheetUUID, playerUUID uuid.UUID, avatarURL, coverURL *string) error
}

type PatchCharacterSheetProfileRequestBody struct {
	AvatarURL *string `json:"avatar_url,omitempty"`
	CoverURL  *string `json:"cover_url,omitempty"`
}

type PatchCharacterSheetProfileRequest struct {
	UUID string `path:"uuid"`
	Body PatchCharacterSheetProfileRequestBody
}

type PatchCharacterSheetProfileResponse struct {
	Status int
}

func PatchCharacterSheetProfileHandler(
	repo IProfileImageUpdater,
) func(context.Context, *PatchCharacterSheetProfileRequest) (*PatchCharacterSheetProfileResponse, error) {
	return func(ctx context.Context, req *PatchCharacterSheetProfileRequest) (*PatchCharacterSheetProfileResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		err = repo.UpdateCharacterSheetProfile(ctx, sheetUUID, userUUID, req.Body.AvatarURL, req.Body.CoverURL)
		if err != nil {
			if errors.Is(err, cs.ErrCharacterSheetNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &PatchCharacterSheetProfileResponse{Status: http.StatusNoContent}, nil
	}
}
