package statuseffect

import (
	"errors"
	"fmt"
)

var (
	// ErrUnknownKind: o tipo de efeito não existe no catálogo.
	ErrUnknownKind = errors.New("unknown status effect kind")
	// ErrInvalidParams: parâmetro ausente, estranho ao tipo ou fora da faixa.
	ErrInvalidParams = errors.New("invalid status effect params")
	// ErrAlreadyApplied: o personagem já tem um efeito com a mesma identidade.
	ErrAlreadyApplied = errors.New("status effect already applied")
	// ErrNotFound: nenhum efeito com esse id.
	ErrNotFound = errors.New("status effect not found")
)

func wrapAlready(id Identity) error {
	if id.Variant == "" {
		return fmt.Errorf("%w: %s", ErrAlreadyApplied, id.Kind)
	}
	return fmt.Errorf("%w: %s/%s", ErrAlreadyApplied, id.Kind, id.Variant)
}

func wrapNotFound() error {
	return fmt.Errorf("%w: id", ErrNotFound)
}
