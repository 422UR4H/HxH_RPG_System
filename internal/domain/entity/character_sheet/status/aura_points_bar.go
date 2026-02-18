package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
)

// const AP_COEF_VALUE = 1000
// TODO: review this value to upgrade AP formula
const AP_COEF_VALUE = 10

type AuraPoints struct {
	spirituals    ability.IAbility
	conscienceNen attribute.IGameAttribute
	mop           skill.ISkill
	*Bar
}

func NewAuraPoints(
	spirituals ability.IAbility,
	conscienceNen attribute.IGameAttribute,
	mop skill.ISkill,
) (*AuraPoints, error) {
	if spirituals == nil {
		return nil, ErrSpiritualIsNil
	}
	bar := &AuraPoints{
		spirituals:    spirituals,
		conscienceNen: conscienceNen,
		mop:           mop,
		Bar:           &Bar{},
	}
	bar.Upgrade()

	return bar, nil
}

func (ap *AuraPoints) Upgrade() {
	bonus := int(ap.spirituals.GetBonus())
	// TODO: review formula
	// TODO: check how the buff interferes here
	// maxVal := AP_COEF_VALUE * (ap.mop.GetLevel() + ap.conscienceNen.GetValue() + bonus)

	// TODO: validar essa fórmula ousada que está no lugar da comentada acima
	// estou assumindo que o mopLvl é 0, o conscienceNenLvl é 1 (liberação dos shoukos)
	// AP_COEF_VALUE é 10 e o bonus: (spiritLvl + charLvl) / 2
	// é maior que 5 e menor que 10. logo fica em torno de 600 e 900 (parece razoável)
	// o MOP vai subir bastante naturalmente mesmo sem treino pelos outros parâmetros
	// e um cálculo de padeiro para valores grandes também pareceu razoável
	coef := float64(ap.mop.GetLevel() + ap.conscienceNen.GetLevel())
	maxVal := int(AP_COEF_VALUE * coef * float64(bonus))
	if ap.curr == ap.max {
		ap.curr = maxVal
	}
	// TODO: Implement else case (ex.: ap.current == ap.max - 1 -> threat % case)
	ap.max = maxVal
}
