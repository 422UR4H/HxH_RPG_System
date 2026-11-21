package statuseffect

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxAliasRunes = 60

func cloneAll(es []Effect) []Effect {
	out := make([]Effect, len(es))
	for i, e := range es {
		out[i] = e.Clone()
	}
	return out
}

func normAlias(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxAliasRunes {
		return "", invalid("alias")
	}
	return s, nil
}

// Apply aplica um efeito ao conjunto atual do personagem. Devolve o novo
// conjunto e o que foi acrescentado (Inconsciente traz Caído, se ainda não há).
// Nunca altera current; falha sem devolver conjunto.
func Apply(current []Effect, characterID uuid.UUID, d Draft, newID func() uuid.UUID, now time.Time) (next, added []Effect, err error) {
	params, err := Normalize(d.Kind, d.Sense, d.Aptitude, d.Params)
	if err != nil {
		return nil, nil, err
	}
	alias, err := normAlias(d.Alias)
	if err != nil {
		return nil, nil, err
	}
	if d.Audience != nil {
		if err := d.Audience.Validate(); err != nil {
			return nil, nil, err
		}
	}
	e := Effect{
		ID: newID(), CharacterID: characterID,
		Kind: d.Kind, Sense: d.Sense, Aptitude: d.Aptitude,
		Params: params, Alias: alias, Audience: cloneAudience(d.Audience),
		AppliedAt: now, UpdatedAt: now,
	}
	for _, c := range current {
		if c.Identity() == e.Identity() {
			return nil, nil, wrapAlready(e.Identity())
		}
	}
	added = []Effect{e}
	if d.Kind == KindUnconscious && !has(current, Identity{Kind: KindProne}) {
		// Inconsciente traz Caído, com parâmetros, apelido e audiência padrão.
		added = append(added, Effect{
			ID: newID(), CharacterID: characterID, Kind: KindProne,
			AppliedAt: now, UpdatedAt: now,
		})
	}
	next = append(cloneAll(current), cloneAll(added)...)
	return next, added, nil
}

func has(es []Effect, id Identity) bool {
	for _, e := range es {
		if e.Identity() == id {
			return true
		}
	}
	return false
}

// Edit troca os parâmetros (inteiros), o apelido e/ou a audiência de um efeito.
func Edit(current []Effect, id uuid.UUID, p Patch, now time.Time) (next []Effect, edited Effect, err error) {
	idx := indexOf(current, id)
	if idx < 0 {
		return nil, Effect{}, wrapNotFound()
	}
	e := current[idx].Clone()
	if p.Params != nil {
		params, err := Normalize(e.Kind, e.Sense, e.Aptitude, *p.Params)
		if err != nil {
			return nil, Effect{}, err
		}
		e.Params = params
	}
	if p.Alias != nil {
		alias, err := normAlias(*p.Alias)
		if err != nil {
			return nil, Effect{}, err
		}
		e.Alias = alias
	}
	if p.SetAudience {
		if p.Audience != nil {
			if err := p.Audience.Validate(); err != nil {
				return nil, Effect{}, err
			}
		}
		e.Audience = cloneAudience(p.Audience)
	}
	e.UpdatedAt = now
	next = cloneAll(current)
	next[idx] = e
	return next, e.Clone(), nil
}

// Remove tira um efeito pelo id. Caído e Inconsciente são independentes
// (decisão do dono 7): remover um não remove o outro.
func Remove(current []Effect, id uuid.UUID) (next []Effect, removed Effect, err error) {
	idx := indexOf(current, id)
	if idx < 0 {
		return nil, Effect{}, wrapNotFound()
	}
	removed = current[idx].Clone()
	next = cloneAll(current)
	next = append(next[:idx], next[idx+1:]...)
	return next, removed, nil
}

func indexOf(es []Effect, id uuid.UUID) int {
	for i, e := range es {
		if e.ID == id {
			return i
		}
	}
	return -1
}
