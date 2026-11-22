package campaign

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCampaignNotFound            = domain.NewValidationError(errtext.Sentence("Campanha não encontrada."))
	ErrNotCampaignOwner            = domain.NewValidationError(errtext.Sentence("Essa campanha não é sua."))
	ErrMinNameLength               = domain.NewValidationError(errtext.Sentence("O nome deve ter pelo menos 5 caracteres."))
	ErrMaxNameLength               = domain.NewValidationError(errtext.Sentence("O nome deve ter no máximo 32 caracteres."))
	ErrInvalidStartDate            = domain.NewValidationError(errtext.Sentence("Informe a data de início da história."))
	ErrMaxCampaignsLimit           = domain.NewValidationError(errtext.Sentence("Você não pode ter mais de 10 campanhas."))
	ErrMaxBriefDescLength          = domain.NewValidationError(errtext.Sentence("A descrição curta deve ter no máximo 255 caracteres."))
	ErrMaxCallLinkLength           = domain.NewValidationError(errtext.Sentence("O link da chamada deve ter no máximo 255 caracteres."))
	ErrCampaignHasStartedMatch     = domain.NewValidationError(errtext.Sentence("A campanha tem uma partida que já começou."))
	ErrCampaignAlreadyEnded        = domain.NewValidationError(errtext.Sentence("A campanha já terminou."))
	ErrLockedAfterMatchStart       = domain.NewValidationError(errtext.Sentence("O nome e a data de início da história não mudam depois que uma partida começa."))
	ErrCannotRegressStoryCurrentAt = domain.NewValidationError(errtext.Sentence("A data atual da história não pode voltar para antes da data de agora."))
	ErrStoryCurrentBeforeStart     = domain.NewValidationError(errtext.Sentence("A data atual da história não pode ser anterior à data de início."))
)
