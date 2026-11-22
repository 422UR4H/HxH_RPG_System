package scenario

import (
	"context"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	scenarioUC "github.com/422UR4H/HxH_RPG_System/internal/application/scenario"
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
	BriefDescription string    `json:"briefDescription"`
	CreatedAt        string    `json:"createdAt"`
	UpdatedAt        string    `json:"updatedAt"`
}

func ListScenariosHandler(
	uc scenarioUC.IListScenarios,
) func(context.Context, *struct{}) (*ListScenariosResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListScenariosResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, apierr.NoUser()
		}

		scenarios, err := uc.ListScenarios(ctx, userUUID)
		if err != nil {
			return nil, apierr.Internal(err)
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
