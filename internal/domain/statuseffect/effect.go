package statuseffect

import (
	"time"

	"github.com/google/uuid"
)

// Params são os parâmetros normalizados de um efeito.
type Params struct {
	Domains      []Domain
	Injuries     map[Domain]InjuryLevel
	Level        SenseLevel
	Faces        []TouchFace
	Points       int
	HPPerRound   int
	Penalty      int
	Disadvantage bool
	Conduct      string
	Irresistible bool
	ControllerID *uuid.UUID
}

// ParamsInput é o que o mestre envia; ponteiro nil = não informado.
type ParamsInput struct {
	Domains      []Domain
	Injuries     map[Domain]InjuryLevel
	Level        SenseLevel
	Faces        []TouchFace
	Points       *int
	HPPerRound   *int
	Penalty      *int
	Disadvantage *bool
	Conduct      string
	Irresistible *bool
	ControllerID *uuid.UUID
}

// Identity distingue efeitos de um mesmo personagem: tipo + variante.
type Identity struct {
	Kind    Kind
	Variant string // Sense | Aptitude | ""
}

// Effect é um efeito aplicado a um personagem.
type Effect struct {
	ID, CharacterID uuid.UUID
	Kind            Kind
	Sense           Sense
	Aptitude        Aptitude
	Params          Params
	Alias           string
	Audience        *Audience // nil = padrão do tipo
	AppliedAt       time.Time
	UpdatedAt       time.Time
}

func (e Effect) Identity() Identity {
	id := Identity{Kind: e.Kind}
	switch e.Kind {
	case KindSense:
		id.Variant = string(e.Sense)
	case KindWeakened:
		id.Variant = string(e.Aptitude)
	}
	return id
}

// Clone faz cópia profunda (slices, mapa e ponteiros).
func (e Effect) Clone() Effect {
	c := e
	c.Params.Domains = append([]Domain(nil), e.Params.Domains...)
	c.Params.Faces = append([]TouchFace(nil), e.Params.Faces...)
	if e.Params.Injuries != nil {
		c.Params.Injuries = make(map[Domain]InjuryLevel, len(e.Params.Injuries))
		for d, l := range e.Params.Injuries {
			c.Params.Injuries[d] = l
		}
	}
	if e.Params.ControllerID != nil {
		id := *e.Params.ControllerID
		c.Params.ControllerID = &id
	}
	c.Audience = cloneAudience(e.Audience)
	return c
}

func cloneAudience(a *Audience) *Audience {
	if a == nil {
		return nil
	}
	c := *a
	if a.Players != nil {
		c.Players = append([]uuid.UUID{}, a.Players...)
	}
	return &c
}

// Write é uma gravação (inclusão/edição ou remoção) de um efeito.
type Write struct {
	Delete bool
	Effect Effect
	// Event é o registro desta mudança no histórico, gravado na mesma operação; nil = nada a
	// registrar.
	Event *Event
}

// Draft é o pedido de aplicação de um efeito.
type Draft struct {
	Kind     Kind
	Sense    Sense
	Aptitude Aptitude
	Params   ParamsInput
	Alias    string
	Audience *Audience
}

// Patch é a edição de um efeito existente.
type Patch struct {
	Params      *ParamsInput // nil = mantém
	Alias       *string      // nil = mantém; "" = limpa
	SetAudience bool         // true = troca a audiência por Audience (nil = padrão)
	Audience    *Audience
}
