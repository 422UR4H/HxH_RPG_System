package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateCampaignRequestBody struct {
	Name                    *string `json:"name,omitempty" doc:"Campaign name (5-32 characters)"`
	BriefInitialDescription *string `json:"brief_initial_description,omitempty" doc:"Brief description (max 255 characters)"`
	Description             *string `json:"description,omitempty" doc:"Full description"`
	IsPublic                *bool   `json:"is_public,omitempty" doc:"Public/private flag"`
	CallLink                *string `json:"call_link,omitempty" doc:"Call link URL (max 255 characters)"`
	StoryStartAt            *string `json:"story_start_at,omitempty" doc:"YYYY-MM-DD (locked after any match starts)"`
	StoryCurrentAt          *string `json:"story_current_at,omitempty" doc:"ISO 8601 date-time (cannot regress after match starts)"`
}

type UpdateCampaignRequest struct {
	UUID uuid.UUID                 `path:"uuid" required:"true" doc:"UUID of the campaign to update"`
	Body UpdateCampaignRequestBody
}

type CampaignEditResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	MasterUUID              uuid.UUID `json:"master_uuid"`
	Name                    string    `json:"name"`
	BriefInitialDescription string    `json:"brief_initial_description"`
	Description             string    `json:"description"`
	IsPublic                bool      `json:"is_public"`
	CallLink                string    `json:"call_link"`
	StoryStartAt            string    `json:"story_start_at"`
	StoryCurrentAt          *string   `json:"story_current_at,omitempty"`
	UpdatedAt               string    `json:"updated_at"`
}

type UpdateCampaignResponseBody struct {
	Campaign CampaignEditResponse `json:"campaign"`
}

type UpdateCampaignResponse struct {
	Body UpdateCampaignResponseBody
}

func UpdateCampaignHandler(
	uc campaignUC.IUpdateCampaign,
) func(context.Context, *UpdateCampaignRequest) (*UpdateCampaignResponse, error) {
	return func(ctx context.Context, req *UpdateCampaignRequest) (*UpdateCampaignResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		input := &campaignUC.UpdateCampaignInput{
			CampaignUUID:            req.UUID,
			MasterUUID:              userUUID,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
			CallLink:                req.Body.CallLink,
			Name:                    req.Body.Name,
		}

		if req.Body.StoryStartAt != nil {
			t, err := time.Parse("2006-01-02", *req.Body.StoryStartAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_start_at date format, use YYYY-MM-DD")
			}
			input.StoryStartAt = &t
		}
		if req.Body.StoryCurrentAt != nil {
			t, err := time.Parse(time.RFC3339, *req.Body.StoryCurrentAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_current_at format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
			}
			input.StoryCurrentAt = &t
		}

		c, err := uc.Update(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, campaignUC.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaignUC.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, campaignUC.ErrCampaignAlreadyEnded),
				errors.Is(err, campaignUC.ErrLockedAfterMatchStart),
				errors.Is(err, campaignUC.ErrCannotRegressStoryCurrentAt):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var storyCurrentAtStr *string
		if c.StoryCurrentAt != nil {
			s := c.StoryCurrentAt.Format(time.RFC3339)
			storyCurrentAtStr = &s
		}

		return &UpdateCampaignResponse{
			Body: UpdateCampaignResponseBody{
				Campaign: CampaignEditResponse{
					UUID:                    c.UUID,
					MasterUUID:              c.MasterUUID,
					Name:                    c.Name,
					BriefInitialDescription: c.BriefInitialDescription,
					Description:             c.Description,
					IsPublic:                c.IsPublic,
					CallLink:                c.CallLink,
					StoryStartAt:            c.StoryStartAt.Format("2006-01-02"),
					StoryCurrentAt:          storyCurrentAtStr,
					UpdatedAt:               c.UpdatedAt.Format(http.TimeFormat),
				},
			},
		}, nil
	}
}
