package enum

type PrincipleName uint8

const (
	Ten = iota
	Zetsu
	Ren
	Gyo
	Hatsu
	Kou
	Ken
	Ryu
	In
	En
	// TODO: create SpiritualAttribute or similar for:
	AuraControl // CoA
	Aop
)

func (pn PrincipleName) String() string {
	switch pn {
	case Ten:
		return "Ten"
	case Zetsu:
		return "Zetsu"
	case Ren:
		return "Ren"
	case Gyo:
		return "Gyo"
	case Hatsu:
		return "Hatsu"
	case Kou:
		return "Kou"
	case Ken:
		return "Ken"
	case Ryu:
		return "Ryu"
	case In:
		return "In"
	case En:
		return "En"
	case AuraControl:
		return "AuraControl"
	case Aop:
		return "Aop"
	}
	return "Unknown"
}
