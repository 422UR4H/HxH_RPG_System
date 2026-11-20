package actioncodec

import (
	"encoding/json"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// EncodeHiddenFrom é a coluna hidden_from (actions):
// {"<charUUID>": {"effect": "<hiddenEffectUUID>", "perceivedBy": ["<playerUUID>", ...]}} — por
// personagem Escondido no instante, o efeito e quem o percebia (lista de permissão). nil (SQL
// NULL) quando nenhum, nunca '{}'. perceivedBy vazio sai [] (ninguém além do mestre e do dono).
func EncodeHiddenFrom(m statuseffect.HiddenFrom) ([]byte, error) {
	if len(m) == 0 {
		return nil, nil
	}
	out := make(statuseffect.HiddenFrom, len(m))
	for c, h := range m {
		if h.PerceivedBy == nil {
			h.PerceivedBy = []uuid.UUID{}
		}
		out[c] = h
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal hidden from: %w", err)
	}
	return raw, nil
}

// DecodeHiddenFrom lê hidden_from. NULL (ou '{}') = nenhum (nil). Um valor ilegível falha
// FECHADO: devolve o erro (o chamador registra) junto do registro-sentinela do personagem da
// própria linha (charUUID: o ator, ou o reator) — o valor zero de statuseffect.HiddenRecord,
// que esconde de todo leitor que não é o mestre nem o dono e que nenhuma revelação cobre.
func DecodeHiddenFrom(raw []byte, charUUID uuid.UUID) (statuseffect.HiddenFrom, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m statuseffect.HiddenFrom
	if err := json.Unmarshal(raw, &m); err != nil {
		return statuseffect.HiddenFrom{charUUID: {}}, err
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}
