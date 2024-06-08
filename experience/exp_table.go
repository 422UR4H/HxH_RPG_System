package experience

import (
	"fmt"
	"math"
)

const (
	MAX_LVL int8    = 101
	A_PARAM float64 = 3.7
	B_PARAM float64 = 2.8
)

type ExpTable struct {
	coefficient    float64
	baseTable      [MAX_LVL]int
	aggregateTable [MAX_LVL]int
}

func expTableFunction(lvl int) float64 {
	return (1700.0 / (1.0 + math.Pow(math.E, A_PARAM/10.0*(12.0-float64(lvl))))) +
		(1800.0 / (1.0 + math.Pow(math.E, A_PARAM/10.0*(38.0-float64(lvl))))) +
		(2000.0 / (1.0 + math.Pow(math.E, B_PARAM/10.0*(74.0-float64(lvl)))))
}

func NewExpTable(coefficient float64) *ExpTable {
	expTable := &ExpTable{coefficient: coefficient}
	expTable.baseTable[0] = 0
	expTable.aggregateTable[0] = 0

	for i := 1; i < int(MAX_LVL); i++ {
		currExp := int(expTable.coefficient * expTableFunction(i))
		expTable.baseTable[i] = currExp
		expTable.aggregateTable[i] = expTable.aggregateTable[i-1] + currExp
	}
	return expTable
}

func NewDefaultExpTable() *ExpTable {
	return NewExpTable(1.0)
}

func (e *ExpTable) GetBaseExpByLvl(lvl int) int {
	return e.baseTable[lvl]
}

func (e *ExpTable) GetAggregateExpByLvl(lvl int) int {
	return e.aggregateTable[lvl]
}

func (e *ExpTable) GetLvlByExp(exp int) int {
	for lvl := len(e.aggregateTable) - 1; lvl >= 0; lvl-- {
		if e.aggregateTable[lvl] <= exp {
			return lvl
		}
	}
	return 0
}

func (e *ExpTable) ToString() string {
	expTable := "=============================\n"
	expTable += "Coef: "
	expTable += fmt.Sprintf("%.1f\n", e.coefficient)
	expTable += "Lvl\t| Base\t| Total\n"
	for lvl := 0; lvl < int(MAX_LVL); lvl++ {
		expTable += " " + fmt.Sprint(lvl) + "\t| "
		expTable += fmt.Sprint(e.baseTable[lvl]) + "\t| "
		expTable += fmt.Sprint(e.aggregateTable[lvl]) + "\n"
	}
	expTable += "=============================\n"
	return expTable
}
