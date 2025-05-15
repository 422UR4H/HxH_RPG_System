package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

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
}

type GetCampaignResponse struct {
	Body GetCampaignResponseBody `json:"body"`
}

func GetCampaignHandler(
	uc domainCampaign.IGetCampaign,
) func(context.Context, *GetCampaignRequest) (*GetCampaignResponse, error) {

	return func(ctx context.Context, req *GetCampaignRequest) (*GetCampaignResponse, error) {
		campaign, err := uc.GetCampaign(req.UUID)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
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

		response := GetCampaignResponseBody{
			UUID: campaign.UUID,
			// ScenarioUUID:     campaign.ScenarioUUID,
			Name:             campaign.Name,
			BriefDescription: campaign.BriefDescription,
			Description:      campaign.Description,
			StoryStartAt:     campaign.StoryStartAt.Format("2006-01-02"),
			StoryCurrentAt:   storyCurrentAtStr,
			StoryEndAt:       storyEndAtStr,
			CreatedAt:        campaign.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:        campaign.UpdatedAt.Format(http.TimeFormat),
		}

		return &GetCampaignResponse{
			Body: response,
		}, nil
	}
}
