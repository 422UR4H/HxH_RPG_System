package scenario

import (
	"context"
	"errors"
	"net/http"

	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetScenarioRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the scenario to retrieve"`
}

type GetScenarioResponseBody struct {
	Scenario ScenarioResponse `json:"scenario"`
}

type GetScenarioResponse struct {
	Body GetScenarioResponseBody `json:"body"`
}

func GetScenarioHandler(
	uc domainScenario.IGetScenario,
) func(context.Context, *GetScenarioRequest) (*GetScenarioResponse, error) {

	return func(ctx context.Context, req *GetScenarioRequest) (*GetScenarioResponse, error) {
		scenario, err := uc.GetScenario(req.UUID)
		if err != nil {
			switch {
			case errors.Is(err, domainScenario.ErrScenarioNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		response := ScenarioResponse{
			UUID:             scenario.UUID,
			Name:             scenario.Name,
			BriefDescription: scenario.BriefDescription,
			Description:      scenario.Description,
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
