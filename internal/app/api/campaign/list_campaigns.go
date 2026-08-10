package campaign

import (
	"context"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListCampaignsResponseBody struct {
	Campaigns []CampaignSummaryResponse `json:"campaigns"`
}

type ListCampaignsResponse struct {
	Body ListCampaignsResponseBody `json:"body"`
}

type CampaignSummaryResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	Name                    string    `json:"name"`
	BriefInitialDescription string    `json:"briefInitialDescription"`
	BriefFinalDescription   *string   `json:"briefFinalDescription,omitempty"`
	IsPublic                bool      `json:"isPublic"`
	CallLink                string    `json:"callLink"`
	StoryStartAt            string    `json:"storyStartAt"`
	StoryCurrentAt          *string   `json:"storyCurrentAt,omitempty"`
	StoryEndAt              *string   `json:"storyEndAt,omitempty"`
	CreatedAt               string    `json:"createdAt"`
	UpdatedAt               string    `json:"updatedAt"`
}

func ListCampaignsHandler(
	uc campaignUC.IListCampaigns,
) func(context.Context, *struct{}) (*ListCampaignsResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListCampaignsResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaigns, err := uc.ListCampaigns(ctx, userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]CampaignSummaryResponse, 0, len(campaigns))
		for _, c := range campaigns {
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
			responses = append(responses, CampaignSummaryResponse{
				UUID:                    c.UUID,
				Name:                    c.Name,
				BriefInitialDescription: c.BriefInitialDescription,
				BriefFinalDescription:   c.BriefFinalDescription,
				IsPublic:                c.IsPublic,
				CallLink:                c.CallLink,
				StoryStartAt:            c.StoryStartAt.Format("2006-01-02"),
				StoryCurrentAt:          storyCurrentAtStr,
				StoryEndAt:              storyEndAtStr,
				CreatedAt:               c.CreatedAt.Format(http.TimeFormat),
				UpdatedAt:               c.UpdatedAt.Format(http.TimeFormat),
			})
		}

		return &ListCampaignsResponse{
			Body: ListCampaignsResponseBody{
				Campaigns: responses,
			},
		}, nil
	}
}
