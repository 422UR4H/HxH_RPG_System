package sheet

import (
	"errors"
	"fmt"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

const invalidOwnerMsg = "A ficha deve ser de um jogador ou de um cenário, nunca dos dois."
const invalidAlignment = "O alinhamento deve combinar Ordeiro, Neutro ou Caótico com Bom, Neutro ou Mau."

var (
	ErrClassNotApplied            = domain.NewDomainError(errtext.Sentence("A classe ainda não foi aplicada à ficha."))
	ErrInvalidOwner               = domain.NewValidationError(errtext.Sentence(invalidOwnerMsg))
	ErrInvalidAlignment           = domain.NewValidationError(errtext.Sentence(invalidAlignment))
	ErrInvalidActiveCategoryCount = domain.NewValidationError(errtext.Sentence("Deixe pelo menos uma categoria ativa."))
	ErrInvalidNicknameLength      = domain.NewValidationError(errtext.Sentence("O apelido deve ter entre 3 e 10 caracteres."))
	ErrInvalidFullNameLength      = domain.NewValidationError(errtext.Sentence("O nome completo deve ter entre 6 e 32 caracteres."))
	ErrNicknameHasEmoji           = domain.NewValidationError(errtext.Sentence("O apelido não pode ter emoji."))
	ErrFullNameHasEmoji           = domain.NewValidationError(errtext.Sentence("O nome completo não pode ter emoji."))
	ErrInvalidBriefDescription    = domain.NewValidationError(errtext.Sentence("A descrição curta deve ter no máximo 255 caracteres."))
	ErrInvalidBirthday            = domain.NewValidationError(errtext.Sentence("Data de nascimento inválida."))
	ErrInvalidAge                 = domain.NewValidationError(errtext.Sentence("A idade deve ser maior que zero."))
	ErrInvalidDistributionPoints  = domain.NewValidationError(errtext.Sentence("Os pontos distribuídos não estão certos."))
	ErrCharClassAlreadyExists     = domain.NewValidationError(errtext.Sentence("A ficha já tem uma classe."))

	ErrFailedToUpgradeStatus          = domain.NewDomainError(errors.New("failed to upgrade status"))
	ErrFailedToBuildHatsu             = domain.NewDomainError(errors.New("failed to build hatsu"))
	ErrFailedToInitHatsu              = domain.NewDomainError(errors.New("failed to init hatsu"))
	ErrFailedToSetStatus              = domain.NewDomainError(errors.New("failed to set status"))
	ErrFailedToGetStatus              = domain.NewDomainError(errors.New("failed to get status"))
	ErrFailedToGetPhysSkillExpRef     = domain.NewDomainError(errors.New("failed to get physical skills experience reference"))
	ErrFailedToIncreasePhysAttrPts    = domain.NewDomainError(errors.New("failed to increase physical primary attribute points"))
	ErrFailedToIncreaseMentalExp      = domain.NewDomainError(errors.New("failed to increase mental exp"))
	ErrFailedToIncreaseSkillExp       = domain.NewDomainError(errors.New("failed to increase skill exp"))
	ErrFailedToIncreasePrincipleExp   = domain.NewDomainError(errors.New("failed to increase principle exp"))
	ErrFailedToIncreaseCategoryExp    = domain.NewDomainError(errors.New("failed to increase category exp"))
	ErrFailedToIncreaseProficiencyExp = domain.NewDomainError(errors.New("failed to increase proficiency exp"))
	ErrFailedToIncreaseJointProfExp   = domain.NewDomainError(errors.New("failed to increase joint proficiency exp"))
	ErrFailedToAddJointSkill          = domain.NewDomainError(errors.New("failed to add joint skill"))
	ErrFailedToAddJointProficiency    = domain.NewDomainError(errors.New("failed to add joint proficiency"))
	ErrFailedToAddCommonProficiency   = domain.NewDomainError(errors.New("failed to add common proficiency"))
)

func NewClassNotAppliedError(msg string) error {
	return fmt.Errorf("%w: %s", ErrClassNotApplied, msg)
}

func NewInvalidNicknameLengthError(nick string) error {
	return errtext.Wrap(ErrInvalidNicknameLength, "O apelido deve ter entre 3 e 10 caracteres.")
}

func NewInvalidFullNameLengthError(name string) error {
	return errtext.Wrap(ErrInvalidFullNameLength, "O nome completo deve ter entre 6 e 32 caracteres.")
}

func NewNicknameHasEmojiError(nick string) error {
	return errtext.Wrap(ErrNicknameHasEmoji, "O apelido não pode ter emoji.")
}

func NewFullNameHasEmojiError(name string) error {
	return errtext.Wrap(ErrFullNameHasEmoji, "O nome completo não pode ter emoji.")
}

func NewInvalidAlignmentError(alignment string) error {
	return ErrInvalidAlignment
}

func NewInvalidBriefDescriptionError(desc string) error {
	return ErrInvalidBriefDescription
}

func NewInvalidBirthdayError() error {
	return ErrInvalidBirthday
}

func NewInvalidAgeError() error {
	return ErrInvalidAge
}
