package game

import (
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// seesCharacterLocked é a resposta ÚNICA a "este jogador vê este personagem agora" (decisão do
// dono 4): o dono sempre vê o seu; partida sem tabuleiro não tem névoa, todos veem; com
// tabuleiro, só se o personagem tem peça, a peça não está com visible:false e o centro dela
// cai dentro da visão do jogador. Personagem sem peça num tabuleiro não é visto pelos outros.
// O mestre não passa por aqui (vê tudo). A Task 10 acrescenta o Escondido.
//
// O chamador segura r.mu (leitura basta).
func (r *Room) seesCharacterLocked(pid uuid.UUID, charID uuid.UUID) bool {
	if r.session != nil && r.session.GetCharToPlayer()[charID.String()] == pid {
		return true
	}
	if !r.hasBoardLocked() {
		return true
	}
	if r.session == nil {
		return false
	}
	pieceID := r.pieceOfLocked(charID.String())
	if pieceID == "" {
		return false
	}
	piece := r.pieces[pieceID]
	if piece.Visible != nil && !*piece.Visible {
		return false
	}
	a, b := slotToAB(piece.Slot)
	x, y := mapservice.SlotCenterToLocal(a, b, r.session.GetGrid())
	return domainservice.IsVisible(domainservice.Point2D{X: x, Y: y}, r.session.GetVisibility(pid))
}
