package matchsession

import (
	"bytes"
	"slices"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Efeitos de status na memória da sessão, por ficha (a mesma chave de charSheets). Quem chama
// segura o r.mu; a sessão não tem lock próprio. Toda leitura devolve cópias.

func cloneEffects(es []statuseffect.Effect) []statuseffect.Effect {
	out := make([]statuseffect.Effect, len(es))
	for i, e := range es {
		out[i] = e.Clone()
	}
	return out
}

// SetStatusEffects troca todos os efeitos de uma vez (nascimento da sessão). Fichas sem efeito
// não ficam no mapa.
func (s *MatchSession) SetStatusEffects(byChar map[uuid.UUID][]statuseffect.Effect) {
	s.statusEffects = make(map[uuid.UUID][]statuseffect.Effect, len(byChar))
	for id, es := range byChar {
		if len(es) > 0 {
			s.statusEffects[id] = cloneEffects(es)
		}
	}
}

// StatusEffectsOf devolve cópias dos efeitos da ficha, em ordem; nunca nil.
func (s *MatchSession) StatusEffectsOf(charID uuid.UUID) []statuseffect.Effect {
	return cloneEffects(s.statusEffects[charID])
}

// ReplaceStatusEffects troca o conjunto da ficha; vazio apaga a entrada.
func (s *MatchSession) ReplaceStatusEffects(charID uuid.UUID, effects []statuseffect.Effect) {
	if len(effects) == 0 {
		delete(s.statusEffects, charID)
		return
	}
	s.statusEffects[charID] = cloneEffects(effects)
}

// FindStatusEffect acha um efeito pelo id em qualquer ficha.
func (s *MatchSession) FindStatusEffect(id uuid.UUID) (statuseffect.Effect, bool) {
	for _, es := range s.statusEffects {
		for _, e := range es {
			if e.ID == id {
				return e.Clone(), true
			}
		}
	}
	return statuseffect.Effect{}, false
}

// CharactersWithStatusEffects lista as fichas com ao menos um efeito, em ordem estável.
func (s *MatchSession) CharactersWithStatusEffects() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(s.statusEffects))
	for id := range s.statusEffects {
		out = append(out, id)
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return out
}

// StatusSummaryOf resume as restrições que os efeitos da ficha impõem.
func (s *MatchSession) StatusSummaryOf(charID uuid.UUID) statuseffect.Summary {
	return statuseffect.Summarize(s.statusEffects[charID])
}

// IsPlayerParticipant diz se o usuário é jogador da partida (dono de uma ficha de jogador).
// O mestre não conta por causa dos NPCs: a entrada de um NPC não vai para o mapa de jogadores.
func (s *MatchSession) IsPlayerParticipant(userID uuid.UUID) bool {
	_, ok := s.participants[userID]
	return ok
}
