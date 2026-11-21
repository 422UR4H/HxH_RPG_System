package match

import (
	"context"
	"errors"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// IStatusEffectEventChecker diz se o efeito é revelável na partida: tem evento de status em
// qualquer partida e a ficha dele participa desta (um Escondido aplicado em outra partida e
// ainda ativo nesta esconde turnos daqui). A implementação real é pgstatuseffect.Repository.
type IStatusEffectEventChecker interface {
	RevealableInMatch(ctx context.Context, matchUUID, effectUUID uuid.UUID) (bool, error)
}

// IStatusEffectRevealWriter grava uma revelação. A implementação real é
// pgstatuseffect.Repository.
type IStatusEffectRevealWriter interface {
	InsertReveal(ctx context.Context, matchUUID, effectUUID uuid.UUID, at time.Time) error
}

type IRevealStatusEffect interface {
	Reveal(ctx context.Context, matchUUID, effectUUID, userUUID uuid.UUID) error
}

// RevealStatusEffectUC é o comando do mestre que torna visível a todos, no GET /history, o que
// um efeito de status escondeu até agora (Revision R3). Não tem regra de jogo: só confere quem
// pede e o que existe. Vale durante e depois da partida.
type RevealStatusEffectUC struct {
	matchReader IMatchReader
	events      IStatusEffectEventChecker
	reveals     IStatusEffectRevealWriter
}

func NewRevealStatusEffectUC(
	matchReader IMatchReader, events IStatusEffectEventChecker, reveals IStatusEffectRevealWriter,
) *RevealStatusEffectUC {
	return &RevealStatusEffectUC{matchReader: matchReader, events: events, reveals: reveals}
}

// Reveal grava uma revelação nova (cada chamada é uma linha: uma posterior cobre os eventos
// posteriores). Partida inexistente = ErrMatchNotFound; quem não é o mestre = ErrNotMatchMaster;
// efeito não revelável aqui (sem evento em partida alguma, ou de ficha que não participa desta)
// = statuseffect.ErrNotFound. O efeito uuid.Nil é a sentinela do
// histórico (linha ilegível escondida de todos) e nunca é revelável.
func (uc *RevealStatusEffectUC) Reveal(ctx context.Context, matchUUID, effectUUID, userUUID uuid.UUID) error {
	mt, err := uc.matchReader.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	if mt.MasterUUID != userUUID {
		return ErrNotMatchMaster
	}
	if effectUUID == uuid.Nil {
		return statuseffect.ErrNotFound
	}
	ok, err := uc.events.RevealableInMatch(ctx, matchUUID, effectUUID)
	if err != nil {
		return err
	}
	if !ok {
		return statuseffect.ErrNotFound
	}
	return uc.reveals.InsertReveal(ctx, matchUUID, effectUUID, time.Now().UTC().Truncate(time.Microsecond))
}
