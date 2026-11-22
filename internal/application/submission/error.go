package submission

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterAlreadySubmitted  = domain.NewValidationError(errtext.Sentence("Essa ficha já foi enviada."))
	ErrMasterCannotSubmitOwnSheet = domain.NewValidationError(errtext.Sentence("O mestre não pode enviar a própria ficha."))
	ErrSubmissionNotFound         = domain.NewValidationError(errtext.Sentence("Envio da ficha não encontrado."))
	ErrNotCampaignMaster          = domain.NewValidationError(errtext.Sentence("Só o mestre da campanha pode fazer isso."))
	ErrNickAlreadyInCampaign      = domain.NewValidationError(errtext.Sentence("Já existe um personagem com esse apelido na campanha."))
)
