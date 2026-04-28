package action

type Skill struct {
	SkillName  string // enum.SkillName was superseded by mental tests (attrs) and
	Difficulty *int   // difficulty class (DC -> CD in pt-br)
	RollCheck
}
