package enum

type PrincipleName string

const (
	Ten   PrincipleName = "Ten"
	Zetsu PrincipleName = "Zetsu"
	Ren   PrincipleName = "Ren"
	Gyo   PrincipleName = "Gyo"
	Hatsu PrincipleName = "Hatsu"
	Kou   PrincipleName = "Kou"
	Ken   PrincipleName = "Ken"
	Ryu   PrincipleName = "Ryu"
	In    PrincipleName = "In"
	En    PrincipleName = "En"
	// TODO: create SpiritualAttribute or similar for:
	AuraControl PrincipleName = "AuraControl" // CoA
	Aop         PrincipleName = "Aop"
)

func (pn PrincipleName) String() string {
	return string(pn)
}

func AllNenPrincipleNames() []PrincipleName {
	return []PrincipleName{
		Ten,
		Zetsu,
		Ren,
		Gyo,
		Hatsu,
		Kou,
		Ken,
		Ryu,
		In,
		En,
		AuraControl,
		Aop,
	}
}
