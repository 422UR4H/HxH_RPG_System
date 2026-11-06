package action

import "github.com/google/uuid"

// IsBlockedIn diz se esta action está bloqueada pela fila dada: é um movimento empilhado
// (Move.StackedOn != uuid.Nil) cujo anterior ainda está entre os pendentes (spec 2026-11-06 D4).
// Uma action bloqueada não é candidata: não abre, não é puxada e não é o que uma reação consome
// enquanto houver outra.
//
// É o ÚNICO lugar da regra. "Pendente" é o que quem pergunta diz que é: a fila de verdade para a
// sessão e para a PriorityQueue, e o que ainda não foi escolhido na simulação de
// RoundScheduler.ProjectRound.
func (a *Action) IsBlockedIn(pending []*Action) bool {
	if a.Move == nil || a.Move.StackedOn == uuid.Nil {
		return false
	}
	for _, p := range pending {
		if p.GetID() == a.Move.StackedOn {
			return true
		}
	}
	return false
}
