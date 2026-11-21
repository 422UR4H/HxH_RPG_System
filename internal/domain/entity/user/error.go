package user

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMissingNick       = domain.NewValidationError(errtext.Sentence("Informe o apelido."))
	ErrInvalidNickLength = domain.NewValidationError(errtext.Sentence("O apelido deve ter entre 3 e 20 caracteres."))
	ErrNickHasEmoji      = domain.NewValidationError(errtext.Sentence("O apelido não pode ter emoji."))
	ErrNickAlreadyExists = domain.NewValidationError(errtext.Sentence("Esse apelido já está em uso."))

	ErrInvalidEmailFormat = domain.NewValidationError(errtext.Sentence("E-mail inválido."))
	ErrMissingEmail       = domain.NewValidationError(errtext.Sentence("Informe o e-mail."))
	ErrInvalidEmailLength = domain.NewValidationError(errtext.Sentence("O e-mail deve ter entre 12 e 64 caracteres."))
	ErrEmailHasEmoji      = domain.NewValidationError(errtext.Sentence("O e-mail não pode ter emoji."))
	ErrEmailAlreadyExists = domain.NewValidationError(errtext.Sentence("Já existe uma conta com esse e-mail."))

	ErrMissingPassword   = domain.NewValidationError(errtext.Sentence("Informe a senha."))
	ErrMismatchPassword  = domain.NewValidationError(errtext.Sentence("As senhas não são iguais."))
	ErrPasswordMinLenght = domain.NewValidationError(errtext.Sentence("A senha deve ter pelo menos 8 caracteres."))
	ErrPasswordMaxLenght = domain.NewValidationError(errtext.Sentence("A senha deve ter no máximo 64 caracteres."))

	ErrMissingConfirmPass = domain.NewValidationError(errtext.Sentence("Confirme a senha."))

	ErrUserNotFound = domain.NewValidationError(errtext.Sentence("Usuário não encontrado."))
)
