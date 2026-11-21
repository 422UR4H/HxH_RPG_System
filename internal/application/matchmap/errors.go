package matchmapuc

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

var (
	ErrMatchMapNotFound    = errtext.Sentence("Esta partida não tem mapa.")
	ErrMatchAlreadyStarted = errtext.Sentence("Não dá para trocar o mapa depois que a partida começou.")
	ErrNotMatchMaster      = errtext.Sentence("Só o mestre da partida pode fazer isso.")
	ErrMapNotFound         = errtext.Sentence("Mapa não encontrado.")
	ErrMatchNotFound       = errtext.Sentence("Partida não encontrada.")

	// New for B16 (spec §4.3): a match can inherit the board of another match of the SAME
	// campaign, currently on the SAME map, that already has a board of its own.
	ErrSourceMatchNotInCampaign = errtext.Sentence("A partida de origem é de outra campanha.")
	ErrSourceMatchOnAnotherMap  = errtext.Sentence("O tabuleiro da partida de origem está em outro mapa.")
	ErrSourceMatchHasNoBoard    = errtext.Sentence("A partida de origem não tem tabuleiro para herdar.")
	// ErrSourceMatchIsTheSameMatch guards against inheritBoardFromMatchUuid == the match
	// being attached to. Without this check, Copy(dst, dst) would delete the very row its
	// own INSERT…SELECT reads from and fail with a non-sentinel error (500) instead of a
	// clean 422 — fix round 1 (review finding).
	ErrSourceMatchIsTheSameMatch = errtext.Sentence("A partida de origem não pode ser a própria partida.")
)
