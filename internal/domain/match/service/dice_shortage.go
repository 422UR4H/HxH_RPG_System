package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// DiceRef endereça um teste numa ação do turno do mesmo jeito que o edit_action: pela ação
// (o zero é a ação do turno) e por Field ou SkillName, nunca pelos dois.
type DiceRef struct {
	ActionID  uuid.UUID
	Field     action.ConditionField
	SkillName string
}

// DiceShortage é um teste que o resolvedor leu com menos dados do que o viés pede: Need é o
// Short do Derive. O resolvedor nunca rola — quem pode rolar (a sessão) tira esses dados uma
// vez e resolve de novo. Nunca vai ao fio nem é gravado.
type DiceShortage struct {
	Ref  DiceRef
	Need int
}

// noteShortage registra a falta de um teste, uma vez por teste: o acerto é derivado de novo
// para cada alvo da cadeia, com os mesmos dados, e não pode pedir duas vezes os mesmos
// extras. Fica a maior falta lida.
func noteShortage(list []DiceShortage, ref DiceRef, need int) []DiceShortage {
	if need <= 0 {
		return list
	}
	for i := range list {
		if list[i].Ref == ref {
			list[i].Need = max(list[i].Need, need)
			return list
		}
	}
	return append(list, DiceShortage{Ref: ref, Need: need})
}
