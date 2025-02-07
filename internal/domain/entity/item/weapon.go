package item

type Weapon struct {
	dice        []int
	damage      int
	defense     int
	penalty     int
	staminaCost int
	height      float64
	weight      float64
	volume      int
}

// penality subtrai da agi, ats e flx
// TODO: tentar calcular weight com base nela
func NewWeapon(
	dice []int,
	damage int,
	defense int,
	penalty int,
	staminaCost int,
	height float64,
	weight float64,
	volume int,
) *Weapon {
	return &Weapon{
		dice:        dice,
		damage:      damage,
		defense:     defense,
		penalty:     penalty,
		staminaCost: staminaCost,
		height:      height,
		weight:      weight,
		volume:      volume,
	}
}

func (w *Weapon) GetDice() []int {
	dice := make([]int, len(w.dice))
	copy(dice, w.dice)
	return dice
}

func (w *Weapon) GetDamage() int {
	return w.damage
}

func (w *Weapon) GetDefense() int {
	return w.defense
}

func (w *Weapon) GetPenality() int {
	return w.penalty
}

func (w *Weapon) GetStaminaCost() int {
	return w.staminaCost
}

func (w *Weapon) GetHeight() float64 {
	return w.height
}

func (w *Weapon) GetWeight() float64 {
	return w.weight
}

func (w *Weapon) GetVolume() int {
	return w.volume
}
