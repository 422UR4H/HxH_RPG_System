package statuseffect

import (
	"fmt"

	"github.com/google/uuid"
)

// Audience é a lista de quem vê um efeito. Everyone = todos; Players vazio (nil ou não) =
// só o mestre; já um *Audience nil no Effect é o padrão do tipo. O dono não entra implicitamente.
type Audience struct {
	Everyone bool
	Players  []uuid.UUID
}

// Viewer é quem olha: o usuário e se ele é o mestre da mesa.
type Viewer struct {
	UserID   uuid.UUID
	IsMaster bool
}

// Subject é o personagem que carrega o efeito.
type Subject struct {
	OwnerID uuid.UUID
	IsNPC   bool
}

// Validate recusa Everyone junto de jogadores, jogador repetido e uuid.Nil.
// Lista vazia é válida: significa só o mestre.
func (a Audience) Validate() error {
	if a.Everyone && len(a.Players) > 0 {
		return fmt.Errorf("%w: everyone com jogadores", ErrInvalidAudience)
	}
	seen := make(map[uuid.UUID]struct{}, len(a.Players))
	for _, p := range a.Players {
		if p == uuid.Nil {
			return fmt.Errorf("%w: jogador inválido", ErrInvalidAudience)
		}
		if _, dup := seen[p]; dup {
			return fmt.Errorf("%w: jogador repetido", ErrInvalidAudience)
		}
		seen[p] = struct{}{}
	}
	return nil
}

// Listed diz se a lista de quem vê (ou o padrão do tipo) inclui este
// espectador. A visão (névoa) NÃO entra aqui: a sala a soma depois.
func Listed(e Effect, v Viewer, s Subject, pcPublicByDefault bool) bool {
	if v.IsMaster {
		return true
	}
	if e.Audience != nil {
		if e.Audience.Everyone {
			return true
		}
		for _, p := range e.Audience.Players {
			if p == v.UserID {
				return true
			}
		}
		return false
	}
	switch {
	case e.Kind.PublicByDefault():
		return true
	case s.IsNPC:
		return false
	case pcPublicByDefault:
		return true
	}
	return v.UserID == s.OwnerID
}
