package game

import (
	"errors"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
)

// genericGameError é o que a pessoa lê quando o erro de um comando não é uma frase de mesa
// (erro de gravação, de leitura, algo inesperado). O motivo real fica no log.
const genericGameError = "Não foi possível fazer isso agora."

// gameErrorText é a frase de um erro de comando para a tela: a do próprio erro quando ele foi
// escrito para ser lido (errtext) e a genérica, com o erro no log, quando não foi.
func gameErrorText(err error) string { return errorTextOr(err, genericGameError) }

// errorTextOr é gameErrorText com a frase genérica escolhida por quem chama.
func errorTextOr(err error, fallback string) string {
	var block *matchsession.StatusBlockError
	if errors.As(err, &block) {
		return block.Error()
	}
	if text, ok := errtext.Of(err); ok {
		return text
	}
	log.Printf("game command refused: %v", err)
	return fallback
}
