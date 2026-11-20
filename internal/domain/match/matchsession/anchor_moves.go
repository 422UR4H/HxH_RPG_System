package matchsession

import "github.com/google/uuid"

// AnchorMove é uma reação que saiu do turno aberto e para onde vai o que estava preso a ela
// (as mudanças de status ancoradas, Task 15b): a reação do personagem que substituiu a sentada
// (ReplaceSeated), a sentada nova que um status pôs no lugar (reseatForStatus), ou a action do
// turno quando a sentada saiu sem substituta (Unseat — o mestre trocou os alvos).
type AnchorMove struct {
	TurnID, From, To uuid.UUID
}

// noteAnchorMove registra uma reação que saiu do turno e o que fica no lugar dela.
func (s *MatchSession) noteAnchorMove(turnID, from, to uuid.UUID) {
	s.anchorMoves = append(s.anchorMoves, AnchorMove{TurnID: turnID, From: from, To: to})
}

// TakeAnchorMoves entrega e esquece as saídas registradas, na ordem em que aconteceram. A sala
// as aplica às mudanças de status pendentes do turno (quem guarda as âncoras é ela).
func (s *MatchSession) TakeAnchorMoves() []AnchorMove {
	out := s.anchorMoves
	s.anchorMoves = nil
	return out
}
