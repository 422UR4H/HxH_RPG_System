package statuseffect

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// HiddenRecord é o registro de um personagem Escondido num instante: o efeito Escondido e os
// jogadores que o percebiam então — a lista resolvida dele (ResolveListing), nada mais. É uma
// lista de PERMISSÃO: um leitor que não está nela não o percebia, inclusive quem nem era
// jogador da sessão (um de fora da partida pública, um membro da campanha não inscrito, um
// inscrito depois) — falha fechada. O mestre e o dono não são decididos aqui: o leitor do
// histórico os deixa passar antes. Só existe quando a lista não é "todos".
//
// O valor zero (Effect uuid.Nil, ninguém) é a sentinela de uma linha ilegível: esconde de
// todos e nenhuma revelação o cobre.
type HiddenRecord struct {
	Effect      uuid.UUID   `json:"effect"`
	PerceivedBy []uuid.UUID `json:"perceivedBy"`
}

// Conceals diz se o registro esconde o personagem de reader no instante at: reader não o
// percebia e nenhuma revelação do efeito em at ou depois o cobre. O chamador já deixou o
// mestre e o dono de fora.
func (h HiddenRecord) Conceals(reader uuid.UUID, at time.Time, reveals RevealLookup) bool {
	if slices.Contains(h.PerceivedBy, reader) {
		return false
	}
	return h.Effect == uuid.Nil || !reveals.Covers(h.Effect, at)
}

// HiddenFrom é, por personagem, o registro do Escondido dele no instante (HiddenRecord); só os
// personagens Escondidos com lista que não é "todos". nil = nenhum. É a forma de
// actions.hidden_from.
type HiddenFrom map[uuid.UUID]HiddenRecord

// Conceals diz se o personagem c fica escondido de reader no instante at; sem registro, não.
func (m HiddenFrom) Conceals(c, reader uuid.UUID, at time.Time, reveals RevealLookup) bool {
	h, ok := m[c]
	return ok && h.Conceals(reader, at, reveals)
}
