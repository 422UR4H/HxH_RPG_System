package match

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiSheet "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListMatchEnrollmentsRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the match"`
}

type ListMatchEnrollmentsResponse struct {
	Body ListMatchEnrollmentsResponseBody `json:"body"`
}

type ListMatchEnrollmentsResponseBody struct {
	Enrollments []EnrollmentResponse `json:"enrollments"`
}

type EnrollmentResponse struct {
	UUID           uuid.UUID                                    `json:"uuid"`
	Status         string                                       `json:"status"`
	CreatedAt      string                                       `json:"created_at"`
	CharacterSheet apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
	Player         PlayerRefResponse                            `json:"player"`
}

type PlayerRefResponse struct {
	UUID uuid.UUID `json:"uuid"`
	Nick string    `json:"nick"`
}

func ListMatchEnrollmentsHandler(
	uc domainMatch.IListMatchEnrollments,
) func(context.Context, *ListMatchEnrollmentsRequest) (*ListMatchEnrollmentsResponse, error) {
	return func(ctx context.Context, req *ListMatchEnrollmentsRequest) (*ListMatchEnrollmentsResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		result, err := uc.List(ctx, req.UUID, userUUID)
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

		out := make([]EnrollmentResponse, 0, len(result.Enrollments))
		for _, e := range result.Enrollments {
			out = append(out, toEnrollmentResponse(e, result.ViewerIsMaster))
		}
		return &ListMatchEnrollmentsResponse{
			Body: ListMatchEnrollmentsResponseBody{Enrollments: out},
		}, nil
	}
}

func toEnrollmentResponse(e *enrollmentEntity.Enrollment, viewerIsMaster bool) EnrollmentResponse {
	sheet := apiSheet.CharacterSheetWithVisibilityResponse{
		CharacterBaseSummaryResponse: apiSheet.ToBaseSummaryResponse(&e.CharacterSheet),
		Private:                      nil,
	}
	if viewerIsMaster {
		p := apiSheet.ToPrivateOnlyResponse(&e.CharacterSheet)
		sheet.Private = &p
	}
	return EnrollmentResponse{
		UUID:           e.UUID,
		Status:         e.Status,
		CreatedAt:      e.CreatedAt.Format(http.TimeFormat),
		CharacterSheet: sheet,
		Player: PlayerRefResponse{
			UUID: e.Player.UUID,
			Nick: e.Player.Nick,
		},
	}
}
