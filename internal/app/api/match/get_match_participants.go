package match

import (
	"context"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiSheet "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchParticipantsRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
}

type ParticipantResponse struct {
	UUID     uuid.UUID                                     `json:"uuid"`
	JoinedAt string                                        `json:"joined_at"`
	LeftAt   *string                                       `json:"left_at,omitempty"`
	Sheet    apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
}

type GetMatchParticipantsResponseBody struct {
	Participants []ParticipantResponse `json:"participants"`
}

type GetMatchParticipantsResponse struct {
	Body GetMatchParticipantsResponseBody
}

func GetMatchParticipantsHandler(
	uc domainMatch.IGetMatchParticipants,
) func(context.Context, *GetMatchParticipantsRequest) (*GetMatchParticipantsResponse, error) {
	return func(ctx context.Context, req *GetMatchParticipantsRequest) (*GetMatchParticipantsResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		result, err := uc.Get(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		out := make([]ParticipantResponse, 0, len(result.Participants))
		for _, p := range result.Participants {
			out = append(out, toParticipantResponse(p, result.ViewerIsMaster))
		}
		return &GetMatchParticipantsResponse{
			Body: GetMatchParticipantsResponseBody{Participants: out},
		}, nil
	}
}

func toParticipantResponse(p *matchEntity.Participant, viewerIsMaster bool) ParticipantResponse {
	sheet := apiSheet.CharacterSheetWithVisibilityResponse{
		CharacterBaseSummaryResponse: apiSheet.ToBaseSummaryResponse(&p.Sheet),
		Private:                      nil,
	}
	if viewerIsMaster {
		priv := apiSheet.ToPrivateOnlyResponse(&p.Sheet)
		sheet.Private = &priv
	}

	var leftAtStr *string
	if p.LeftAt != nil {
		s := p.LeftAt.Format(time.RFC3339)
		leftAtStr = &s
	}

	return ParticipantResponse{
		UUID:     p.UUID,
		JoinedAt: p.JoinedAt.Format(time.RFC3339),
		LeftAt:   leftAtStr,
		Sheet:    sheet,
	}
}
