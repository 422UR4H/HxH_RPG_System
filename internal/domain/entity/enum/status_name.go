package enum

type StatusName string

const (
	Health  StatusName = "Health"
	Stamina StatusName = "Stamina"
	Aura    StatusName = "Aura"
)

func (sn StatusName) String() string {
	return string(sn)
}
