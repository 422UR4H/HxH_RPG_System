package game

import (
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// seesCharacterLocked é a resposta ÚNICA a "este jogador vê este personagem agora" (decisão do
// dono 4): o dono sempre vê o seu; um personagem Escondido não é visto por quem está fora da
// lista dele (decisão do dono 6), com ou sem tabuleiro; partida sem tabuleiro não tem névoa,
// todos veem; com tabuleiro, só se o personagem tem peça, a peça não está escondida para pid
// (pieceHiddenForLocked) e o centro dela cai dentro da visão do jogador. Personagem sem peça
// num tabuleiro não é visto pelos outros. O mestre não passa por aqui (vê tudo).
//
// O chamador segura r.mu (leitura basta).
func (r *Room) seesCharacterLocked(pid uuid.UUID, charID uuid.UUID) bool {
	if r.session != nil && r.session.GetCharToPlayer()[charID.String()] == pid {
		return true
	}
	if r.hiddenFromLocked(pid, charID) {
		return false
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
	if r.pieceHiddenForLocked(pid, piece) {
		return false
	}
	a, b := slotToAB(piece.Slot)
	x, y := mapservice.SlotCenterToLocal(a, b, r.session.GetGrid())
	return domainservice.IsVisible(domainservice.Point2D{X: x, Y: y}, r.session.GetVisibility(pid))
}

// hiddenFromLocked diz se charID está Escondido para pid: tem um efeito hidden e pid não está
// na lista de quem o vê (statuseffect.Listed, com o mesmo Subject da projeção —
// statusSubjectLocked). O mestre e o dono nunca: o mestre vê tudo, o dono sabe onde está.
//
// O chamador segura r.mu (leitura basta).
func (r *Room) hiddenFromLocked(pid uuid.UUID, charID uuid.UUID) bool {
	if r.session == nil || pid == r.masterUUID {
		return false
	}
	if r.session.GetCharToPlayer()[charID.String()] == pid {
		return false
	}
	var subject statuseffect.Subject
	var pcPublic, resolved bool
	viewer := statuseffect.Viewer{UserID: pid}
	for _, e := range r.session.StatusEffectsOf(charID) {
		if e.Kind != statuseffect.KindHidden {
			continue
		}
		if !resolved {
			subject, pcPublic = r.statusSubjectLocked(charID)
			resolved = true
		}
		if !statuseffect.Listed(e, viewer, subject, pcPublic) {
			return true
		}
	}
	return false
}

// pieceHiddenForLocked é o predicado de toda porta da névoa: "esta peça está escondida para
// pid". Dois motivos à parte: a peça está com visible:false (global, gravado com o tabuleiro)
// ou o personagem dela está Escondido para pid (por jogador, nunca escrito em Visible). O
// mestre só cai no primeiro — e só onde a porta já o aplicava a ele.
//
// O chamador segura r.mu (leitura basta).
func (r *Room) pieceHiddenForLocked(pid uuid.UUID, p PieceMovedPayload) bool {
	if p.Visible != nil && !*p.Visible {
		return true
	}
	charID, err := uuid.Parse(p.CharacterID)
	if err != nil {
		return false
	}
	return r.hiddenFromLocked(pid, charID)
}

// seesPieceLocked é "pid tem esta peça na tela agora", a mesma conta do map_full_state
// (FilterMapState): não escondida para ele e, então, dele ou dentro da visão. Usada para saber
// quem ganha ou perde a peça quando um Escondido muda.
//
// O chamador segura r.mu (leitura basta) e r.session não é nil.
func (r *Room) seesPieceLocked(pid uuid.UUID, p PieceMovedPayload) bool {
	if r.pieceHiddenForLocked(pid, p) {
		return false
	}
	if owner, ok := r.session.GetCharToPlayer()[p.CharacterID]; ok && owner == pid {
		return true
	}
	x, y := slotPayloadToLocal(p.Slot, r.session.GetGrid())
	return domainservice.IsVisible(domainservice.Point2D{X: x, Y: y}, r.session.GetVisibility(pid))
}

// syncSight reacerta, depois de tudo o que pode mudar a visão de alguém (uma peça andou ou
// saiu, uma parede mudou, um NPC entrou, um Escondido mudou), as barras e os status de cada
// um: broadcastBars (cada jogador só recebe se a vista dele mudou) e syncStatusSight.
//
// O chamador NÃO segura nenhuma trava.
func (r *Room) syncSight() {
	r.mu.RLock()
	session := r.session
	r.mu.RUnlock()
	if session == nil {
		return
	}
	r.broadcastBars(session)
	r.syncStatusSight()
}

// syncStatusSight é a metade de status do syncSight: todo personagem com efeitos, cada um
// comparado com o cache de quem o vê (quem não o vê agora não recebe nada — decisão do dono 4).
//
// O chamador NÃO segura nenhuma trava.
func (r *Room) syncStatusSight() {
	r.mu.RLock()
	var chars []uuid.UUID
	if r.session != nil {
		chars = r.session.CharactersWithStatusEffects()
	}
	r.mu.RUnlock()
	if len(chars) > 0 {
		r.syncStatusViews(chars)
	}
}

// pieceSightLocked tira o retrato de quem tem cada peça de personagem na tela: por jogador
// conectado (o mestre fora), o conjunto de peças que ele vê. É o "antes" e o "depois" de uma
// mudança de Escondido.
//
// O chamador segura r.mu (leitura basta).
func (r *Room) pieceSightLocked() map[uuid.UUID]map[string]bool {
	if r.session == nil || !r.hasBoardLocked() {
		return nil
	}
	out := make(map[uuid.UUID]map[string]bool, len(r.clients))
	for pid := range r.clients {
		if pid == r.masterUUID {
			continue
		}
		seen := make(map[string]bool)
		for id, p := range r.pieces {
			if p.CharacterID != "" && r.seesPieceLocked(pid, p) {
				seen[id] = true
			}
		}
		out[pid] = seen
	}
	return out
}

// relayHiddenChange manda a quem perdeu uma peça (o Escondido entrou ou a lista o tirou)
// piece_removed, e a quem a ganhou (o Escondido saiu ou a lista o pôs) piece_moved com a peça
// como está agora — mensagem do servidor. before/after são de pieceSightLocked.
//
// O chamador NÃO segura nenhuma trava.
func (r *Room) relayHiddenChange(before, after map[uuid.UUID]map[string]bool) {
	if before == nil || after == nil {
		return
	}
	type send struct {
		c    *Client
		msgs []Message
	}
	r.mu.RLock()
	var out []send
	for pid, now := range after {
		c, ok := r.clients[pid]
		if !ok {
			continue
		}
		was := before[pid]
		var msgs []Message
		for id := range was {
			if !now[id] {
				msgs = append(msgs, NewServerMessage(MsgTypePieceRemoved, PieceRemovedPayload{PieceID: id}))
			}
		}
		for id := range now {
			if !was[id] {
				if p, ok := r.pieces[id]; ok {
					msgs = append(msgs, NewServerMessage(MsgTypePieceMoved, p))
				}
			}
		}
		if len(msgs) > 0 {
			out = append(out, send{c: c, msgs: msgs})
		}
	}
	r.mu.RUnlock()
	for _, s := range out {
		for _, m := range s.msgs {
			s.c.SendMessage(m)
		}
	}
}
