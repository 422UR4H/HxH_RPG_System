package charactersheet

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterSheetNotFound        = domain.NewValidationError(errtext.Sentence("Ficha não encontrada."))
	ErrCharacterClassNotFound        = domain.NewValidationError(errtext.Sentence("Classe não encontrada."))
	ErrNicknameNotAllowed            = domain.NewValidationError(errtext.Sentence("Esse apelido não é permitido."))
	ErrNicknameAlreadyExists         = domain.NewValidationError(errtext.Sentence("Esse apelido já está em uso."))
	ErrNenHexNotInitialized          = domain.NewValidationError(errtext.Sentence("O hexágono de Nen ainda não foi iniciado."))
	ErrMaxCharacterSheetsLimit       = domain.NewValidationError(errtext.Sentence("Você não pode ter mais de 10 fichas de personagem."))
	ErrNotCharacterSheetOwner        = domain.NewValidationError(errtext.Sentence("Essa ficha não é sua."))
	ErrCharacterSheetHasNoOwner      = domain.NewDomainError(errtext.Sentence("A ficha não tem dono."))
	ErrInvalidUpdateHexValMethod     = domain.NewDomainError(errtext.Sentence("Jeito inválido de mudar o hexágono de Nen."))
	ErrCharacterSheetNotFreeToManage = domain.NewValidationError(errtext.Sentence("Essa ficha não pode ser gerenciada agora."))
)

func NewCharacterClassNotFoundError(className string) error {
	return errtext.Wrap(ErrCharacterClassNotFound, "A classe "+className+" não foi encontrada.")
}

func NewNicknameNotAllowedError(nickname string) error {
	return errtext.Wrap(ErrNicknameNotAllowed, "O apelido "+nickname+" não é permitido.")
}

func NewNicknameAlreadyExistsError(nickname string) error {
	return errtext.Wrap(ErrNicknameAlreadyExists, "O apelido "+nickname+" já está em uso.")
}
