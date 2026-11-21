package domain

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

// As categorias abaixo só classificam o erro (errors.Is); o texto que chega à tela é o da frase
// do erro de dentro, sem prefixo técnico.
var (
	ErrValidation = errtext.Sentence("Os dados enviados não estão certos.")
	ErrDomain     = errtext.Sentence("Isso não é permitido pelas regras.")
	ErrDB         = errtext.Sentence("Não foi possível acessar os dados agora.")
)

// categorized lê como o erro de dentro e continua sendo da categoria (e do erro de dentro) para errors.Is.
type categorized struct {
	kind  error
	inner error
}

func (e *categorized) Error() string   { return e.inner.Error() }
func (e *categorized) Unwrap() []error { return []error{e.kind, e.inner} }

func NewValidationError(err error) error { return &categorized{kind: ErrValidation, inner: err} }

func NewDomainError(err error) error { return &categorized{kind: ErrDomain, inner: err} }

func NewDBError(err error) error { return &categorized{kind: ErrDB, inner: err} }
