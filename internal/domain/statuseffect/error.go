package statuseffect

import "github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

// Os textos destes erros chegam ao mestre na tela: são frases em português, escritas para quem
// joga. Quem precisa dar mais detalhe usa errtext.Wrap, que mantém o sentinela para errors.Is.
var (
	// ErrUnknownKind: o tipo de efeito não existe no catálogo.
	ErrUnknownKind = errtext.Sentence("Esse efeito não existe.")
	// ErrInvalidParams: parâmetro ausente, estranho ao tipo ou fora da faixa.
	ErrInvalidParams = errtext.Sentence("Os dados do efeito não estão certos.")
	// ErrAlreadyApplied: o personagem já tem um efeito com a mesma identidade.
	ErrAlreadyApplied = errtext.Sentence("O personagem já tem esse efeito.")
	// ErrInvalidAudience: lista de quem vê malformada (todos + jogadores, repetido ou uuid nulo).
	ErrInvalidAudience = errtext.Sentence("A lista de quem vê o efeito não está certa.")
	// ErrNotFound: nenhum efeito com esse id.
	ErrNotFound = errtext.Sentence("Esse efeito não está mais no personagem.")
)

func wrapAlready(id Identity) error {
	return errtext.Wrap(ErrAlreadyApplied, "O personagem já está "+id.StatePhrase()+".")
}

func wrapNotFound() error {
	return ErrNotFound
}
