package matchsession

import (
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// A morte no fechamento do turno (decisão do dono 4): o sistema aplica o Morto a quem o dano
// que o fechamento DE FATO aplicou deixou com o HP abaixo do mínimo da barra (um além do
// máximo negativo). Decidido pelo HP aplicado, não pelo "dies" da resolução: o cálculo é uma
// prévia, o fechamento é o que vale.

// IsDead diz se o personagem tem o Morto.
func (s *MatchSession) IsDead(charID uuid.UUID) bool {
	for _, e := range s.statusEffects[charID] {
		if e.Kind == statuseffect.KindDead {
			return true
		}
	}
	return false
}

// applyDeaths aplica o Morto a cada ficha danificada que ficou abaixo do mínimo e ainda não
// estava morta, e tira da fila (pelo mesmo caminho de qualquer status) o que ela tinha
// pendente. Devolve as gravações (uma aplicação por morto, com o evento ChangeApplied que a sala
// carimba) e as ações que saíram da fila, para a sala gravá-las como canceladas pelo mestre.
func (s *MatchSession) applyDeaths(damaged []DamagedCharacter, now time.Time) ([]statuseffect.Write, []StatusDrop) {
	var writes []statuseffect.Write
	var drops []StatusDrop
	for _, dc := range damaged {
		if s.IsDead(dc.CharacterID) || dc.Sheet == nil {
			continue
		}
		bar, ok := dc.Sheet.GetAllStatusBar()[enum.Health]
		if !ok || bar == nil || dc.NewHP >= bar.GetMin() {
			continue
		}
		next, added, err := statuseffect.Apply(s.statusEffects[dc.CharacterID], dc.CharacterID,
			statuseffect.Draft{Kind: statuseffect.KindDead}, uuid.New, now)
		if err != nil {
			// O Morto não tem parâmetros nem traz outro efeito: falhar aqui é um defeito.
			log.Printf("matchsession: Morto não aplicado em %s: %v", dc.CharacterID, err)
			continue
		}
		s.ReplaceStatusEffects(dc.CharacterID, next)
		for _, e := range added {
			writes = append(writes, statuseffect.Write{
				Effect: e.Clone(), Event: &statuseffect.Event{Change: statuseffect.ChangeApplied},
			})
		}
		// O turno já fechou: não há reação não aberta a refazer, só a fila.
		d, _ := s.DropRefusedByStatusWithReactions(dc.CharacterID)
		drops = append(drops, d...)
	}
	return writes, drops
}
