package scenario

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type CreateScenarioRequestBody struct {
	Name             string `json:"name" required:"true" maxLength:"32" doc:"Name of the scenario"`
	BriefDescription string `json:"brief_description" maxLength:"64" doc:"Brief description of the scenario"`
	Description      string `json:"description" doc:"Full description of the scenario"`
}

type CreateScenarioRequest struct {
	Body CreateScenarioRequestBody `json:"body"`
}

type CreateScenarioResponseBody struct {
	Scenario ScenarioResponse `json:"scenario"`
}

type CreateScenarioResponse struct {
	Body   CreateScenarioResponseBody `json:"body"`
	Status int                        `json:"status"`
}

type ScenarioResponse struct {
	UUID             uuid.UUID `json:"uuid"`
	Name             string    `json:"name"`
	BriefDescription string    `json:"brief_description"`
	Description      string    `json:"description"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

func CreateScenarioHandler(
	uc domainScenario.ICreateScenario,
) func(context.Context, *CreateScenarioRequest) (*CreateScenarioResponse, error) {

	return func(ctx context.Context, req *CreateScenarioRequest) (*CreateScenarioResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		input := &domainScenario.CreateScenarioInput{
			UserUUID:         userUUID,
			Name:             req.Body.Name,
			BriefDescription: req.Body.BriefDescription,
			Description:      req.Body.Description,
		}
		scenario, err := uc.CreateScenario(input)
		if err != nil {
			switch {
			case errors.Is(err, domainScenario.ErrScenarioNameAlreadyExists):
				return nil, huma.Error409Conflict(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
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

		return &CreateScenarioResponse{
			Body: CreateScenarioResponseBody{
				Scenario: response,
			},
			Status: http.StatusCreated,
		}, nil
	}
}
