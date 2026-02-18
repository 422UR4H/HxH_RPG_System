package enum

type PrincipleName string

const (
	Ten   PrincipleName = "Ten"
	Zetsu PrincipleName = "Zetsu"
	Ren   PrincipleName = "Ren"
	Gyo   PrincipleName = "Gyo"
	Hatsu PrincipleName = "Hatsu"
	Shu   PrincipleName = "Shu"
	Kou   PrincipleName = "Kou"
	Ken   PrincipleName = "Ken"
	Ryu   PrincipleName = "Ryu"
	In    PrincipleName = "In"
	En    PrincipleName = "En"
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
		Shu,
		Kou,
		Ken,
		Ryu,
		In,
		En,
	}
}
