package die

import (
	cryptoRand "crypto/rand"
	"math/big"
	mathRand "math/rand/v2"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Die struct {
	sides  enum.DieSides // name - ex.: D10, D20.
	result int
}

func NewDie(sides enum.DieSides) *Die {
	return &Die{
		sides: sides,
	}
}

func (d *Die) GetSides() int {
	return d.sides.GetSides()
}

func (d *Die) GetResult() int {
	return d.result
}

func (d *Die) Roll() int {
	sides := d.GetSides()

	// try to get secure true random with crypto/rand
	n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(sides)))
	if err == nil {
		d.result = int(n.Int64()) + 1
		return d.result
	}
	// TODO: log err
	// log.Printf("failed to create random number with crypto/rand: %v", err)

	// fallback
	d.result = mathRand.IntN(sides) + 1
	return d.result
}
