package item

// TODO: improve weapons
type Weapon struct {
	dice    []int
	damage  int
	defense int
	// weight will directly determine penalty and stamina cost
	weight       float64
	height       float64
	volume       int
	isFireWeapon bool
}

// penality subtrai da agi, ats e flx
func NewWeapon(
	dice []int,
	damage int,
	defense int,
	height float64,
	weight float64,
	volume int,
	isFireWeapon bool,
) *Weapon {
	return &Weapon{
		dice:         dice,
		damage:       damage,
		defense:      defense,
		height:       height,
		weight:       weight,
		volume:       volume,
		isFireWeapon: isFireWeapon,
	}
}

func (w *Weapon) GetDice() []int {
	dice := make([]int, len(w.dice))
	copy(dice, w.dice)
	return dice
}

func (w *Weapon) GetPenality() float64 {
	if w.isFireWeapon {
		if w.weight >= 1.0 {
			return 1.0
		}
		return 0.0
	}
	return w.weight
}

func (w *Weapon) GetStaminaCost() float64 {
	if w.isFireWeapon {
		return 1.0
	}
	return w.weight
}

func (w *Weapon) GetDamage() int {
	return w.damage
}

func (w *Weapon) GetDefense() int {
	return w.defense
}

func (w *Weapon) GetWeight() float64 {
	return w.weight
}

func (w *Weapon) GetHeight() float64 {
	return w.height
}

func (w *Weapon) GetVolume() int {
	return w.volume
}

func (w *Weapon) IsFireWeapon() bool {
	return w.isFireWeapon
}
