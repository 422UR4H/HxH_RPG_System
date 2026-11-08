package match

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/google/uuid"
)

type Match struct {
	UUID                    uuid.UUID
	MasterUUID              uuid.UUID
	CampaignUUID            uuid.UUID
	Title                   string
	BriefInitialDescription string
	BriefFinalDescription   *string
	Description             string
	IsPublic                bool
	scenes                  []*scene.Scene
	events                  []GameEvent //nolint:unused // WIP: match system under development
	GameScheduledAt         time.Time
	GameStartAt             *time.Time
	StoryStartAt            time.Time
	StoryEndAt              *time.Time
	// StoryStartTime é o horário da história em que a partida começa: deslocamento desde a
	// meia-noite, em segundos inteiros. nil = partida sem horário.
	StoryStartTime *time.Duration
	// GameEndAt marca a partida encerrada (instante real). StoryEndAt é outro conceito
	// (fim da história na linha do tempo) e não indica encerramento.
	GameEndAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMatch(
	masterUUID uuid.UUID,
	campaignUUID uuid.UUID,
	title string,
	briefInitialDescription string,
	description string,
	isPublic bool,
	gameScheduledAt time.Time,
	storyStartAt time.Time,
) (*Match, error) {
	now := time.Now()
	return &Match{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		CampaignUUID:            campaignUUID,
		Title:                   title,
		BriefInitialDescription: briefInitialDescription,
		Description:             description,
		IsPublic:                isPublic,
		GameScheduledAt:         gameScheduledAt,
		StoryStartAt:            storyStartAt,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}

// IsFinished informa se a partida foi encerrada (GameEndAt gravado).
func (m *Match) IsFinished() bool {
	return m.GameEndAt != nil
}

func (m *Match) AddScene(scene *scene.Scene) {
	m.scenes = append(m.scenes, scene)
}

func (m *Match) GetScenes() []*scene.Scene {
	return m.scenes
}
