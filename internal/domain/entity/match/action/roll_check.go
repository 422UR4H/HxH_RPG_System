package action

type RollCheck struct {
	Context    RollContext // strategy set dice based on campaign\match rules
	SkillValue int         // filled with ValueForTest of the character sheet
	Result     int
}
