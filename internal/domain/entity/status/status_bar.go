package status

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
)

type Bar struct {
	min  int
	curr int
	max  int
}

// TODO: implement one for each status bar (hp, sp, ap)
func NewStatusBar() *Bar {
	points := 0
	return &Bar{
		min:  points,
		curr: points,
		max:  points,
	}
}

func (b *Bar) IncreaseAt(value int) int {
	temp := b.curr + value
	if temp > b.max {
		b.curr = b.max
	} else {
		b.curr = temp
	}
	return b.curr
}

func (b *Bar) DecreaseAt(value int) int {
	temp := b.curr - value
	if temp < b.min {
		b.curr = b.min
	} else {
		b.curr = temp
	}
	return b.curr
}

func (b *Bar) Upgrade(skLvl int, attr attribute.IGameAttribute) {
	// old formula
	// this.hpMax = (this.getProHp() + this.modCon + this.coefHp) * this.lvl + this.valCon + Ficha.getHP_INICIAL();
	// new formula -> attrBonus * (skLvl + attrLvl + attrPoints)

	// TODO: Implement Min for hit_points
	// Min = generateStatus.GetLvl();
	// TODO: check how the buff interferes here
	fmt.Println("Upgrade status bar")
	fmt.Printf("skLvl: %d, attrLevel: %d, attrPoints: %d, attrBonus: %f\n", skLvl, attr.GetLevel(), attr.GetPoints(), attr.GetBonus())

	coeff := float64(skLvl + attr.GetLevel() + attr.GetPoints())
	maxVal := int(coeff * attr.GetBonus())
	if b.curr == b.max {
		b.curr = maxVal
	}
	// TODO: Implement else case (ex.: b.current == b.max - 1 -> threat % case)
	b.max = maxVal
}

func (b *Bar) GetMin() int {
	return b.min
}

func (b *Bar) GetCurrent() int {
	return b.curr
}

func (b *Bar) GetMax() int {
	return b.max
}
