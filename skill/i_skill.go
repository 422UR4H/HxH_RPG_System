package skill

type ISkill interface {
	GetLvl() int
	GetValueForTest() int
	GetExpPoints() int
	IncreaseExp(points int) int
}
