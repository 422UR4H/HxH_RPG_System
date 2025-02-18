package characterclass

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type ClassProfile struct {
	Name             enum.CharacterClassName
	Alignment        string
	Description      string
	BriefDescription string
}

func NewClassProfile(
	name enum.CharacterClassName,
	alignment string,
	description string,
	briefDescription string,
) *ClassProfile {
	return &ClassProfile{
		Name:             name,
		Alignment:        alignment,
		Description:      description,
		BriefDescription: briefDescription,
	}
}
