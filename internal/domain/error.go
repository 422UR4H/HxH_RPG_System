package domain

import (
	"errors"
)

// As categorias abaixo só classificam o erro (errors.Is); nunca são lidas na tela: o texto que
// chega lá é o da frase do erro de dentro, sem prefixo técnico. Por isso ficam em inglês e fora
// do errtext (errtext.Of não as toma por frase).
var (
	ErrValidation = errors.New("validation error")
	ErrDomain     = errors.New("domain base error")
	ErrDB         = errors.New("database error")
)

// categorized lê como o erro de dentro e continua sendo da categoria (e do erro de dentro) para errors.Is.
type categorized struct {
	kind  error
	inner error
}

func (e *categorized) Error() string   { return e.inner.Error() }
func (e *categorized) Unwrap() []error { return []error{e.inner, e.kind} }

func NewValidationError(err error) error { return &categorized{kind: ErrValidation, inner: err} }

func NewDomainError(err error) error { return &categorized{kind: ErrDomain, inner: err} }

func NewDBError(err error) error { return &categorized{kind: ErrDB, inner: err} }
