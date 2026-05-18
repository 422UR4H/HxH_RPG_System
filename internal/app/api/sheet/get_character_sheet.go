package sheet

import (
	"context"
	"errors"
	"log"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCharacterSheetRequest struct {
	UUID    string `path:"uuid" required:"true" doc:"UUID of the character sheet"`
	Include string `query:"include" required:"false" doc:"Comma-separated list of includes: submission"`
}

type GetCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type GetCharacterSheetResponse struct {
	Body   GetCharacterSheetResponseBody `json:"body"`
	Status int                           `json:"status"`
}

func GetCharacterSheetHandler(
	uc cs.IGetCharacterSheet,
	submissionFetcher cs.ISubmissionFetcher,
) func(context.Context, *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {

	return func(ctx context.Context, req *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		charSheetId, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := uc.GetCharacterSheet(ctx, charSheetId, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				log.Printf("[ERROR] GetCharacterSheet uuid=%s: %v", req.UUID, err)
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		response := NewCharacterSheetResponse(characterSheet)

		if req.Include == "submission" || containsInclude(req.Include, "submission") {
			info, err := submissionFetcher.GetSubmissionInfoBySheetUUID(ctx, charSheetId)
			if err != nil {
				log.Printf("[ERROR] GetSubmissionInfo uuid=%s: %v", req.UUID, err)
				return nil, huma.Error500InternalServerError(err.Error())
			}
			if info != nil {
				response.Submission = &SubmissionResponse{
					CampaignUUID: info.CampaignUUID.String(),
					CreatedAt:    info.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				}
			}
		}

		return &GetCharacterSheetResponse{
			Body: GetCharacterSheetResponseBody{
				CharacterSheet: *response,
			},
			Status: http.StatusOK,
		}, nil
	}
}

// containsInclude checks if a comma-separated include string contains the target.
func containsInclude(include, target string) bool {
	if include == "" {
		return false
	}
	for _, s := range splitIncludes(include) {
		if s == target {
			return true
		}
	}
	return false
}

func splitIncludes(include string) []string {
	result := []string{}
	start := 0
	for i := 0; i <= len(include); i++ {
		if i == len(include) || include[i] == ',' {
			part := trimSpace(include[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
