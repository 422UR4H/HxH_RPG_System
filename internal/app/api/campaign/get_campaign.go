package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCampaignRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the campaign to retrieve"`
}

type GetCampaignResponseBody struct {
	UUID uuid.UUID `json:"uuid"`
	// ScenarioUUID     uuid.UUID `json:"scenario_uuid"`
	Name             string  `json:"name"`
	BriefDescription string  `json:"brief_description"`
	Description      string  `json:"description"`
	StoryStartAt     string  `json:"story_start_at"`
	StoryCurrentAt   *string `json:"story_current_at,omitempty"`
	StoryEndAt       *string `json:"story_end_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`

	CharacterSheets []sheet.CharacterSummaryResponse `json:"character_sheets,omitempty"`
	Matches         []match.MatchSummaryResponse     `json:"matches,omitempty"`
}

type GetCampaignResponse struct {
	Body GetCampaignResponseBody `json:"body"`
}

func GetCampaignHandler(
	uc domainCampaign.IGetCampaign,
) func(context.Context, *GetCampaignRequest) (*GetCampaignResponse, error) {

	return func(ctx context.Context, req *GetCampaignRequest) (*GetCampaignResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		campaign, err := uc.GetCampaign(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var storyCurrentAtStr *string
		if campaign.StoryCurrentAt != nil {
			formattedTime := campaign.StoryCurrentAt.Format(time.RFC3339)
			storyCurrentAtStr = &formattedTime
		}

		var storyEndAtStr *string
		if campaign.StoryEndAt != nil {
			formattedDate := campaign.StoryEndAt.Format("2006-01-02")
			storyEndAtStr = &formattedDate
		}

		sheetsLen := len(campaign.CharacterSheets)
		characterSheets := make([]sheet.CharacterSummaryResponse, 0, sheetsLen)
		for _, cs := range campaign.CharacterSheets {
			characterSheets = append(characterSheets, sheet.ToSummaryResponse(cs))
		}

		matches := make([]match.MatchSummaryResponse, 0, len(campaign.Matches))
		for _, m := range campaign.Matches {
			matches = append(matches, match.ToSummaryResponse(&m))
		}

		response := GetCampaignResponseBody{
			UUID: campaign.UUID,
			// ScenarioUUID:     campaign.ScenarioUUID,
			Name:             campaign.Name,
			BriefDescription: campaign.BriefDescription,
			Description:      campaign.Description,
			StoryStartAt:     campaign.StoryStartAt.Format("2006-01-02"),
			StoryCurrentAt:   storyCurrentAtStr,
			StoryEndAt:       storyEndAtStr,
			CharacterSheets:  characterSheets,
			Matches:          matches,
			CreatedAt:        campaign.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:        campaign.UpdatedAt.Format(http.TimeFormat),
		}
		return &GetCampaignResponse{
			Body: response,
		}, nil
	}
}
