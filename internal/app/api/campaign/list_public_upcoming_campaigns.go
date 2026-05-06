package campaign

import (
	"context"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type PublicCampaignSummaryResponse struct {
	CampaignSummaryResponse
	NextGameScheduledAt *string `json:"next_game_scheduled_at,omitempty"`
}

type ListPublicCampaignsResponseBody struct {
	Campaigns []PublicCampaignSummaryResponse `json:"campaigns"`
}

type ListPublicCampaignsResponse struct {
	Body ListPublicCampaignsResponseBody `json:"body"`
}

func ListPublicUpcomingCampaignsHandler(
	uc domainCampaign.IListPublicUpcomingCampaigns,
) func(context.Context, *struct{}) (*ListPublicCampaignsResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListPublicCampaignsResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaigns, err := uc.ListPublicUpcomingCampaigns(ctx, userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]PublicCampaignSummaryResponse, 0, len(campaigns))
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
			var nextGameScheduledAtStr *string
			if c.NextGameScheduledAt != nil {
				formatted := c.NextGameScheduledAt.Format(time.RFC3339)
				nextGameScheduledAtStr = &formatted
			}
			responses = append(responses, PublicCampaignSummaryResponse{
				CampaignSummaryResponse: CampaignSummaryResponse{
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
				},
				NextGameScheduledAt: nextGameScheduledAtStr,
			})
		}

		return &ListPublicCampaignsResponse{
			Body: ListPublicCampaignsResponseBody{
				Campaigns: responses,
			},
		}, nil
	}
}
