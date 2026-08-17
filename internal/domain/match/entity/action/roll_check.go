package action

type RollCheck struct {
	Context    RollContext // strategy set dice based on campaign\match rules
	SkillName  string      // skill used for the roll check (test)
	SkillValue int         // filled with ValueForTest of the character sheet
	// Attempts are the dice, rolled once when the action arrived. Derive reads them as
	// many times as the master edits, and never rolls again.
	Attempts RollAttempts
	Result   int
}
