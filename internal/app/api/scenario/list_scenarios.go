package scenario

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListScenariosResponseBody struct {
	Scenarios []ScenarioSummaryResponse `json:"scenarios"`
}

type ListScenariosResponse struct {
	Body ListScenariosResponseBody `json:"body"`
}

type ScenarioSummaryResponse struct {
	UUID             uuid.UUID `json:"uuid"`
	Name             string    `json:"name"`
	BriefDescription string    `json:"brief_description"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

func ListScenariosHandler(
	uc domainScenario.IListScenarios,
) func(context.Context, *struct{}) (*ListScenariosResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListScenariosResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		scenarios, err := uc.ListScenarios(userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]ScenarioSummaryResponse, 0, len(scenarios))
		for _, s := range scenarios {
			responses = append(responses, ScenarioSummaryResponse{
				UUID:             s.UUID,
				Name:             s.Name,
				BriefDescription: s.BriefDescription,
				CreatedAt:        s.CreatedAt.Format(http.TimeFormat),
				UpdatedAt:        s.UpdatedAt.Format(http.TimeFormat),
			})
		}

		return &ListScenariosResponse{
			Body: ListScenariosResponseBody{
				Scenarios: responses,
			},
		}, nil
	}
}
