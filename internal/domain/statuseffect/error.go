package statuseffect

// Os textos destes erros chegam ao mestre na tela: são frases em português, escritas para quem
// joga. Quem precisa dar mais detalhe usa Wrap, que mantém o sentinela para errors.Is.
var (
	// ErrUnknownKind: o tipo de efeito não existe no catálogo.
	ErrUnknownKind = Sentence("Esse efeito não existe.")
	// ErrInvalidParams: parâmetro ausente, estranho ao tipo ou fora da faixa.
	ErrInvalidParams = Sentence("Os dados do efeito não estão certos.")
	// ErrAlreadyApplied: o personagem já tem um efeito com a mesma identidade.
	ErrAlreadyApplied = Sentence("O personagem já tem esse efeito.")
	// ErrInvalidAudience: lista de quem vê malformada (todos + jogadores, repetido ou uuid nulo).
	ErrInvalidAudience = Sentence("A lista de quem vê o efeito não está certa.")
	// ErrNotFound: nenhum efeito com esse id.
	ErrNotFound = Sentence("Esse efeito não está mais no personagem.")
)

// Sentence cria um sentinela cujo texto é uma frase (com ponto final, que errors.New avisa).
func Sentence(msg string) error { return &detailed{msg: msg} }

// detailed é um sentinela com a frase completa que o mestre lê no lugar do texto genérico.
// Error() é só a frase (sem "sentinela: detalhe"); Unwrap devolve o sentinela para errors.Is.
type detailed struct {
	msg      string
	sentinel error
}

func (e *detailed) Error() string { return e.msg }
func (e *detailed) Unwrap() error { return e.sentinel }

// Wrap devolve um erro que lê como a frase msg e continua sendo o sentinela para errors.Is.
func Wrap(sentinel error, msg string) error {
	return &detailed{msg: msg, sentinel: sentinel}
}

func wrapAlready(id Identity) error {
	return Wrap(ErrAlreadyApplied, "O personagem já está "+id.Kind.StatePhrase()+".")
}

func wrapNotFound() error {
	return ErrNotFound
}
