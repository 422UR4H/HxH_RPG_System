package action

type RollCheck struct {
	Context    RollContext // strategy set dice based on campaign\match rules
	SkillName  string      // skill used for the roll check (test)
	SkillValue int         // filled with ValueForTest of the character sheet
	Result     int
}
