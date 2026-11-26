package matchsession

import (
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// deriveCheck deriva um teste e, se faltarem dados para o viés que ele lê (Short), tira só
// os que faltam, uma vez (RollCalculator.TopUp), e deriva de novo. É assim que um teste
// passivo vira rolado no primeiro viés — "se pôs vantagem, é rolado" (decisão do dono,
// 2026-10-12) — e que um viés que cresce ganha seus extras. As velocidades e skills[] pedem
// aqui (deriveSpeeds, deriveSkills); os testes que o resolvedor lê pedem por TurnResolution.DiceShort
// (drawShortDice).
//
// Nunca é uma nova rolagem: o que caiu fica, então tirar o viés e pô-lo de volta lê os mesmos
// dados. É a mesma regra que applySkillEdit já segue para uma perícia que não existia.
func (s *MatchSession) deriveCheck(rc *action.RollCheck, in service.RollInput) service.RollOutcome {
	calc := service.RollCalculator{}
	out := calc.Derive(s.rules, rc.Attempts, in)
	if out.Short > 0 {
		calc.TopUp(s.rules, &rc.Attempts, out.Short, s.rollSource)
		out = calc.Derive(s.rules, rc.Attempts, in)
	}
	return out
}

// drawShortDice tira, uma vez, os dados que a resolução disse faltar, cada um no teste vivo
// que a falta endereça — o mesmo endereço do edit_action (actionOnTurn + resolveRollCheck;
// "defense" numa reação que mantém a padrão cria a DefaultDefense). Um endereço que não se
// encontra é um defeito do resolvedor: fica no log, e a segunda resolução o informa de novo.
func (s *MatchSession) drawShortDice(t *turn.Turn, short []service.DiceShortage) {
	calc := service.RollCalculator{}
	for _, ds := range short {
		a, err := s.actionOnTurn(t, ds.Ref.ActionID)
		if err != nil {
			log.Printf("matchsession: falta de dados numa ação fora do turno %s: %+v: %v", t.GetID(), ds, err)
			continue
		}
		rc, err := resolveRollCheck(a, action.ConditionEdit{Field: ds.Ref.Field, SkillName: ds.Ref.SkillName})
		if err != nil {
			log.Printf("matchsession: falta de dados num teste que a ação não tem: %+v: %v", ds, err)
			continue
		}
		calc.TopUp(s.rules, &rc.Attempts, ds.Need, s.rollSource)
	}
}
