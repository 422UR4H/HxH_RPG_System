package action

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

type Move struct {
	Category enum.MoveCategory
	// From is the actor's OWN PIECE POSITION on the server's board at enqueue time — never
	// the client's (B6, spec §4.3 "B5, B6 e B10"). room.go's enqueue_action arm derives it
	// from r.pieceSlotOf and writes it here; buildAction never sets it, and whatever the
	// client sent under "from" is parsed and discarded. nil means the actor has no piece on
	// the board, so there was nothing to check and nothing to report.
	//
	// Coordinate convention: [a, b, z], where (a, b) is (col, row) on a square grid or (q, r)
	// axial on a hex grid, and z is not read by the server — applyMove preserves the piece's
	// own virtual height instead (see its own doc).
	//
	// Num movimento empilhado (StackedOn != uuid.Nil) From é o destino do movimento anterior da
	// cadeia, não a casa da peça (D3, spec 2026-11-06) — MatchSession.EnqueueAction o reescreve.
	From       *[3]int
	Position   [3]int // [a, b, z] — same convention as From
	Speed      *RollCheck
	Charge     *RollCheck
	FinalSpeed int
	// StackedOn é o id do movimento pendente do mesmo personagem sobre o qual este se empilhou
	// (o fim da cadeia quando ele chegou); uuid.Nil = não empilhado. Enquanto esse anterior está
	// na fila, este está bloqueado (Action.IsBlockedIn): não abre, não é puxado e não é o que uma
	// reação consome. Só movimentos se empilham (puros ou combinados). Spec 2026-11-06 D3–D6.
	//
	// omitzero, não omitempty: uuid.UUID é um array, que omitempty nunca omite — a linha gravada
	// (actioncodec) de um movimento não empilhado fica sem a chave.
	StackedOn uuid.UUID `json:"stackedOn,omitzero"`
}
