package scenario

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetScenarioRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the scenario to retrieve"`
}

type GetScenarioResponseBody struct {
	Scenario ScenarioWithCampaignsResponse `json:"scenario"`
}

type GetScenarioResponse struct {
	Body GetScenarioResponseBody `json:"body"`
}

type ScenarioWithCampaignsResponse struct {
	UUID             uuid.UUID                          `json:"uuid"`
	Name             string                             `json:"name"`
	BriefDescription string                             `json:"brief_description"`
	Description      string                             `json:"description"`
	Campaigns        []campaign.CampaignSummaryResponse `json:"campaigns"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func GetScenarioHandler(
	uc domainScenario.IGetScenario,
) func(context.Context, *GetScenarioRequest) (*GetScenarioResponse, error) {

	return func(ctx context.Context, req *GetScenarioRequest) (*GetScenarioResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		scenario, err := uc.GetScenario(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainScenario.ErrScenarioNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		campaignResponses := make([]campaign.CampaignSummaryResponse, 0, len(scenario.Campaigns))
		for _, c := range scenario.Campaigns {
			var storyCurrentAtStr *string
			if c.StoryCurrentAt != nil {
				formatted := c.StoryCurrentAt.Format("2006-01-02")
				storyCurrentAtStr = &formatted
			}

			var storyEndAtStr *string
			if c.StoryEndAt != nil {
				formatted := c.StoryEndAt.Format("2006-01-02")
				storyEndAtStr = &formatted
			}

			campaignResponses = append(campaignResponses, campaign.CampaignSummaryResponse{
				UUID:             c.UUID,
				Name:             c.Name,
				BriefDescription: c.BriefDescription,
				StoryStartAt:     c.StoryStartAt.Format("2006-01-02"),
				StoryCurrentAt:   storyCurrentAtStr,
				StoryEndAt:       storyEndAtStr,
				CreatedAt:        c.CreatedAt.Format(http.TimeFormat),
				UpdatedAt:        c.UpdatedAt.Format(http.TimeFormat),
			})
		}

		response := ScenarioWithCampaignsResponse{
			UUID:             scenario.UUID,
			Name:             scenario.Name,
			BriefDescription: scenario.BriefDescription,
			Description:      scenario.Description,
			Campaigns:        campaignResponses,
			CreatedAt:        scenario.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:        scenario.UpdatedAt.Format(http.TimeFormat),
		}

		return &GetScenarioResponse{
			Body: GetScenarioResponseBody{
				Scenario: response,
			},
		}, nil
	}
}
