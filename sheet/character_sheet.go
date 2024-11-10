package charactersheet

import (
	"github.com/422UR4H/HxH_RPG_Environment.Domain/ability"
	"github.com/422UR4H/HxH_RPG_Environment.Domain/attribute"
	"github.com/422UR4H/HxH_RPG_Environment.Domain/skill"
)

type CharacterSheet struct {
	profile   CharacterProfile
	ability   ability.Manager
	attribute attribute.Manager
	skill     skill.Manager
	principle principle.Manager
}
