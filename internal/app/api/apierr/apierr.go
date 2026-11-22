// Package apierr reúne as respostas de erro de servidor dos handlers REST. O erro interno nunca
// vai para a tela: o detail é uma frase fixa e o motivo real fica no log.
package apierr

import (
	"log"
	"runtime"
	"strings"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"github.com/danielgtaylor/huma/v2"
)

// InternalMessage é o que a pessoa lê quando algo dá errado por dentro.
const InternalMessage = "Algo deu errado. Tente de novo."

// NoUserMessage é o que a pessoa lê quando o token não traz a conta (não deveria acontecer
// com o middleware de auth no ar).
const NoUserMessage = "Não foi possível identificar sua conta. Entre de novo."

// Internal devolve um 500 com a frase fixa e loga o erro original com o nome de quem chamou.
func Internal(err error) huma.StatusError {
	log.Printf("[ERROR] %s: %v", caller(), err)
	return huma.Error500InternalServerError(InternalMessage)
}

// FallbackMessage é o que a pessoa lê num 4xx cujo erro não é uma frase de mesa.
const FallbackMessage = "Não foi possível fazer isso."

// Text é o detail de um 4xx: a frase do erro (errtext) ou, se ele não é uma frase, a genérica,
// com o erro original no log.
func Text(err error) string {
	if text, ok := errtext.Of(err); ok {
		return text
	}
	log.Printf("[ERROR] %s: refusal without a sentence: %v", caller(), err)
	return FallbackMessage
}

// NoUser devolve o 500 de "o contexto não tem o usuário".
func NoUser() huma.StatusError {
	log.Printf("[ERROR] %s: user id missing from context", caller())
	return huma.Error500InternalServerError(NoUserMessage)
}

func caller() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "handler"
	}
	name := runtime.FuncForPC(pc).Name()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}
