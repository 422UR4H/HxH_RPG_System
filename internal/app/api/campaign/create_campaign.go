package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// ScenarioUUID will now only be enabled for shared scenario
type CreateCampaignRequestBody struct {
	// ScenarioUUID     uuid.UUID `json:"scenario_uuid" required:"true" doc:"UUID of the scenario this campaign is based on"`
	Name             string  `json:"name" required:"true" maxLength:"32" doc:"Name of the campaign"`
	BriefDescription string  `json:"brief_description" maxLength:"64" doc:"Brief description of the campaign"`
	Description      string  `json:"description" doc:"Full description of the campaign"`
	StoryStartAt     string  `json:"story_start_at" required:"true" doc:"Date when the campaign story starts (YYYY-MM-DD)"`
	StoryCurrentAt   *string `json:"story_current_at,omitempty" doc:"Current date and time in the campaign story (ISO 8601)"`
	StoryEndAt       *string `json:"story_end_at,omitempty" doc:"End date of the campaign story (YYYY-MM-DD)"`
}

type CreateCampaignRequest struct {
	Body CreateCampaignRequestBody `json:"body"`
}

type CreateCampaignResponseBody struct {
	Campaign CampaignResponse `json:"campaign"`
}

type CreateCampaignResponse struct {
	Body   CreateCampaignResponseBody `json:"body"`
	Status int                        `json:"status"`
}

type CampaignResponse struct {
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

func CreateCampaignHandler(
	uc domainCampaign.ICreateCampaign,
) func(context.Context, *CreateCampaignRequest) (*CreateCampaignResponse, error) {

	return func(ctx context.Context, req *CreateCampaignRequest) (*CreateCampaignResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		if len(req.Body.Name) < 5 {
			return nil, huma.Error422UnprocessableEntity(domainCampaign.ErrMinNameLength.Error())
		}

		if len(req.Body.Name) > 32 {
			return nil, huma.Error422UnprocessableEntity(domainCampaign.ErrMaxNameLength.Error())
		}

		if len(req.Body.BriefDescription) > 64 {
			return nil, huma.Error422UnprocessableEntity(domainCampaign.ErrMaxBriefDescLength.Error())
		}

		// if req.Body.ScenarioUUID == uuid.Nil {
		// 	return nil, huma.Error422UnprocessableEntity("scenario_uuid cannot be empty")
		// }

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

		var storyEndAtPtr *time.Time
		if req.Body.StoryEndAt != nil {
			storyEndAt, err := time.Parse("2006-01-02", *req.Body.StoryEndAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid story_end_at date format, use YYYY-MM-DD")
			}
			storyEndAtPtr = &storyEndAt
		}

		input := &domainCampaign.CreateCampaignInput{
			UserUUID:         userUUID,
			ScenarioUUID:     nil, //req.Body.ScenarioUUID,
			Name:             req.Body.Name,
			BriefDescription: req.Body.BriefDescription,
			Description:      req.Body.Description,
			StoryStartAt:     storyStartAt,
			StoryCurrentAt:   storyCurrentAtPtr,
			StoryEndAt:       storyEndAtPtr,
		}

		campaign, err := uc.CreateCampaign(input)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrScenarioNotFound):
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

		response := CampaignResponse{
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

		return &CreateCampaignResponse{
			Body: CreateCampaignResponseBody{
				Campaign: response,
			},
			Status: http.StatusCreated,
		}, nil
	}
}
