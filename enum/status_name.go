package enum

type StatusName int

const (
	Health StatusName = iota
	Stamina
	Aura
)

func (sn StatusName) String() string {
	switch sn {
	case Health:
		return "Health"
	case Stamina:
		return "Stamina"
	case Aura:
		return "Aura"
	}
	return "Unknown"
}

// TODO: create SpiritualAttribute or similar to handle these attributes below
// AuraControl
// Aop
