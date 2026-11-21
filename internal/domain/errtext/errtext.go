// Package errtext cria erros cujo texto é uma frase em português, escrita para quem joga:
// é o que a API REST (detail) e o servidor do jogo (error.message) entregam na tela.
// Error() é só a frase, sem "sentinela: detalhe"; Unwrap mantém o sentinela para errors.Is.
package errtext

// Sentence cria um sentinela cujo texto é uma frase (com ponto final, que errors.New avisa).
func Sentence(msg string) error { return &detailed{msg: msg} }

// Wrap devolve um erro que lê como a frase msg e continua sendo o sentinela para errors.Is.
func Wrap(sentinel error, msg string) error {
	return &detailed{msg: msg, sentinel: sentinel}
}

type detailed struct {
	msg      string
	sentinel error
}

func (e *detailed) Error() string { return e.msg }
func (e *detailed) Unwrap() error { return e.sentinel }
