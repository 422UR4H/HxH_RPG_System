package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// ScenarioUUID will now only be enabled for shared scenario
type CreateCampaignRequestBody struct {
	// ScenarioUUID     uuid.UUID `json:"scenario_uuid" required:"true" doc:"UUID of the scenario this campaign is based on"`
	Name                    string  `json:"name" required:"true" maxLength:"32" doc:"Name of the campaign"`
	BriefInitialDescription string  `json:"brief_initial_description" maxLength:"255" doc:"Brief initial description of the campaign"`
	Description             string  `json:"description" doc:"Full description of the campaign"`
	IsPublic                bool    `json:"is_public" default:"false" doc:"Whether the campaign is public or private"`
	CallLink                string  `json:"call_link" maxLength:"255" doc:"Link to the campaign call (e.g., Discord)"`
	StoryStartAt            string  `json:"story_start_at" required:"true" doc:"Date when the campaign story starts (YYYY-MM-DD)"`
	StoryCurrentAt          *string `json:"story_current_at,omitempty" doc:"Current date and time in the campaign story (ISO 8601)"`
}

type CreateCampaignRequest struct {
	Body CreateCampaignRequestBody `json:"body"`
}

type CreateCampaignResponseBody struct {
	UUID uuid.UUID `json:"uuid"`
	// ScenarioUUID     uuid.UUID `json:"scenario_uuid"`
	Name                    string  `json:"name"`
	BriefInitialDescription string  `json:"brief_initial_description"`
	Description             string  `json:"description"`
	IsPublic                bool    `json:"is_public"`
	CallLink                string  `json:"call_link"`
	StoryStartAt            string  `json:"story_start_at"`
	StoryCurrentAt          *string `json:"story_current_at,omitempty"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

type CreateCampaignResponse struct {
	Body   CreateCampaignResponseBody `json:"body"`
	Status int                        `json:"status"`
}

func CreateCampaignHandler(
	uc domainCampaign.ICreateCampaign,
) func(context.Context, *CreateCampaignRequest) (*CreateCampaignResponse, error) {

	return func(ctx context.Context, req *CreateCampaignRequest) (*CreateCampaignResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		storyStartAt, err := time.Parse("2006-01-02", req.Body.StoryStartAt)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid story_start_at date format, use YYYY-MM-DD")
		}

		var storyCurrentAtPtr *time.Time
		if req.Body.StoryCurrentAt != nil {
			storyCurrentAt, err := time.Parse(time.RFC3339, *req.Body.StoryCurrentAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid story_current_at format, use ISO 8601")
			}
			storyCurrentAtPtr = &storyCurrentAt
		}

		input := &domainCampaign.CreateCampaignInput{
			MasterUUID:              userUUID,
			ScenarioUUID:            nil, //req.Body.ScenarioUUID,
			Name:                    req.Body.Name,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
			CallLink:                req.Body.CallLink,
			StoryStartAt:            storyStartAt,
			StoryCurrentAt:          storyCurrentAtPtr,
		}

		campaign, err := uc.CreateCampaign(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrMaxCampaignsLimit):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, scenario.ErrScenarioNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var storyCurrentAtStr *string
		if campaign.StoryCurrentAt != nil {
			formattedTime := campaign.StoryCurrentAt.Format(time.RFC3339)
			storyCurrentAtStr = &formattedTime
		}

		response := CreateCampaignResponseBody{
			UUID: campaign.UUID,
			// ScenarioUUID:     campaign.ScenarioUUID,
			Name:                    campaign.Name,
			BriefInitialDescription: campaign.BriefInitialDescription,
			Description:             campaign.Description,
			IsPublic:                campaign.IsPublic,
			CallLink:                campaign.CallLink,
			StoryStartAt:            campaign.StoryStartAt.Format("2006-01-02"),
			StoryCurrentAt:          storyCurrentAtStr,
			CreatedAt:               campaign.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:               campaign.UpdatedAt.Format(http.TimeFormat),
		}
		return &CreateCampaignResponse{
			Body:   response,
			Status: http.StatusCreated,
		}, nil
	}
}
